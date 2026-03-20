package sandboxv2

import (
	"context"
	"errors"
	"fmt"
	"log"
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

// ExecuteSandboxStream is the V2 replacement for executeSandboxStream.
// It calls runner.Stream, handles interrupts, and performs cleanup/lifecycle
// in defer.
func ExecuteSandboxStream(
	ctx *agentContext.Context,
	req *ExecuteRequest,
	handler message.StreamFunc,
) (*agentContext.CompletionResponse, error) {

	if req.Runner == nil || req.Computer == nil {
		return nil, fmt.Errorf("runner and computer are required")
	}

	stdCtx := ctx.Context
	panicked := true // Assume panic; set false on normal exit.

	// Resolve stop timeout from config (default 2s).
	stopTimeout := 2 * time.Second
	if req.Config != nil && req.Config.StopTimeout != "" {
		if d, err := time.ParseDuration(req.Config.StopTimeout); err == nil {
			stopTimeout = d
		}
	}

	// Panic recovery (registered first, executes last in LIFO order).
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[sandbox/v2] panic in stream: %v", r)
			cleanCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
			defer cancel()
			req.Runner.Cleanup(cleanCtx, req.Computer)
			LifecycleAction(cleanCtx, req.Config, req.Computer, req.Manager)
		}
	}()

	// Lifecycle action (registered second, executes second-to-last).
	defer func() {
		if !panicked {
			LifecycleAction(stdCtx, req.Config, req.Computer, req.Manager)
		}
	}()

	// Runner cleanup (registered last, executes first).
	defer func() {
		if !panicked {
			cleanCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
			defer cancel()
			req.Runner.Cleanup(cleanCtx, req.Computer)
		}
	}()

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

	panicked = false // Normal exit reached.

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
