package global

import (
	"os"
	"path"
	"testing"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/table"
)

var cfg config.Config

func TestMain(m *testing.M) {

	// 加载模型等
	Load(Conf)

	// 加载表格(临时)
	root := "fs://" + path.Join(Conf.Source, "/app/tables/service.json")
	table.Load(root, "service").Reload()

	// Run test suites
	exitVal := m.Run()

	// we can do clean up code here
	gou.KillPlugins()

	os.Exit(exitVal)
}
