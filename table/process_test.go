package table

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/model"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xun/capsule"
)

func init() {
	share.DBConnect(config.Conf.Database)
	model.Load(config.Conf)
	share.Load(config.Conf)
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
	process := gou.NewProcess("xiang.table.Search", args...)
	response := ProcessSearch(process)
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

func TestTableProcessFind(t *testing.T) {
	args := []interface{}{
		"service",
		1,
		&gin.Context{},
	}
	process := gou.NewProcess("xiang.table.Find", args...)
	response := ProcessFind(process)
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.Equal(t, any.Of(res.Get("id")).CInt(), 1)
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
	process := gou.NewProcess("xiang.table.Save", args...)
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()

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
	response := ProcessSave(process)
	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())

	id := any.Of(response).CInt()
	args = []interface{}{
		"service",
		any.Of(response).Int(),
		"id",
	}
	process = gou.NewProcess("xiang.table.DeleteIn", args...)
	response = ProcessDeleteIn(process)

	assert.NotNil(t, response)
	assert.True(t, any.Of(response).IsInt())
	assert.Equal(t, any.Of(response).CInt(), 1)

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
