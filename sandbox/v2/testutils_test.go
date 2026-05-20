package sandbox_test

import (
	"os"
	"testing"

	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	sandboxtest.PurgeStaleContainers("sb-")
	code := m.Run()
	testprepare.Cleanup()
	os.Exit(code)
}
