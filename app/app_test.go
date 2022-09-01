package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {
	os.Setenv("YAO_LANG", "zh-cn")
	Load(config.Conf)
	assert.Equal(t, "YAO", share.App.L["Yao"])
	assert.Equal(t, "象传", share.App.L["Xiang"])
}
