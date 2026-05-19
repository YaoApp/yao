package i18n_test

import (
	"os"
	"testing"

	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	code := m.Run()
	testprepare.Cleanup()
	os.Exit(code)
}
