package model

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/lang"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {

	os.Setenv("YAO_LANG", "zh-hk")
	lang.Load(config.Conf)
	share.DBConnect(config.Conf.DB)

	gou.Models = make(map[string]*gou.Model)
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range gou.Models {
		keys = append(keys, key)
	}
	assert.Equal(t, 11, len(keys))

	demo := gou.Select("demo")
	assert.Equal(t, demo.MetaData.Name, "演示")
	assert.Equal(t, demo.Columns["action"].Label, "動作")
}
