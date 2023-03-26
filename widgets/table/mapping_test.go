package table

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestMapping(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	tab, err := Get("compute")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 11, len(tab.Computes.View))
	assert.Equal(t, 1, len(tab.Computes.View["created_at"]))
	assert.Equal(t, 1, len(tab.Computes.View["stay"]))
	assert.Equal(t, 4, len(tab.Computes.Edit))
	assert.Equal(t, 1, len(tab.Computes.Edit["stay"]))
	assert.Equal(t, 2, len(tab.Computes.Filter))
	assert.Equal(t, 2, len(tab.Computes.Filter["where.name.like"]))
}

func TestMappingFind(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"compute", 1}
	res, err := process.New("yao.table.find", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "cat::Cookie-checked-compute", data.Get("name_view"))
}

func TestMappingGet(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	params := map[string]interface{}{"limit": 2}

	args := []interface{}{"compute", params}
	res, err := process.New("yao.table.get", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	arr := any.Of(res).CArray()
	data := any.Of(arr[0]).MapStr().Dot()
	assert.Equal(t, "cat::Cookie-checked-compute", data.Get("name_view"))

}

func TestMappingSearch(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	params := map[string]interface{}{}
	args := []interface{}{"compute", params, 1, 5}
	res, err := process.New("yao.table.search", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "cat::Cookie-checked-compute", data.Get("data.0.name_view"))
}

func TestMappingSave(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
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

	res, err := process.New("yao.table.Save", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "4", fmt.Sprintf("%v", res))

	res, err = process.New("yao.table.find", "compute", res).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestMappingUpdate(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
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

	_, err := process.New("yao.table.Update", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := process.New("yao.table.find", "compute", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestMappingInsert(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)

	args := []interface{}{"compute",
		[]string{"name", "type", "status", "mode", "stay", "cost", "doctor_id"},
		[][]interface{}{
			{"  Cookie  ", "cat", "checked", "enabled", 200, 105, 1},
			{"Baby", "dog", "checked", "enabled", 186, 24, 1},
			{"Poo", "others", "checked", "enabled", 199, 66, 1},
		},
	}

	_, err := process.New("yao.table.Insert", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := process.New("yao.table.find", "compute", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "Cookie", data.Get("name"))
}
