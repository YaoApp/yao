package workflow

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/flow"
	"github.com/yaoapp/xiang/model"
	"github.com/yaoapp/xiang/query"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xun/capsule"
)

func init() {
	share.DBConnect(config.Conf.Database)
	share.Load(config.Conf)
	model.Load(config.Conf)
	engineModels := path.Join(config.Conf.Source, "xiang", "models")
	model.LoadFrom(engineModels, "xiang.")
	query.Load(config.Conf)
	flow.Load(config.Conf)
	Load(config.Conf)
}

func TestLoad(t *testing.T) {
	share.DBConnect(config.Conf.Database)
	share.Load(config.Conf)
	model.Load(config.Conf)
	query.Load(config.Conf)
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func TestSave(t *testing.T) {
	assignFlow := Select("assign")
	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
		Data: map[string]interface{}{"id": 1, "name": "云主机"},
		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
	})

	data := maps.Of(wflow).Dot()
	assert.Equal(t, int64(1), data.Get("id"))
	assert.Equal(t, "assign", data.Get("name"))
	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
	assert.Equal(t, "进行中", data.Get("node_status"))
	assert.Equal(t, "进行中", data.Get("status"))
	assert.Equal(t, float64(1), data.Get("input.选择商务负责人.data.id"))
	assert.Equal(t, "云主机", data.Get("input.选择商务负责人.data.name"))
	assert.Equal(t, float64(1), data.Get("input.选择商务负责人.form.biz_id"))
	assert.Equal(t, "张良明", data.Get("input.选择商务负责人.form.name"))
	// 清理数据
	capsule.Query().From("xiang_workflow").Truncate()
}

func TestSaveUpdate(t *testing.T) {
	assignFlow := Select("assign")
	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
		Data: map[string]interface{}{"id": 1, "name": "云主机"},
		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
	})

	wflow = assignFlow.Save(1, "选择商务负责人", 1, Input{
		Data: map[string]interface{}{"id": 1, "name": "云存储"},
		Form: map[string]interface{}{"biz_id": 1, "name": "李明博"},
	})

	utils.Dump(wflow)

	data := maps.Of(wflow).Dot()
	assert.Equal(t, int64(1), data.Get("id"))
	assert.Equal(t, "assign", data.Get("name"))
	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
	assert.Equal(t, "进行中", data.Get("node_status"))
	assert.Equal(t, "进行中", data.Get("status"))
	assert.Equal(t, float64(1), data.Get("input.选择商务负责人.data.id"))
	assert.Equal(t, "云存储", data.Get("input.选择商务负责人.data.name"))
	assert.Equal(t, float64(1), data.Get("input.选择商务负责人.form.biz_id"))
	assert.Equal(t, "李明博", data.Get("input.选择商务负责人.form.name"))

	// 清理数据
	capsule.Query().From("xiang_workflow").Truncate()
}

func TestOpen(t *testing.T) {
	assignFlow := Select("assign")
	assignFlow.Save(1, "选择商务负责人", 1, Input{
		Data: map[string]interface{}{"id": 1, "name": "云主机"},
		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
	})
	wflow := assignFlow.Open(1, 1)
	data := maps.Of(wflow).Dot()
	assert.Equal(t, int64(1), data.Get("id"))
	assert.Equal(t, "assign", data.Get("name"))
	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
	assert.Equal(t, "进行中", data.Get("node_status"))
	assert.Equal(t, "进行中", data.Get("status"))
	assert.Equal(t, int64(1), data.Get("user_id"))
	assert.Equal(t, []interface{}{float64(1)}, data.Get("users"))

	// 清理数据
	capsule.Query().From("xiang_workflow").Truncate()
}

func TestOpenEmpty(t *testing.T) {
	assignFlow := Select("assign")
	wflow := assignFlow.Open(1, 1)
	data := maps.Of(wflow).Dot()
	assert.Equal(t, false, data.Has("id"))
	assert.Equal(t, "assign", data.Get("name"))
	assert.Equal(t, "选择商务负责人", data.Get("node_name"))
	assert.Equal(t, "进行中", data.Get("node_status"))
	assert.Equal(t, "进行中", data.Get("status"))
	assert.Equal(t, 1, data.Get("user_id"))
	assert.Equal(t, []interface{}{1}, data.Get("users"))
}

func TestNext(t *testing.T) {
	assignFlow := Select("assign")
	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
		Data: map[string]interface{}{"id": 1, "name": "云主机"},
		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
	})
	wflow = assignFlow.Next(1, any.Of(wflow["id"]).CInt(), map[string]interface{}{
		"项目名称":    "测试项目",
		"商务负责人名称": "林明波",
	})
	data := maps.Of(wflow).Dot()
	assert.Equal(t, int64(1), data.Get("id"))
	assert.Equal(t, "assign", data.Get("name"))
	assert.Equal(t, "项目负责人审批", data.Get("node_name"))
	assert.Equal(t, "进行中", data.Get("node_status"))
	assert.Equal(t, "进行中", data.Get("status"))
	assert.Equal(t, int64(2), data.Get("user_id"))
	assert.Equal(t, true, data.Has("users"))
	assert.Equal(t, "测试项目", data.Get("output.项目名称"))
	assert.Equal(t, "林明波", data.Get("output.商务负责人名称"))

	// 清理数据
	capsule.Query().From("xiang_workflow").Truncate()
}

func TestNextWhen(t *testing.T) {
	assignFlow := Select("assign")
	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
		Data: map[string]interface{}{"id": 1, "name": "云主机"},
		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
	})
	id := any.Of(wflow["id"]).CInt()
	assignFlow.Next(1, id, map[string]interface{}{
		"项目名称":    "测试项目",
		"商务负责人名称": "林明波",
	})

	// Next When
	wflow = assignFlow.Next(2, id, map[string]interface{}{"审批结果": "通过"})
	data := maps.Of(wflow).Dot()
	assert.Equal(t, int64(1), data.Get("id"))
	assert.Equal(t, "assign", data.Get("name"))
	assert.Equal(t, "审批通过", data.Get("node_name"))
	assert.Equal(t, "进行中", data.Get("node_status"))
	assert.Equal(t, "进行中", data.Get("status"))
	assert.Equal(t, int64(1), data.Get("user_id"))
	assert.Equal(t, true, data.Has("users"))
	assert.Equal(t, "测试项目", data.Get("output.项目名称"))
	assert.Equal(t, "林明波", data.Get("output.商务负责人名称"))
	assert.Equal(t, "通过", data.Get("output.审批结果"))

	// 清理数据
	capsule.Query().From("xiang_workflow").Truncate()
}

func check(t *testing.T) {
	keys := []string{}
	for key, workflow := range WorkFlows {
		keys = append(keys, key)
		workflow.Reload()
	}
	assert.Equal(t, 1, len(keys))
}
