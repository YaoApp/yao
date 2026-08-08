package image_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMain(m *testing.M) {
	if !agentEnvExists() {
		fmt.Println("SKIP: agent-test.env not found (only available in Agent Unit Test workflow)")
		os.Exit(0)
	}
	testprepare.MustLoadEnv()
	code := m.Run()
	testprepare.Cleanup()
	os.Exit(code)
}

func agentEnvExists() bool {
	_, thisFile, _, _ := runtime.Caller(0)
	yaoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	envFile := filepath.Join(yaoRoot, "unit-test", "agent", "env", "agent-test.env")
	_, err := os.Stat(envFile)
	return err == nil
}
