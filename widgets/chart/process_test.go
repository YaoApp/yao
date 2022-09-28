package chart

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	q "github.com/yaoapp/yao/query"
)

func TestProcessData(t *testing.T) {
	load(t)
	args := []interface{}{"dashboard", map[string]interface{}{"range": "2022-01-02", "status": "checked"}}
	res, err := gou.NewProcess("yao.chart.Data", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr()
	assert.Equal(t, 14, len(data))
}

func TestProcessComponent(t *testing.T) {
	load(t)
	args := []interface{}{
		"dashboard",
		"fields.filter.状态.edit.props.xProps",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}

	res, err := gou.NewProcess("yao.chart.Component", args...).Exec()
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
	load(t)
	args := []interface{}{
		"dashboard",
		"fields.filter.edit.props.状态.::not-exist",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}
	_, err := gou.NewProcess("yao.chart.Component", args...).Exec()
	assert.Contains(t, err.Error(), "fields.filter.edit.props.状态.::not-exist")
}

func TestProcessSetting(t *testing.T) {
	load(t)
	args := []interface{}{"dashboard"}
	res, err := gou.NewProcess("yao.chart.Setting", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/chart/dashboard/component/fields.filter.状态.edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
}

func TestProcessXgen(t *testing.T) {
	load(t)
	args := []interface{}{"dashboard"}
	res, err := gou.NewProcess("yao.chart.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/chart/dashboard/component/fields.filter.状态.edit.props.xProps/remote", data.Get("fields.filter.状态.edit.props.xProps.remote.api"))
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
