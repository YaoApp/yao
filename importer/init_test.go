package importer

import (
	"os"
	"testing"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/query"
	"github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/share"
)

func TestMain(m *testing.M) {
	runtime.Load(config.Conf)
	share.DBConnect(config.Conf.DB)
	model.Load(config.Conf)
	query.Load(config.Conf)
	share.Load(config.Conf)
	script.Load(config.Conf)
	Load(config.Conf)
	code := m.Run()
	os.Exit(code)
}
