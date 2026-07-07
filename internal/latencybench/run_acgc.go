package latencybench

import (
	"context"
	"fmt"
	"time"

	pb "github.com/shekhartata/acgcProject/api/proto"
)

// WarmSessionGRPC replays scripted warm_pairs via CaptureEvent so the server's
// state tree matches the scripted conversation before timed Run probes.
func WarmSessionGRPC(
	ctx context.Context,
	client pb.ACGCServiceClient,
	sessionID, taskID string,
	pairs []WarmPairTurn,
	perCaptureDelay time.Duration,
) error {
	for _, p := range pairs {
		for _, step := range []struct {
			evtType string
			payload string
		}{
			{evtUserPrompt, p.User},
			{evtLLMResp, p.Assistant},
		} {
			_, err := client.CaptureEvent(ctx, &pb.CaptureEventRequest{
				SessionId: sessionID,
				TaskId:    taskID,
				EventType: step.evtType,
				Payload:   step.payload,
			})
			if err != nil {
				return fmt.Errorf("capture %s for session %s: %w", step.evtType, sessionID, err)
			}
			if perCaptureDelay > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(perCaptureDelay):
				}
			}
		}
	}
	return nil
}

const evtUserPrompt = "user_prompt"
const evtLLMResp = "llm_response"

// GRPCRunMeasured performs a Run RPC with optional llm credentials and conversation history metadata.
func GRPCRunMeasured(
	ctx context.Context,
	client pb.ACGCServiceClient,
	req *pb.RunRequest,
) (resp *pb.RunResponse, roundTrip time.Duration, err error) {
	start := time.Now()
	resp, err = client.Run(ctx, req)
	rt := time.Since(start)
	if err != nil {
		return nil, rt, err
	}
	return resp, rt, nil
}
