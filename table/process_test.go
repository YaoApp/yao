package table

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/flow"
	_ "github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/query"
	"github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/share"
)

func init() {
	runtime.Load(config.Conf)
	share.DBConnect(config.Conf.DB)
	model.Load(config.Conf)
	share.Load(config.Conf)
	query.Load(config.Conf)
	script.Load(config.Conf)
	flow.LoadFrom(filepath.Join(config.Conf.Root, "flows", "hooks"), "hooks.")
	Load(config.Conf)
}
func TestTableProcessSearch(t *testing.T) {

	args := []interface{}{
		"service",
		gou.QueryParam{
			Wheres: []gou.QueryWhere{
				{Column: "status", Value: "enabled"},
			},
		},
		1,
		2,
		&gin.Context{},
	}
	response := gou.NewProcess("xiang.table.Search", args...).Run()
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.True(t, res.Has("data"))
	assert.True(t, res.Has("next"))
	assert.True(t, res.Has("page"))
	assert.True(t, res.Has("pagecnt"))
	assert.True(t, res.Has("pagesize"))
	assert.True(t, res.Has("prev"))
	assert.True(t, res.Has("total"))
	assert.Equal(t, 1, res.Get("page"))
	assert.Equal(t, 2, res.Get("pagesize"))
}

func TestTableProcessSearchWithHook(t *testing.T) {

	args := []interface{}{"hooks.search"}
	response := gou.NewProcess("xiang.table.Search", args...).Run()
	// utils.Dump(response)

	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.True(t, res.Has("data"))
	assert.True(t, res.Has("next"))
	assert.True(t, res.Has("page"))
	assert.True(t, res.Has("pagecnt"))
	assert.True(t, res.Has("pagesize"))
	assert.True(t, res.Has("prev"))
	assert.True(t, res.Has("total"))
	assert.Equal(t, 1, res.Get("page"))
	assert.Equal(t, 2, res.Get("pagesize"))
	assert.Equal(t, float64(100), res.Get("after"))
}

func TestTableProcessFind(t *testing.T) {
	args := []interface{}{
		"service",
		1,
		&gin.Context{},
	}
	response := gou.NewProcess("xiang.table.Find", args...).Run()
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.Equal(t, 1, any.Of(res.Get("id")).CInt())
}

func TestTableProcessFindWithHook(t *testing.T) {
	args := []interface{}{"hooks.find"}
	response := gou.NewProcess("xiang.table.Find", args...).Run()
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.Equal(t, 1, any.Of(res.Get("id")).CInt())
	assert.Equal(t, float64(100), res.Get("after"))
}

func TestTableProcessSave(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	response := gou.NewProcess("xiang.table.Save", args...).Run()
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()

	// 清空数据
	capsule.Query().Table("service").Where("id", id).Delete()
}

func TestTableProcessSaveWithHook(t *testing.T) {
	args := []interface{}{"hooks.save"}
	response := gou.NewProcess("xiang.table.Save", args...).Run()
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.True(t, any.Of(res.Get("id")).IsInt())
	assert.Equal(t, float64(100), res.Get("after"))

	id := any.Of(res.Get("id")).CInt()

	// 清空数据
	capsule.Query().Table("service").Where("id", id).Delete()
}

func TestTableProcessDelete(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	process := gou.NewProcess("xiang.table.Save", args...)
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()
	args = []interface{}{
		"service",
		id,
	}
	process = gou.NewProcess("xiang.table.Delete", args...)
	response = ProcessDelete(process)
	assert.Nil(t, response)

	// 清空数据
	capsule.Query().Table("service").Where("id", id).Delete()
}

func TestTableProcessInsert(t *testing.T) {
	args := []interface{}{
		"service",
		[]string{"name", "short_name", "kind_id", "manu_id", "price_options"},
		[][]interface{}{
			{"I腾讯云主机I1", "高性能云主机", 3, 1, []string{"按月订阅"}},
			{"I腾讯云主机I2", "高性能云主机", 3, 1, []string{"按月订阅"}},
		},
	}
	process := gou.NewProcess("xiang.table.Insert", args...)
	response := ProcessInsert(process)
	assert.Nil(t, response)

	// 清空数据
	capsule.Query().Table("service").Where("name", "like", "I腾讯云主机I%").Delete()
}

func TestTableProcessDeleteWhere(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	process := gou.NewProcess("xiang.table.Save", args...)
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()
	args = []interface{}{
		"service",
		gou.QueryParam{
			Wheres: []gou.QueryWhere{
				{Column: "id", Value: id},
			},
		},
	}
	process = gou.NewProcess("xiang.table.DeleteWhere", args...)
	response = ProcessDeleteWhere(process)

	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())
	assert.Equal(t, any.Of(response).CInt(), 1)

	// 清空数据
	capsule.Query().Table("service").Where("id", id).Delete()
}

func TestTableProcessDeleteIn(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	process := gou.NewProcess("xiang.table.Save", args...)
	id := ProcessSave(process)
	assert.NotNil(t, id)
	assert.True(t, any.Of(id).IsNumber())

	// id := any.Of(response).CInt()
	args = []interface{}{"service", id, "id"}
	process = gou.NewProcess("xiang.table.DeleteIn", args...)
	id = ProcessDeleteIn(process)

	assert.NotNil(t, id)
	assert.True(t, any.Of(id).IsNumber())
	assert.Equal(t, id, 1)

	// 清空数据
	capsule.Query().Table("service").Where("id", id).Delete()
}

func TestTableProcessUpdateWhere(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	process := gou.NewProcess("xiang.table.Save", args...)
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()
	args = []interface{}{
		"service",
		gou.QueryParam{
			Wheres: []gou.QueryWhere{
				{Column: "id", Value: id},
			},
		},
		map[string]interface{}{
			"name": "腾讯黑岩云主机UP",
		},
	}
	process = gou.NewProcess("xiang.table.UpdateWhere", args...)
	response = ProcessUpdateWhere(process)

	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())
	assert.Equal(t, any.Of(response).CInt(), 1)

	// 清空数据
	capsule.Query().Table("service").Where("id", id).Delete()
}

func TestTableProcessUpdateIn(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	process := gou.NewProcess("xiang.table.Save", args...)
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()
	args = []interface{}{
		"service",
		id,
		"id",
		map[string]interface{}{
			"name": "腾讯黑岩云主机UP",
		},
	}
	process = gou.NewProcess("xiang.table.UpdateIn", args...)
	response = ProcessUpdateIn(process)

	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())
	assert.Equal(t, any.Of(response).CInt(), 1)

	// 清空数据
	capsule.Query().Table("service").Where("id", id).Delete()
}

func TestTableProcessSetting(t *testing.T) {
	args := []interface{}{"service", ""}
	process := gou.NewProcess("xiang.table.Setting", args...)
	response := ProcessSetting(process)
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.Equal(t, res.Get("name"), "云服务库")
	assert.True(t, res.Has("title"))
	assert.True(t, res.Has("decription"))
	assert.True(t, res.Has("columns"))
	assert.True(t, res.Has("filters"))
	assert.True(t, res.Has("list"))
	assert.True(t, res.Has("edit"))
	assert.True(t, res.Has("view"))
	assert.True(t, res.Has("insert"))
}
func TestTableProcessSettingList(t *testing.T) {
	args := []interface{}{"service", "list"}
	process := gou.NewProcess("xiang.table.Setting", args...)
	response := ProcessSetting(process)
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.True(t, res.Has("actions"))
	assert.True(t, res.Has("layout"))
	assert.True(t, res.Has("primary"))
}

func TestTableProcessSettingListEdit(t *testing.T) {
	args := []interface{}{"service", "list, edit"}
	process := gou.NewProcess("xiang.table.Setting", args...)
	response := ProcessSetting(process)
	assert.NotNil(t, response)
	res := any.Of(response).MapStr().Dot()
	assert.True(t, res.Has("list.actions"))
	assert.True(t, res.Has("list.layout"))
	assert.True(t, res.Has("list.primary"))
	assert.True(t, res.Has("edit.actions"))
	assert.True(t, res.Has("edit.layout"))
	assert.True(t, res.Has("edit.primary"))
}

func TestTableProcessQuickSave(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	process := gou.NewProcess("xiang.table.Save", args...)
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()
	args = []interface{}{
		"service",
		id,
	}

	process = gou.NewProcess("xiang.table.QuickSave",
		"service",
		map[string]interface{}{
			"delete": []int{id},
			"data": []map[string]interface{}{
				{
					"name":          "腾讯黑岩云主机2",
					"short_name":    "高性能云主机2",
					"kind_id":       3,
					"manu_id":       1,
					"price_options": []string{"按月订阅"},
				},
			},
			"query": map[string]interface{}{"manu_id": 1},
		})
	res := process.Run()
	newID, ok := res.([]interface{})
	assert.True(t, ok)
	assert.NotEqual(t, id, newID[0])

	// 清空数据
	capsule.Query().Table("service").WhereIn("id", []interface{}{id, newID[0]}).Delete()
}

func TestTableProcessSelect(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	process := gou.NewProcess("xiang.table.Save", args...)
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()

	args = []interface{}{"service", "腾讯", "name", "id"}

	data := gou.NewProcess("xiang.table.select", args...).Run().([]maps.MapStrAny)
	assert.Greater(t, len(data), 1)
	for _, row := range data {
		assert.Contains(t, row.Get("name"), "腾讯")
	}

	// 清空数据
	capsule.Query().Table("service").WhereIn("id", []int{id}).Delete()
}

func TestTableProcessSelectWithHook(t *testing.T) {
	args := []interface{}{
		"service",
		map[string]interface{}{
			"name":          "腾讯黑岩云主机",
			"short_name":    "高性能云主机",
			"kind_id":       3,
			"manu_id":       1,
			"price_options": []string{"按月订阅"},
		},
	}
	process := gou.NewProcess("xiang.table.Save", args...)
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()

	args = []interface{}{"hooks.select"}

	data := gou.NewProcess("xiang.table.select", args...).Run().([]maps.MapStrAny)
	assert.Greater(t, len(data), 1)
	for _, row := range data {
		assert.Contains(t, row.Get("name"), "腾讯")
		assert.Equal(t, float64(100), row.Get("after"))
	}

	// 清空数据
	capsule.Query().Table("service").WhereIn("id", []int{id}).Delete()
}

func TestTableProcessExport(t *testing.T) {

	args := []interface{}{
		"service",
		gou.QueryParam{Wheres: []gou.QueryWhere{{Column: "status", Value: "enabled"}}},
		2,
	}
	response := gou.NewProcess("xiang.table.Export", args...).Run()
	assert.NotNil(t, response)
	// fmt.Println(response)
	// res := any.Of(response).Map()
	// assert.True(t, res.Has("data"))
	// assert.True(t, res.Has("next"))
	// assert.True(t, res.Has("page"))
	// assert.True(t, res.Has("pagecnt"))
	// assert.True(t, res.Has("pagesize"))
	// assert.True(t, res.Has("prev"))
	// assert.True(t, res.Has("total"))
	// assert.Equal(t, 1, res.Get("page"))
	// assert.Equal(t, 2, res.Get("pagesize"))
}

func TestTableProcessExportWithHook(t *testing.T) {

	args := []interface{}{"hooks.search"}
	response := gou.NewProcess("xiang.table.Export", args...).Run()

	assert.NotNil(t, response)
	// res := any.Of(response).Map()
	// assert.True(t, res.Has("data"))
	// assert.True(t, res.Has("next"))
	// assert.True(t, res.Has("page"))
	// assert.True(t, res.Has("pagecnt"))
	// assert.True(t, res.Has("pagesize"))
	// assert.True(t, res.Has("prev"))
	// assert.True(t, res.Has("total"))
	// assert.Equal(t, 1, res.Get("page"))
	// assert.Equal(t, 2, res.Get("pagesize"))
	// assert.Equal(t, float64(100), res.Get("after"))
}

func TestTableProcessExportWithScriptHook(t *testing.T) {
	// testData()
	args := []interface{}{"hooks.search_script"}
	response := gou.NewProcess("xiang.table.Export", args...).Run()
	// utils.Dump(response)

	assert.NotNil(t, response)
	// res := any.Of(response).Map()
	// assert.True(t, res.Has("data"))
	// assert.True(t, res.Has("next"))
	// assert.True(t, res.Has("page"))
	// assert.True(t, res.Has("pagecnt"))
	// assert.True(t, res.Has("pagesize"))
	// assert.True(t, res.Has("prev"))
	// assert.True(t, res.Has("total"))
	// assert.Equal(t, 1, res.Get("page"))
	// assert.Equal(t, 2, res.Get("pagesize"))
	// assert.Equal(t, float64(100), res.Get("after"))
}

func testData() {
	m := gou.Select("service")
	data := [][]interface{}{}
	for i := 0; i < 2000; i++ {
		col := []interface{}{fmt.Sprintf("NAME-%d", i), 1, 1}
		data = append(data, col)
	}

	m.DestroyWhere(gou.QueryParam{
		Wheres: []gou.QueryWhere{
			{Column: "id", OP: "ge", Value: 0},
		},
	})
	err := m.Insert([]string{"name", "kind_id", "manu_id"}, data)
	fmt.Println(err)
}
