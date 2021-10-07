package global

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/table"
)

func TestLoad(t *testing.T) {
	defer Load(config.Conf)
	assert.NotPanics(t, func() {
		Load(config.Conf)
	})
}

// 从文件系统载入引擎文件
func TestLoadEngineFS(t *testing.T) {
	defer Load(config.Conf)
	root := "fs://" + path.Join(config.Conf.Source, "/xiang")
	assert.NotPanics(t, func() {
		LoadEngine(root)
	})

}

// 从BinDataz载入引擎文件
func TestLoadEngineBin(t *testing.T) {
	defer Load(config.Conf)
	root := "bin://xiang"
	assert.NotPanics(t, func() {
		LoadEngine(root)
	})
}

// 从文件系统载入应用脚本
func TestLoadAppFS(t *testing.T) {
	defer Load(config.Conf)
	assert.NotPanics(t, func() {
		LoadApp(AppRoot{
			APIs:    config.Conf.RootAPI,
			Flows:   config.Conf.RootFLow,
			Models:  config.Conf.RootModel,
			Plugins: config.Conf.RootPlugin,
			Tables:  config.Conf.RootTable,
			Charts:  config.Conf.RootChart,
			Screens: config.Conf.RootScreen,
		})
	})
}

func TestTableLoad(t *testing.T) {
	defer Load(config.Conf)
	table, has := table.Tables["service"]
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
