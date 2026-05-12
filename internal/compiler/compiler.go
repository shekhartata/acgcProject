package compiler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chandrashekhartata/acgc/internal/domain"
)

type Compiler struct {
	tokenBudget int
}

func NewCompiler(tokenBudget int) *Compiler {
	return &Compiler{tokenBudget: tokenBudget}
}

// Compile builds an optimized prompt from the state tree nodes.
// Nodes are selected by retention score until the token budget is exhausted.
// The prompt follows the PRD's assembly order:
//  1. System instructions
//  2. Current user request
//  3. Active goal
//  4. Non-negotiable constraints
//  5. Important decisions
//  6. Compressed context
//  7. Required tool outputs
//  8. Open issues
func (c *Compiler) Compile(
	sessionID, taskID, userMessage string,
	nodes []*domain.StateNode,
	systemPrompt string,
) *domain.CompiledPrompt {
	cp := &domain.CompiledPrompt{
		CompiledPromptID:   fmt.Sprintf("cp_%d", time.Now().UnixNano()),
		SessionID:          sessionID,
		TaskID:             taskID,
		CurrentUserMessage: userMessage,
		CreatedAt:          time.Now(),
	}

	// Estimate original cost: system + user + all active nodes
	originalTokens := estimateTokens(systemPrompt) + estimateTokens(userMessage)
	for _, n := range nodes {
		originalTokens += n.TokenCount
	}
	cp.OriginalTokenCount = originalTokens

	// Categorize nodes
	var goals, constraints, decisions, toolOutputs, compressed, issues []*domain.StateNode
	var rest []*domain.StateNode

	for _, n := range nodes {
		switch n.NodeType {
		case domain.NodeGoal:
			goals = append(goals, n)
		case domain.NodeConstraint:
			constraints = append(constraints, n)
		case domain.NodeDecision:
			decisions = append(decisions, n)
		case domain.NodeToolResult:
			toolOutputs = append(toolOutputs, n)
		case domain.NodeCompressedBranch, domain.NodeSummary:
			compressed = append(compressed, n)
		case domain.NodeIssue:
			issues = append(issues, n)
		default:
			rest = append(rest, n)
		}
	}

	// Sort each category by retention score (highest first)
	sortByScore(decisions)
	sortByScore(toolOutputs)
	sortByScore(compressed)
	sortByScore(rest)

	// Build prompt sections within token budget
	var sections []string
	tokensUsed := 0

	// Always include: system prompt + user message
	if systemPrompt != "" {
		sections = append(sections, systemPrompt)
		tokensUsed += estimateTokens(systemPrompt)
	}

	sections = append(sections, fmt.Sprintf("## Current Request\n%s", userMessage))
	tokensUsed += estimateTokens(userMessage)

	// Goals are always included
	if len(goals) > 0 {
		cp.ActiveGoal = goals[0].Title
		goalText := formatSection("Active Goals", goals)
		sections = append(sections, goalText)
		tokensUsed += estimateTokens(goalText)
	}

	// Constraints are always included
	if len(constraints) > 0 {
		for _, cn := range constraints {
			cp.ActiveConstraints = append(cp.ActiveConstraints, cn.Summary)
		}
		constText := formatSection("Constraints", constraints)
		sections = append(sections, constText)
		tokensUsed += estimateTokens(constText)
	}

	// Fill remaining budget by priority: decisions → compressed → tool outputs → issues → rest
	remaining := c.tokenBudget - tokensUsed
	priorityBuckets := [][]*domain.StateNode{decisions, compressed, toolOutputs, issues, rest}
	bucketLabels := []string{"Key Decisions", "Prior Context", "Tool Outputs", "Open Issues", "Additional Context"}

	for i, bucket := range priorityBuckets {
		if remaining <= 0 || len(bucket) == 0 {
			continue
		}

		selected, excluded, used := selectWithinBudget(bucket, remaining)
		if len(selected) > 0 {
			text := formatSection(bucketLabels[i], selected)
			sections = append(sections, text)
			tokensUsed += used
			remaining -= used

			switch bucketLabels[i] {
			case "Key Decisions":
				for _, s := range selected {
					cp.RelevantDecisions = append(cp.RelevantDecisions, s.Summary)
				}
			case "Tool Outputs":
				for _, s := range selected {
					cp.RelevantToolOutputs = append(cp.RelevantToolOutputs, s.Summary)
				}
			case "Prior Context":
				for _, s := range selected {
					cp.CompressedContext = append(cp.CompressedContext, s.Summary)
				}
			case "Open Issues":
				for _, s := range selected {
					cp.OpenIssues = append(cp.OpenIssues, s.Summary)
				}
			}

			for _, e := range excluded {
				cp.ExcludedNodeRefs = append(cp.ExcludedNodeRefs, e.NodeID)
			}
		}
	}

	cp.FinalPrompt = strings.Join(sections, "\n\n---\n\n")
	cp.CompiledTokenCount = estimateTokens(cp.FinalPrompt)

	return cp
}

func selectWithinBudget(nodes []*domain.StateNode, budget int) (selected, excluded []*domain.StateNode, tokensUsed int) {
	for _, n := range nodes {
		cost := n.TokenCount
		if cost == 0 {
			cost = estimateTokens(n.Summary)
		}
		if tokensUsed+cost <= budget {
			selected = append(selected, n)
			tokensUsed += cost
		} else {
			excluded = append(excluded, n)
		}
	}
	return
}

func formatSection(label string, nodes []*domain.StateNode) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## %s\n", label))
	for _, n := range nodes {
		if n.Summary != "" {
			b.WriteString(fmt.Sprintf("- %s\n", n.Summary))
		} else if n.Title != "" {
			b.WriteString(fmt.Sprintf("- %s\n", n.Title))
		}
	}
	return b.String()
}

func sortByScore(nodes []*domain.StateNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Scores.RetentionScore > nodes[j].Scores.RetentionScore
	})
}

func estimateTokens(s string) int {
	// ~4 chars per token is a reasonable approximation
	return len(s) / 4
}
