package tai

import (
	"context"
	"fmt"

	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// TaiRunner is a placeholder for the future Tai direct-execution runner.
type TaiRunner struct{}

func New() *TaiRunner { return &TaiRunner{} }

func (r *TaiRunner) Name() string { return "tai" }

func (r *TaiRunner) Prepare(_ context.Context, _ *types.PrepareRequest) error {
	return fmt.Errorf("tai runner not yet implemented")
}

func (r *TaiRunner) Stream(_ context.Context, _ *types.StreamRequest, _ message.StreamFunc) error {
	return fmt.Errorf("tai runner not yet implemented")
}

func (r *TaiRunner) Cleanup(_ context.Context, _ infra.Computer) error {
	return nil
}
