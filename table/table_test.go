package table

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {

	share.DBConnect(config.Conf.DB)
	share.Load(config.Conf)
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
	assert.Equal(t, 11, len(keys))

	// demo := Select("demo")
	// utils.Dump(demo)

	// assert.NotNil(t, demo.Columns["類型"])
	// assert.NotNil(t, demo.Columns["類型"].Edit.Props["options"])

	// options := demo.Columns["類型"].Edit.Props["options"].([]interface{})
	// opt1 := options[0].(map[string]interface{})
	// assert.Equal(t, "貓", opt1["label"])

}
