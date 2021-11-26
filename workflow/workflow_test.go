package workflow

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xiang/config"
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
	assignFlow.Save(1, "选择商务负责人", 1, Input{
		Data: map[string]interface{}{"id": 1, "name": "云主机"},
		Form: map[string]interface{}{"biz_id": 1, "name": "张良明"},
	})

	wflow := assignFlow.Save(1, "选择商务负责人", 1, Input{
		Data: map[string]interface{}{"id": 1, "name": "云存储"},
		Form: map[string]interface{}{"biz_id": 1, "name": "李明博"},
	})

	data := maps.Of(wflow).Dot()
	assert.Equal(t, int64(1), data.Get("id"))
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

func check(t *testing.T) {
	keys := []string{}
	for key, workflow := range WorkFlows {
		keys = append(keys, key)
		workflow.Reload()
	}
	assert.Equal(t, 1, len(keys))
}
