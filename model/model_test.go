package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {
	share.DBConnect(config.Conf.DB)
	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range model.Models {
		keys = append(keys, key)
	}
	assert.Equal(t, 11, len(keys))

	demo := model.Select("demo")
	assert.Equal(t, demo.MetaData.Name, "::Demo")
	assert.Equal(t, demo.Columns["action"].Label, "::Action")
}
