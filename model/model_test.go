package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {
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
	assert.Equal(t, demo.MetaData.Name, "::Demo")
	assert.Equal(t, demo.Columns["action"].Label, "::Action")
}
