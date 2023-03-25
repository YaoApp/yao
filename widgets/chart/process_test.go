package chart

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessData(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"dashboard", map[string]interface{}{"range": "2022-01-02", "status": "checked"}}
	res, err := process.New("yao.chart.Data", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr()
	assert.Equal(t, 14, len(data))
}

func TestProcessComponent(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{
		"dashboard",
		"fields.filter.状态.edit.props.xProps",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}

	res, err := process.New("yao.chart.Component", args...).Exec()
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
}

func TestProcessComponentError(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{
		"dashboard",
		"fields.filter.edit.props.状态.::not-exist",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}
	_, err := process.New("yao.chart.Component", args...).Exec()
	assert.Contains(t, err.Error(), "fields.filter.edit.props.状态.::not-exist")
}

func TestProcessSetting(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"dashboard"}
	res, err := process.New("yao.chart.Setting", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/chart/dashboard/component/fields.filter."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
}

func TestProcessXgen(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"dashboard"}
	res, err := process.New("yao.chart.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/chart/dashboard/component/fields.filter."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
}

func TestProcessXgenWithPermissions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	session.Global().Set("__permissions", map[string]interface{}{
		"charts.dashboard": []string{
			"7f46a38d7ff3f1832375ff63cd412f41", // operation.actions[0] 跳转至大屏
			"09302a46b1b6f13a346deeea79b859dd", // filter.columns[0].时间区间
			"f11f01be1f77fe6563f8577806a46158", // 综合评分
		},
	})

	args := []interface{}{"dashboard"}
	res, err := process.New("yao.chart.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/chart/dashboard/component/fields.filter."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
	assert.NotEqual(t, "时间区间", data.Get("filter.columns[0].name"))
	assert.Equal(t, nil, data.Get("operation.actions[0]"))
	assert.Equal(t, nil, data.Get("fields.chart.综合评分"))

	session.Global().Set("__permissions", nil)
	res, err = process.New("yao.chart.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data = any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/chart/dashboard/component/fields.filter."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "时间区间", data.Get("filter.columns[0].name"))
	assert.NotEqual(t, nil, data.Get("operation.actions[0]"))
	assert.NotEqual(t, nil, data.Get("fields.chart.综合评分"))
}

func testData(t *testing.T) {
	pet := model.Select("pet")
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
	for _, m := range model.Models {
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
