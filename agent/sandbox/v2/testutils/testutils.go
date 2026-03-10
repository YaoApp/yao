package testutils

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	agenttestutils "github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/config"
	sandboxv2 "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
)

// Prepare initializes the full environment required for sandbox V2 E2E tests:
//   - agent layer (assistants, LLM, caller)
//   - tai registry + local node
//   - sandbox V2 manager
func Prepare(t *testing.T) {
	t.Helper()

	agenttestutils.Prepare(t)

	if registry.Global() == nil {
		registry.Init(nil)
	}

	dataDir := filepath.Join(config.Conf.DataRoot, "workspaces")
	os.MkdirAll(dataDir, 0755)
	tai.RegisterLocal(tai.WithDataDir(dataDir))

	sandboxv2.Init()
	if err := sandboxv2.M().Start(context.Background()); err != nil {
		t.Fatalf("sandbox v2 manager start: %v", err)
	}

	t.Cleanup(func() {
		sandboxv2.M().Close()
	})
}

// Clean tears down the test environment.
func Clean(t *testing.T) {
	t.Helper()
	agenttestutils.Clean(t)
}
