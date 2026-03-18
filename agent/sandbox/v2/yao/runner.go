package yao

import (
	"context"

	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// YaoRunner is a no-op Runner for pure Hook-driven sandbox interactions.
// When runner.name == "yao", the assistant relies entirely on Create/Next
// hooks for logic; no external CLI is invoked.
type YaoRunner struct{}

func New() *YaoRunner { return &YaoRunner{} }

func (r *YaoRunner) Name() string { return "yao" }

// Prepare runs user-defined prepare steps (copy, exec, file) but adds
// no runner-specific steps. Connector is not required.
func (r *YaoRunner) Prepare(ctx context.Context, req *types.PrepareRequest) error {
	if req.RunSteps != nil && len(req.Config.Prepare) > 0 {
		return req.RunSteps(ctx, req.Config.Prepare, req.Computer, req.Config.ID, req.ConfigHash, req.AssistantDir)
	}
	return nil
}

// Stream is a no-op — hooks handle all interaction. Returns immediately
// so the assistant framework proceeds to the Next hook.
func (r *YaoRunner) Stream(_ context.Context, _ *types.StreamRequest, _ message.StreamFunc) error {
	return nil
}

// Cleanup is a no-op for the yao runner.
func (r *YaoRunner) Cleanup(_ context.Context, _ infra.Computer) error {
	return nil
}
