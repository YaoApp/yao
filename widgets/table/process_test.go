package table

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	q "github.com/yaoapp/yao/query"
)

func TestProcessSearch(t *testing.T) {

	load(t)
	clear(t)
	testData(t)

	params := map[string]interface{}{
		"withs": map[string]interface{}{
			"user": map[string]interface{}{},
		},
	}

	args := []interface{}{"pet", params, 1, 5}
	res, err := gou.NewProcess("yao.table.search", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "AfterSearch", data.Get("after:hook"))
	assert.Equal(t, "1", fmt.Sprintf("%v", data.Get("pagesize")))
	assert.Equal(t, "3", fmt.Sprintf("%v", data.Get("total")))
	assert.Equal(t, "checked", data.Get("data.0.status"))
}

func TestProcessGet(t *testing.T) {

	load(t)
	clear(t)
	testData(t)

	params := map[string]interface{}{
		"limit": 2,
		"withs": map[string]interface{}{
			"user": map[string]interface{}{},
		},
	}

	args := []interface{}{"pet", params}
	res, err := gou.NewProcess("yao.table.get", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	arr := any.Of(res).CArray()
	assert.Equal(t, 2, len(arr))

	data := any.Of(arr[0]).MapStr().Dot()
	assert.Equal(t, "checked", data.Get("status"))

}

func TestProcessFind(t *testing.T) {

	load(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet", 1}
	res, err := gou.NewProcess("yao.table.find", args...).Exec()
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

	res, err := gou.NewProcess("yao.table.Save", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "4", fmt.Sprintf("%v", res))

	res, err = gou.NewProcess("yao.table.find", "pet", res).Exec()
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

	res, err := gou.NewProcess("yao.table.Create", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "6", fmt.Sprintf("%v", res))

	res, err = gou.NewProcess("yao.table.find", "pet", res).Exec()
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

	_, err := gou.NewProcess("yao.table.Update", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := gou.NewProcess("yao.table.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessUpdateWhere(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet",
		map[string]interface{}{"wheres": []map[string]interface{}{{"column": "id", "value": 1}}},
		map[string]interface{}{
			"name":      "New Pet",
			"type":      "cat",
			"status":    "checked",
			"mode":      "enabled",
			"stay":      66,
			"cost":      24,
			"doctor_id": 1,
		}}

	_, err := gou.NewProcess("yao.table.UpdateWhere", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := gou.NewProcess("yao.table.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessUpdateIn(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", "1",
		map[string]interface{}{
			"name":      "New Pet",
			"type":      "cat",
			"status":    "checked",
			"mode":      "enabled",
			"stay":      66,
			"cost":      24,
			"doctor_id": 1,
		}}

	_, err := gou.NewProcess("yao.table.UpdateIn", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := gou.NewProcess("yao.table.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessInsert(t *testing.T) {
	load(t)
	clear(t)
	args := []interface{}{"pet",
		[]string{"name", "type", "status", "mode", "stay", "cost", "doctor_id"},
		[][]interface{}{
			{"Cookie", "cat", "checked", "enabled", 200, 105, 1},
			{"Baby", "dog", "checked", "enabled", 186, 24, 1},
			{"Poo", "others", "checked", "enabled", 199, 66, 1},
		},
	}

	_, err := gou.NewProcess("yao.table.Insert", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := gou.NewProcess("yao.table.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "Cookie", data.Get("name"))
}

func TestProcessDelete(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", 1}

	_, err := gou.NewProcess("yao.table.Delete", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	_, err = gou.NewProcess("yao.table.find", "pet", 1).Exec()
	assert.Contains(t, err.Error(), "ID=1")
}

func TestProcessDeleteWhere(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{
		"pet",
		map[string]interface{}{"wheres": []map[string]interface{}{{"column": "id", "value": 1}}},
	}

	_, err := gou.NewProcess("yao.table.DeleteWhere", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	_, err = gou.NewProcess("yao.table.find", "pet", 1).Exec()
	assert.Contains(t, err.Error(), "ID=1")
}

func TestProcessDeleteIn(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", "1"}

	_, err := gou.NewProcess("yao.table.DeleteIn", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	_, err = gou.NewProcess("yao.table.find", "pet", 1).Exec()
	assert.Contains(t, err.Error(), "ID=1")
}

func TestProcessComponent(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{
		"pet",
		"fields.filter.状态.edit.props.xProps",
		"remote",
		map[string]interface{}{
			"model":    "pet",
			"label":    "name",
			"value":    "status",
			"wheres[]": `{"column":"id","op":"ge","value":0}`,
			"limit":    "2",
		},
	}

	res, err := gou.NewProcess("yao.table.Component", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	pets, ok := res.([]map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(pets))
	assert.Equal(t, "Cookie", pets[0]["label"])
	assert.Equal(t, "checked", pets[0]["value"])
	assert.Equal(t, "Baby", pets[1]["label"])
	assert.Equal(t, "checked", pets[1]["value"])
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
	_, err := gou.NewProcess("yao.table.Component", args...).Exec()
	assert.Contains(t, err.Error(), "fields.filter.edit.props.状态.::not-exist")
}

func TestProcessUpload(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{
		"pet",
		"fields.table.相关图片.edit.props",
		"api",
		gou.UploadFile{TempFile: tempFile(t)},
	}

	res, err := gou.NewProcess("yao.table.Upload", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	file, ok := res.(string)
	assert.True(t, ok)
	assert.NotEmpty(t, file)
}

func TestProcessDownload(t *testing.T) {
	load(t)
	clear(t)
	testData(t)

	jwt := helper.JwtMake(1, map[string]interface{}{"id": 1}, map[string]interface{}{"sid": 1})
	fs := fs.MustGet("system")
	_, err := fs.WriteFile("/text.txt", []byte("Hello"), uint32(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}

	args := []interface{}{"pet", "images", "/text.txt", jwt.Token}
	res, err := gou.NewProcess("yao.table.Download", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	body, ok := res.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, []byte("Hello"), body["content"])
	assert.Equal(t, "text/plain; charset=utf-8", body["type"])
}

func TestProcessSetting(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet"}
	res, err := gou.NewProcess("yao.table.Setting", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/xiang/import/pet", data.Get("header.preset.import.api.import"))
	assert.Equal(t, "查看详情1", data.Get("header.preset.import.actions[0].title"))
	assert.Equal(t, "查看详情2", data.Get("header.preset.import.actions[1].title"))
	assert.Equal(t, "/api/__yao/table/pet/component/fields.table."+url.QueryEscape("入院状态")+".view.props.xProps/remote", data.Get("fields.table.入院状态.view.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/table/pet/component/fields.table."+url.QueryEscape("入院状态")+".edit.props.xProps/remote", data.Get("fields.table.入院状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/table/pet/upload/fields.table."+url.QueryEscape("相关图片")+".edit.props/api", data.Get("fields.table.相关图片.edit.props.api"))
}

func TestProcessXgen(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet"}
	res, err := gou.NewProcess("yao.table.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/xiang/import/pet", data.Get("header.preset.import.api.import"))
	assert.Equal(t, "查看详情1", data.Get("header.preset.import.actions[0].title"))
	assert.Equal(t, "查看详情2", data.Get("header.preset.import.actions[1].title"))
	assert.Equal(t, "/api/__yao/table/pet/component/fields.table."+url.QueryEscape("入院状态")+".view.props.xProps/remote", data.Get("fields.table.入院状态.view.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/table/pet/component/fields.table."+url.QueryEscape("入院状态")+".edit.props.xProps/remote", data.Get("fields.table.入院状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/table/pet/upload/fields.table."+url.QueryEscape("相关图片")+".edit.props/api", data.Get("fields.table.相关图片.edit.props.api"))

}

func TestProcessExport(t *testing.T) {
	load(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", gou.QueryParam{Wheres: []gou.QueryWhere{{Column: "mode", Value: "enabled"}}}, 2}
	response := gou.NewProcess("yao.table.Export", args...).Run()
	assert.NotNil(t, response)
	fs := fs.MustGet("system")
	size, _ := fs.Size(response.(string))
	assert.Greater(t, size, 1000)
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

func tempFile(t *testing.T) string {
	file, err := os.CreateTemp("", "unit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	_, err = file.Write([]byte("HELLO"))
	if err != nil {
		t.Fatal(err)
	}

	return file.Name()
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
