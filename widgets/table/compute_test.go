package table

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
)

func TestComputeMapping(t *testing.T) {
	load(t)
	clear(t)
	testData(t)

	tab, err := Get("compute")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 11, len(tab.computes.view))
	assert.Equal(t, 1, len(tab.computes.view["created_at"]))
	assert.Equal(t, 1, len(tab.computes.view["stay"]))
	assert.Equal(t, 4, len(tab.computes.edit))
	assert.Equal(t, 1, len(tab.computes.edit["stay"]))
	assert.Equal(t, 2, len(tab.computes.filter))
	assert.Equal(t, 2, len(tab.computes.filter["where.name.like"]))
}

func TestComputeFind(t *testing.T) {

	load(t)
	clear(t)
	testData(t)

	args := []interface{}{"compute", 1}
	res, err := gou.NewProcess("yao.table.find", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "cat::Cookie-checked", data.Get("name_view"))
}

func TestComputeGet(t *testing.T) {

	load(t)
	clear(t)
	testData(t)

	params := map[string]interface{}{"limit": 2}

	args := []interface{}{"compute", params}
	res, err := gou.NewProcess("yao.table.get", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	arr := any.Of(res).CArray()
	data := any.Of(arr[0]).MapStr().Dot()
	assert.Equal(t, "cat::Cookie-checked", data.Get("name_view"))

}

func TestComputeSearch(t *testing.T) {

	load(t)
	clear(t)
	testData(t)

	params := map[string]interface{}{}
	args := []interface{}{"compute", params, 1, 5}
	res, err := gou.NewProcess("yao.table.search", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "cat::Cookie-checked", data.Get("data.0.name_view"))
}

func TestComputeSave(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"compute", map[string]interface{}{
		"name":      "  New Pet  ",
		"type":      "cat",
		"status":    "checked",
		"mode":      "enabled",
		"stay":      66,
		"cost":      24,
		"doctor_id": 1,
	}}

	res, err := gou.NewProcess("yao.table.Save", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "4", fmt.Sprintf("%v", res))

	res, err = gou.NewProcess("yao.table.find", "compute", res).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestComputeUpdate(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"compute", 1, map[string]interface{}{
		"name":      "  New Pet  ",
		"type":      "cat",
		"status":    "checked",
		"mode":      "enabled",
		"stay":      66,
		"cost":      24,
		"doctor_id": 1,
	}}

	_, err := gou.NewProcess("yao.table.Update", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := gou.NewProcess("yao.table.find", "compute", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestComputeInsert(t *testing.T) {
	load(t)
	clear(t)
	args := []interface{}{"compute",
		[]string{"name", "type", "status", "mode", "stay", "cost", "doctor_id"},
		[][]interface{}{
			{"  Cookie  ", "cat", "checked", "enabled", 200, 105, 1},
			{"Baby", "dog", "checked", "enabled", 186, 24, 1},
			{"Poo", "others", "checked", "enabled", 199, 66, 1},
		},
	}

	_, err := gou.NewProcess("yao.table.Insert", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := gou.NewProcess("yao.table.find", "compute", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "Cookie", data.Get("name"))
}
