package form

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	q "github.com/yaoapp/yao/query"
)

func TestProcessFind(t *testing.T) {

	load(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet", 1}
	res, err := gou.NewProcess("yao.form.find", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "checked", data.Get("status"))
}

func TestProcessSave(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", map[string]interface{}{
		"name":      "New Pet",
		"type":      "cat",
		"status":    "checked",
		"mode":      "enabled",
		"stay":      66,
		"cost":      24,
		"doctor_id": 1,
	}}

	res, err := gou.NewProcess("yao.form.Save", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "4", fmt.Sprintf("%v", res))

	res, err = gou.NewProcess("yao.form.find", "pet", res).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessCreate(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", map[string]interface{}{
		"id":        6,
		"name":      "New Pet",
		"type":      "cat",
		"status":    "checked",
		"mode":      "enabled",
		"stay":      66,
		"cost":      24,
		"doctor_id": 1,
	}}

	res, err := gou.NewProcess("yao.form.Create", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "6", fmt.Sprintf("%v", res))

	res, err = gou.NewProcess("yao.form.find", "pet", res).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessUpdate(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", 1, map[string]interface{}{
		"name":      "New Pet",
		"type":      "cat",
		"status":    "checked",
		"mode":      "enabled",
		"stay":      66,
		"cost":      24,
		"doctor_id": 1,
	}}

	_, err := gou.NewProcess("yao.form.Update", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := gou.NewProcess("yao.form.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessDelete(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", 1}

	_, err := gou.NewProcess("yao.form.Delete", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	_, err = gou.NewProcess("yao.form.find", "pet", 1).Exec()
	assert.Contains(t, err.Error(), "ID=1")
}

func TestProcessComponent(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{
		"pet",
		"fields.form.状态.edit.props.xProps",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}

	res, err := gou.NewProcess("yao.form.Component", args...).Exec()
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
	clear(t)
	testData(t)
	args := []interface{}{
		"pet",
		"fields.filter.edit.props.状态.::not-exist",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}
	_, err := gou.NewProcess("yao.form.Component", args...).Exec()
	assert.Contains(t, err.Error(), "fields.filter.edit.props.状态.::not-exist")
}

func TestProcessSetting(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet"}
	res, err := gou.NewProcess("yao.form.Setting", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.form.状态.edit.props.xProps.remote.api"))
}

func TestProcessXgen(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet"}
	res, err := gou.NewProcess("yao.form.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.form.状态.edit.props.xProps.remote.api"))
}

func load(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	q.Load(config.Conf)
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
