package dashboard

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	q "github.com/yaoapp/yao/query"
)

func TestProcessData(t *testing.T) {
	load(t)
	args := []interface{}{"workspace", map[string]interface{}{"range": "2022-01-02", "status": "checked"}}
	res, err := gou.NewProcess("yao.dashboard.Data", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr()
	assert.Equal(t, 14, len(data))
}

func TestProcessComponent(t *testing.T) {
	load(t)
	args := []interface{}{
		"workspace",
		"fields.filter.状态.edit.props.xProps",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}

	res, err := gou.NewProcess("yao.dashboard.Component", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	pets, ok := res.([]maps.MapStr)
	assert.True(t, ok)
	assert.Equal(t, 2, len(pets))
	assert.Equal(t, "Cookie", pets[0]["name"])
	assert.Equal(t, "checked", pets[0]["status"])
	assert.Equal(t, "Baby", pets[1]["name"])
	assert.Equal(t, "checked", pets[1]["status"])

	args = []interface{}{
		"workspace",
		"fields.dashboard.图表展示1",
		"data",
		map[string]interface{}{"foo": "bar"},
	}

	res2, err := gou.NewProcess("yao.dashboard.Component", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	values, ok := res2.([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(values), 1)

	args = []interface{}{
		"workspace",
		"fields.dashboard.图表展示2",
		"data",
		map[string]interface{}{"foo": "bar"},
	}

	res2, err = gou.NewProcess("yao.dashboard.Component", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	values, ok = res2.([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(values), 1)

}

func TestProcessComponentError(t *testing.T) {
	load(t)
	args := []interface{}{
		"workspace",
		"fields.filter.edit.props.状态.::not-exist",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}
	_, err := gou.NewProcess("yao.dashboard.Component", args...).Exec()
	assert.Contains(t, err.Error(), "fields.filter.edit.props.状态.::not-exist")
}

func TestProcessSetting(t *testing.T) {
	load(t)
	args := []interface{}{"workspace"}
	res, err := gou.NewProcess("yao.dashboard.Setting", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.filter."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.dashboard."+url.QueryEscape("图表展示1")+"/data", data.Get("fields.dashboard.图表展示1.data.api"))
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.dashboard."+url.QueryEscape("图表展示2")+"/data", data.Get("fields.dashboard.图表展示2.data.api"))
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.dashboard."+url.QueryEscape("宠物列表")+".view.props/"+url.QueryEscape("on:change"), data.Get("hooks.onChange.宠物列表.api"))
}

func TestProcessXgen(t *testing.T) {
	load(t)
	args := []interface{}{"workspace"}
	res, err := gou.NewProcess("yao.dashboard.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.filter."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.dashboard."+url.QueryEscape("图表展示1")+"/data", data.Get("fields.dashboard.图表展示1.data.api"))
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.dashboard."+url.QueryEscape("图表展示2")+"/data", data.Get("fields.dashboard.图表展示2.data.api"))
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.dashboard."+url.QueryEscape("宠物列表")+".view.props/"+url.QueryEscape("on:change"), data.Get("hooks.onChange.宠物列表.api"))
}

func TestProcessXgenWithPermissions(t *testing.T) {
	load(t)
	session.Global().Set("__permissions", map[string]interface{}{
		"dashboards.workspace": []string{
			"7f46a38d7ff3f1832375ff63cd412f41", // operation.actions[0] 跳转至大屏
			"09302a46b1b6f13a346deeea79b859dd", // 时间区间
			"8b445709024e0e5361d8bcdd58c75fcb", // 图表展示2
			"0bdee1c9858ef2a821a0ff7109d3fc5b", // 图表展示1
		},
	})

	args := []interface{}{"workspace"}
	res, err := gou.NewProcess("yao.dashboard.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.filter."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
	assert.NotEqual(t, "时间区间", data.Get("filter.columns[0].name"))
	assert.Equal(t, nil, data.Get("actions[0]"))
	assert.Equal(t, nil, data.Get("fields.dashboard.图表展示1"))
	assert.Equal(t, nil, data.Get("fields.dashboard.图表展示2"))

	session.Global().Set("__permissions", nil)
	res, err = gou.NewProcess("yao.dashboard.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data = any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/dashboard/workspace/component/fields.filter."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "时间区间", data.Get("filter.columns[0].name"))
	assert.NotEqual(t, nil, data.Get("actions[0]"))
	assert.NotEqual(t, nil, data.Get("fields.dashboard.图表展示1"))
	assert.NotEqual(t, nil, data.Get("fields.dashboard.图表展示2"))
}

func load(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	q.Load(config.Conf)
	clear(t)
	testData(t)
}

func testData(t *testing.T) {
	pet := gou.Select("pet")
	err := pet.Insert(
		[]string{"name", "type", "status", "mode", "stay", "cost", "doctor_id"},
		[][]interface{}{
			{"Cookie", "cat", "checked", "enabled", 200, 105, 1},
			{"Baby", "dog", "checked", "enabled", 186, 24, 1},
			{"Poo", "others", "checked", "enabled", 199, 66, 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func clear(t *testing.T) {
	for _, m := range gou.Models {
		err := m.DropTable()
		if err != nil {
			t.Fatal(err)
		}
		err = m.Migrate(true)
		if err != nil {
			t.Fatal(err)
		}
	}
}
