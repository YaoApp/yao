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

	table, has := Tables["service"]
	assert.Equal(t, table.Table, "service")
	assert.True(t, has)

	_, has = table.Columns["id"]
	assert.True(t, has)

	price, has := table.Columns["计费方式"]
	assert.True(t, has)
	if has {
		assert.True(t, price.Edit.Props["multiple"].(bool))
	}

	_, has = table.Filters["id"]
	assert.True(t, has)

	keywords, has := table.Filters["关键词"]
	assert.True(t, has)
	if has {
		assert.True(t, keywords.Bind == "where.name.like")
	}
}

func check(t *testing.T) {
	keys := []string{}
	for key := range Tables {
		keys = append(keys, key)
	}
	assert.Equal(t, 2, len(keys))
}
