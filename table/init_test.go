package table

import (
	"os"
	"path"
	"testing"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/global"
)

func TestMain(m *testing.M) {

	// 加载模型等
	global.Load(global.Conf)

	// 加载表格(临时)
	root := "fs://" + path.Join(global.Conf.Source, "/app/tables/service.json")
	Load(root, "service").Reload()

	// Run test suites
	exitVal := m.Run()

	// we can do clean up code here
	gou.KillPlugins()

	os.Exit(exitVal)
}
