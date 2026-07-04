package scoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/chandrashekhartata/acgc/eval/datasets"
	"github.com/chandrashekhartata/acgc/eval/harness"
	"github.com/chandrashekhartata/acgc/internal/llm"
)

const judgeSystemPrompt = `You are a strict, neutral evaluator scoring responses from two AI assistants (labeled A and B) for the same question.

You will see:
- The original question
- Optional success criteria / rubric
- Response from assistant A
- Response from assistant B

You must score each response independently on a 0-5 scale, where:
  5 = fully correct, complete, follows all criteria
  4 = mostly correct, minor omissions
  3 = partially correct, some important gaps
  2 = mostly wrong but acknowledges the topic
  1 = wrong with weak relevance
  0 = entirely wrong or off-topic

Return STRICTLY valid JSON in this exact shape, no other text:
{
  "score_a": <0-5 number>,
  "score_b": <0-5 number>,
  "reason_a": "<one-sentence justification>",
  "reason_b": "<one-sentence justification>"
}`

// JudgeClient is a small wrapper around the LLM client configured for
// blinded paired comparison. It is intentionally a different "instance"
// from the eval pipelines so the judge can use a different model if desired.
type JudgeClient struct {
	client *llm.Client
	model  string
	rng    *rand.Rand
}

func NewJudgeClient(baseURL, apiKey, model string, seed int64) *JudgeClient {
	return &JudgeClient{
		client: llm.NewClient(llm.Config{BaseURL: baseURL, APIKey: apiKey, Model: model}),
		model:  model,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

// JudgePair scores both responses blinded. Order is randomized per call so
// position bias averages out across the eval.
func (j *JudgeClient) JudgePair(
	ctx context.Context,
	probe datasets.Probe,
	baseline, acgc harness.ProbeResult,
) (Score, Score, error) {
	// Randomize which response is A vs B.
	aIsBaseline := j.rng.Intn(2) == 0
	respA, respB := acgc.Response, baseline.Response
	if aIsBaseline {
		respA, respB = baseline.Response, acgc.Response
	}

	userMsg := buildJudgePrompt(probe, respA, respB)

	// Empty / non-JSON outputs from reasoning models are common when the small
	// MaxTokens budget is consumed by hidden reasoning. Try once, then retry
	// with a larger budget on parse failure before giving up.
	scoreA, reasonA, scoreB, reasonB, lastFinish, lastLen, lastRaw, err := j.judgeWithRetry(ctx, userMsg)
	if err != nil {
		return Score{}, Score{}, fmt.Errorf("parse judge: %w (finish=%q, content_len=%d, raw: %s)",
			err, lastFinish, lastLen, truncateMid(lastRaw, 200))
	}

	// Un-randomize back to baseline/acgc.
	baseScore, acgcScore := scoreB, scoreA
	baseReason, acgcReason := reasonB, reasonA
	if aIsBaseline {
		baseScore, acgcScore = scoreA, scoreB
		baseReason, acgcReason = reasonA, reasonB
	}

	bs := Score{
		ScenarioID: baseline.ScenarioID,
		ProbeID:    baseline.ProbeID,
		Pipeline:   baseline.Pipeline,
		Value:      baseScore,
		Method:     "judge",
		Detail:     baseReason,
	}
	as := Score{
		ScenarioID: acgc.ScenarioID,
		ProbeID:    acgc.ProbeID,
		Pipeline:   acgc.Pipeline,
		Value:      acgcScore,
		Method:     "judge",
		Detail:     acgcReason,
	}
	return bs, as, nil
}

// judgeMaxTokensFirst is the budget for the first judge call. Reasoning
// models (gpt-5, o1, o3) burn part of MaxTokens on hidden reasoning before
// the visible JSON body, so this needs to be roomy.
const (
	judgeMaxTokensFirst = 1500
	judgeMaxTokensRetry = 3000
)

// callJudge issues one judge generation. Surfaces FinishReason for diagnostics.
// Transient provider failures (429/5xx/timeouts) are retried with backoff.
func (j *JudgeClient) callJudge(ctx context.Context, userMsg string) (*llm.GenerateResult, error) {
	return harness.GenerateWithRetry(ctx, j.client, []llm.ChatMessage{
		{Role: "system", Content: judgeSystemPrompt},
		{Role: "user", Content: userMsg},
	}, 0, judgeMaxTokensFirst)
}

// callJudgeRetry issues a retry with a larger token budget — protects against
// "length"-finish failures on reasoning models that ate the first budget.
func (j *JudgeClient) callJudgeRetry(ctx context.Context, userMsg string) (*llm.GenerateResult, error) {
	return harness.GenerateWithRetry(ctx, j.client, []llm.ChatMessage{
		{Role: "system", Content: judgeSystemPrompt},
		{Role: "user", Content: userMsg},
	}, 0, judgeMaxTokensRetry)
}

// judgeWithRetry runs callJudge and parses; on call-failure or parse-failure
// it retries once with a larger budget, then returns the last failure context
// (finish_reason, content length, raw content) for clearer error messages.
func (j *JudgeClient) judgeWithRetry(ctx context.Context, userMsg string) (
	scoreA float64, reasonA string, scoreB float64, reasonB string,
	finishReason string, contentLen int, rawContent string, err error,
) {
	result, callErr := j.callJudge(ctx, userMsg)
	if callErr == nil {
		finishReason, rawContent = result.FinishReason, result.Content
		contentLen = len(result.Content)
		scoreA, reasonA, scoreB, reasonB, err = parseJudgeResponse(result.Content)
		if err == nil {
			return
		}
	} else {
		err = callErr
	}

	retry, retryErr := j.callJudgeRetry(ctx, userMsg)
	if retryErr != nil {
		// Surface whichever error gives more signal — prefer the call error.
		if err == nil {
			err = retryErr
		}
		return
	}
	finishReason, rawContent = retry.FinishReason, retry.Content
	contentLen = len(retry.Content)
	scoreA, reasonA, scoreB, reasonB, err = parseJudgeResponse(retry.Content)
	return
}

func buildJudgePrompt(probe datasets.Probe, respA, respB string) string {
	var b strings.Builder
	b.WriteString("Question:\n")
	b.WriteString(probe.Question)
	b.WriteString("\n\n")
	if probe.JudgeRubric != "" {
		b.WriteString("Success criteria / rubric:\n")
		b.WriteString(probe.JudgeRubric)
		b.WriteString("\n\n")
	} else if len(probe.ExpectedAny) > 0 {
		b.WriteString("Expected concepts to mention: ")
		b.WriteString(strings.Join(probe.ExpectedAny, ", "))
		b.WriteString("\n\n")
	}
	b.WriteString("--- Response A ---\n")
	b.WriteString(respA)
	b.WriteString("\n\n--- Response B ---\n")
	b.WriteString(respB)
	b.WriteString("\n\nNow score both responses. Return JSON only.")
	return b.String()
}

var jsonBlockRe = regexp.MustCompile(`(?s)\{.*\}`)

func parseJudgeResponse(s string) (scoreA float64, reasonA string, scoreB float64, reasonB string, err error) {
	// The model sometimes wraps JSON in ```json ... ```; strip that.
	cleaned := jsonBlockRe.FindString(s)
	if cleaned == "" {
		err = fmt.Errorf("no JSON object found")
		return
	}
	var parsed struct {
		ScoreA  json.Number `json:"score_a"`
		ScoreB  json.Number `json:"score_b"`
		ReasonA string      `json:"reason_a"`
		ReasonB string      `json:"reason_b"`
	}
	if err = json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return
	}
	scoreA, _ = parsed.ScoreA.Float64()
	scoreB, _ = parsed.ScoreB.Float64()
	scoreA = clamp(scoreA, 0, 5)
	scoreB = clamp(scoreB, 0, 5)
	reasonA = parsed.ReasonA
	reasonB = parsed.ReasonB
	return
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func truncateMid(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
