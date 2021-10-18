package table

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/model"
	"github.com/yaoapp/xiang/share"
)

func TestLoad(t *testing.T) {
	share.DBConnect(config.Conf.Database)
	model.Load(config.Conf)

	Tables = make(map[string]*Table)
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range Tables {
		keys = append(keys, key)
	}
	assert.Equal(t, 2, len(keys))
}
