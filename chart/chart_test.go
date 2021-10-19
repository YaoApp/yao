package chart

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/model"
	"github.com/yaoapp/xiang/query"
	"github.com/yaoapp/xiang/share"
)

func TestLoad(t *testing.T) {
	share.DBConnect(config.Conf.Database)
	model.Load(config.Conf)
	query.Load(config.Conf)

	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range Charts {
		keys = append(keys, key)
	}
	assert.Equal(t, 1, len(keys))
}
