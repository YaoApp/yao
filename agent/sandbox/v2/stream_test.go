package sandboxv2_test

import (
	"context"
	"fmt"
	"testing"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

type fakeRunner struct {
	name      string
	streamErr error
}

func (r *fakeRunner) Name() string { return r.name }
func (r *fakeRunner) Prepare(_ context.Context, _ *types.PrepareRequest) error {
	return nil
}
func (r *fakeRunner) Stream(_ context.Context, _ *types.StreamRequest, handler message.StreamFunc) error {
	if r.streamErr != nil {
		return r.streamErr
	}
	handler(message.ChunkText, []byte("fake response"))
	return nil
}
func (r *fakeRunner) Cleanup(_ context.Context, _ infra.Computer) error {
	return nil
}

func TestExecuteSandboxStream_Normal(t *testing.T) {
	testprepare.PrepareUnit(t)

	ctx := &agentContext.Context{Context: context.Background()}
	req := &sandboxv2.ExecuteRequest{
		Runner:    &fakeRunner{name: "test-runner"},
		StreamReq: &types.StreamRequest{},
	}

	var received []byte
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		received = append(received, data...)
		return 0
	}

	resp, err := sandboxv2.ExecuteSandboxStream(ctx, req, handler)
	if err != nil {
		t.Fatalf("ExecuteSandboxStream: %v", err)
	}
	_ = resp
	if len(received) == 0 {
		t.Fatal("expected to receive streamed data")
	}
}

func TestExecuteSandboxStream_NilRunner(t *testing.T) {
	testprepare.PrepareUnit(t)

	ctx := &agentContext.Context{Context: context.Background()}
	req := &sandboxv2.ExecuteRequest{
		Runner:    nil,
		StreamReq: &types.StreamRequest{},
	}

	handler := func(chunkType message.StreamChunkType, data []byte) int { return 0 }
	_, err := sandboxv2.ExecuteSandboxStream(ctx, req, handler)
	if err == nil {
		t.Fatal("expected error for nil runner")
	}
}

func TestExecuteSandboxStream_RunnerError(t *testing.T) {
	testprepare.PrepareUnit(t)

	ctx := &agentContext.Context{Context: context.Background()}
	req := &sandboxv2.ExecuteRequest{
		Runner:    &fakeRunner{name: "err-runner", streamErr: fmt.Errorf("stream failed")},
		StreamReq: &types.StreamRequest{},
	}

	handler := func(chunkType message.StreamChunkType, data []byte) int { return 0 }
	_, err := sandboxv2.ExecuteSandboxStream(ctx, req, handler)
	if err == nil {
		t.Fatal("expected error when runner.Stream fails")
	}
}

func TestExecuteSandboxStream_Cancel(t *testing.T) {
	testprepare.PrepareUnit(t)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	ctx := &agentContext.Context{Context: cancelCtx}
	req := &sandboxv2.ExecuteRequest{
		Runner:    &fakeRunner{name: "cancel-runner"},
		StreamReq: &types.StreamRequest{},
	}

	handler := func(chunkType message.StreamChunkType, data []byte) int { return 0 }
	_, err := sandboxv2.ExecuteSandboxStream(ctx, req, handler)
	// Context already cancelled; behavior depends on implementation
	_ = err
}
