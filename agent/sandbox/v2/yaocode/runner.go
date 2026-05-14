package yaocode

import (
	"context"
	"fmt"

	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// YaoRunner is the Yao Code SDK runner, equivalent to claude/opencode.
// It executes locally in the Yao Server process (mode="local") without
// requiring a Tai node or container. The SDK is under development;
// Stream currently returns a not-implemented error.
type YaoRunner struct{}

func New() *YaoRunner { return &YaoRunner{} }

func (r *YaoRunner) Name() string { return "yaocode" }

// Prepare runs user-defined prepare steps (copy, exec, file) but adds
// no runner-specific steps. Connector is not required.
func (r *YaoRunner) Prepare(ctx context.Context, req *types.PrepareRequest) error {
	if req.RunSteps != nil && len(req.Config.Prepare) > 0 {
		return req.RunSteps(ctx, req.Config.Prepare, req.Computer, req.Config.ID, req.ConfigHash, req.AssistantDir)
	}
	return nil
}

// Stream executes the Yao Code SDK agent loop. Not yet implemented.
func (r *YaoRunner) Stream(_ context.Context, _ *types.StreamRequest, _ message.StreamFunc) error {
	return fmt.Errorf("yaocode runner (Yao Code SDK) is not yet implemented")
}

// Cleanup is a no-op for the yaocode runner.
func (r *YaoRunner) Cleanup(_ context.Context, _ infra.Computer) error {
	return nil
}
