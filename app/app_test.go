package app

import (
	"os"
	"testing"

	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	os.Setenv("YAO_LANG", "zh-cn")
	Load(config.Conf)
	// assert.Equal(t, "YAO", share.App.L["Yao"])
	// assert.Equal(t, "象传", share.App.L["Xiang"])
}
