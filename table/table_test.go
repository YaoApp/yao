package table

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/global"
)

func TestLoad(t *testing.T) {
	global.Load(global.Conf)
	root := "fs://" + path.Join(global.Conf.Source, "/app/tables/service.json")
	Load(root, "service").Reload()

	table, has := Tables["service"]
	assert.Equal(t, table.Table, "service")
	assert.Equal(t, table.Source, root)
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
