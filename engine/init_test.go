package engine

import (
	"os"
	"testing"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
)

var cfg config.Config

func TestMain(m *testing.M) {

	// 加载模型等
	Load(config.Conf)

	// Run test suites
	exitVal := m.Run()

	// we can do clean up code here
	gou.KillPlugins()

	os.Exit(exitVal)
}
