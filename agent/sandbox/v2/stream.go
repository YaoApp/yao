package sandboxv2

import (
	"context"
	"errors"
	"fmt"
	"time"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// ExecuteRequest consolidates all parameters for ExecuteSandboxStream.
type ExecuteRequest struct {
	Computer     infra.Computer
	Runner       types.Runner
	Config       *types.SandboxConfig
	StreamReq    *types.StreamRequest
	Manager      *infra.Manager
	LoadingMsgID string
}

// ExecuteSandboxStream runs runner.Stream and bridges agentContext interrupts.
//
// Cleanup (runner.Cleanup + LifecycleAction) is NOT performed here; the caller
// (agent.go sandboxCleanup closure) is responsible for all lifecycle management
// so that cleanup happens exactly once regardless of code path.
func ExecuteSandboxStream(
	ctx *agentContext.Context,
	req *ExecuteRequest,
	handler message.StreamFunc,
) (*agentContext.CompletionResponse, error) {

	if req.Runner == nil || req.Computer == nil {
		return nil, fmt.Errorf("runner and computer are required")
	}

	stdCtx := ctx.Context

	// Build a cancellable runnerCtx that bridges agentContext interrupts.
	runnerCtx, cancelRunner := context.WithCancel(stdCtx)
	defer cancelRunner() // Prevent goroutine leak.

	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if ctx.Interrupt != nil {
					if sig := ctx.Interrupt.Peek(); sig != nil {
						cancelRunner()
						return
					}
					if ctx.Interrupt.IsInterrupted() {
						cancelRunner()
						return
					}
				}
			case <-stdCtx.Done():
				cancelRunner()
				return
			}
		}
	}()

	if req.LoadingMsgID != "" {
		waitMsg := &message.Message{
			MessageID:   req.LoadingMsgID,
			Delta:       true,
			DeltaAction: message.DeltaReplace,
			Type:        message.TypeLoading,
			Props: map[string]any{
				"message": i18n.T(ctx.Locale, "sandbox.waiting_response"),
			},
		}
		ctx.Send(waitMsg)
	}

	var textContent []byte
	loadingClosed := false
	wrappedHandler := func(chunkType message.StreamChunkType, data []byte) int {
		if !loadingClosed && req.LoadingMsgID != "" {
			if chunkType == message.ChunkText || chunkType == message.ChunkToolCall || chunkType == message.ChunkExecute || chunkType == message.ChunkMessageStart {
				closeLoading(ctx, req.LoadingMsgID)
				loadingClosed = true
			}
		}
		if chunkType == message.ChunkText {
			textContent = append(textContent, data...)
		}
		if handler != nil {
			return handler(chunkType, data)
		}
		return 0
	}

	err := req.Runner.Stream(runnerCtx, req.StreamReq, wrappedHandler)

	if !loadingClosed && req.LoadingMsgID != "" {
		closeLoading(ctx, req.LoadingMsgID)
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("runner.Stream: %w", err)
	}

	resp := &agentContext.CompletionResponse{
		Role:         "assistant",
		FinishReason: agentContext.FinishReasonStop,
	}
	if len(textContent) > 0 {
		resp.Content = string(textContent)
	}
	return resp, nil
}

func closeLoading(ctx *agentContext.Context, loadingMsgID string) {
	if loadingMsgID == "" || ctx == nil {
		return
	}
	msg := &message.Message{
		MessageID:   loadingMsgID,
		Delta:       true,
		DeltaAction: message.DeltaReplace,
		Type:        message.TypeLoading,
		Props: map[string]any{
			"done":    true,
			"message": "",
		},
	}
	ctx.Send(msg)
}
