package sandboxv2_test

import (
	"os"
	"testing"

	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	sandboxtest.PurgeStaleContainers("sb-prep-", "sb-lc-")
	code := m.Run()
	testprepare.Cleanup()
	os.Exit(code)
}
