package global

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/xiang/table"
)

func TestLoad(t *testing.T) {
	assert.NotPanics(t, func() {
		Load(Conf)
	})
}

// 从文件系统载入引擎文件
func TestLoadEngineFS(t *testing.T) {
	root := "fs://" + path.Join(Conf.Source, "/xiang")
	assert.NotPanics(t, func() {
		LoadEngine(root)
	})

}

// 从BinDataz载入引擎文件
func TestLoadEngineBin(t *testing.T) {
	root := "bin://xiang"
	assert.NotPanics(t, func() {
		LoadEngine(root)
	})
}

// 从文件系统载入应用脚本
func TestLoadAppFS(t *testing.T) {
	assert.NotPanics(t, func() {
		LoadApp(AppRoot{
			APIs:    Conf.RootAPI,
			Flows:   Conf.RootFLow,
			Models:  Conf.RootModel,
			Plugins: Conf.RootPlugin,
			Tables:  Conf.RootTable,
			Charts:  Conf.RootChart,
			Screens: Conf.RootScreen,
		})
	})
}

func TestTableLoad(t *testing.T) {

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
