package table

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/test"
	"github.com/yaoapp/yao/widgets/component"
)

func TestProcessSearch(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	params := map[string]interface{}{
		"withs": map[string]interface{}{
			"user": map[string]interface{}{},
		},
	}

	args := []interface{}{"pet", params, 1, 5}
	res, err := process.New("yao.table.search", args...).Exec()
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

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	params := map[string]interface{}{
		"limit": 2,
		"withs": map[string]interface{}{
			"user": map[string]interface{}{},
		},
	}

	args := []interface{}{"pet", params}
	res, err := process.New("yao.table.get", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	arr := any.Of(res).CArray()
	assert.Equal(t, 2, len(arr))

	data := any.Of(arr[0]).MapStr().Dot()
	assert.Equal(t, "checked", data.Get("status"))

}

func TestProcessFind(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet", 1}
	res, err := process.New("yao.table.find", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "checked", data.Get("status"))
}

func TestProcessSave(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
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

	res, err := process.New("yao.table.Save", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "4", fmt.Sprintf("%v", res))

	res, err = process.New("yao.table.find", "pet", res).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessCreate(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
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

	res, err := process.New("yao.table.Create", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "6", fmt.Sprintf("%v", res))

	res, err = process.New("yao.table.find", "pet", res).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessUpdate(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
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

	_, err := process.New("yao.table.Update", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := process.New("yao.table.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessUpdateWhere(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
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

	_, err := process.New("yao.table.UpdateWhere", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := process.New("yao.table.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessUpdateIn(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
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

	_, err := process.New("yao.table.UpdateIn", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := process.New("yao.table.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessInsert(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)

	args := []interface{}{"pet",
		[]string{"name", "type", "status", "mode", "stay", "cost", "doctor_id"},
		[][]interface{}{
			{"Cookie", "cat", "checked", "enabled", 200, 105, 1},
			{"Baby", "dog", "checked", "enabled", 186, 24, 1},
			{"Poo", "others", "checked", "enabled", 199, 66, 1},
		},
	}

	_, err := process.New("yao.table.Insert", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := process.New("yao.table.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "Cookie", data.Get("name"))
}

func TestProcessDelete(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet", 1}

	_, err := process.New("yao.table.Delete", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	_, err = process.New("yao.table.find", "pet", 1).Exec()
	assert.Contains(t, err.Error(), "ID=1")
}

func TestProcessDeleteWhere(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{
		"pet",
		map[string]interface{}{"wheres": []map[string]interface{}{{"column": "id", "value": 1}}},
	}

	_, err := process.New("yao.table.DeleteWhere", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	_, err = process.New("yao.table.find", "pet", 1).Exec()
	assert.Contains(t, err.Error(), "ID=1")
}

func TestProcessDeleteIn(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet", "1"}

	_, err := process.New("yao.table.DeleteIn", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	_, err = process.New("yao.table.find", "pet", 1).Exec()
	assert.Contains(t, err.Error(), "ID=1")
}

func TestProcessComponent(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{
		"pet",
		"fields.filter.状态.edit.props.xProps",
		"remote",
	}

	res, err := process.New("yao.table.Component", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	pets, ok := res.([]component.Option)
	assert.True(t, ok)
	assert.Equal(t, 2, len(pets))
	assert.Equal(t, "Cookie", pets[0].Label)
	assert.Equal(t, "checked", pets[0].Value)
	assert.Equal(t, "Baby", pets[1].Label)
	assert.Equal(t, "checked", pets[1].Value)
}

func TestProcessComponentError(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{
		"pet",
		"fields.filter.edit.props.状态.::not-exist",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}
	_, err := process.New("yao.table.Component", args...).Exec()
	assert.Contains(t, err.Error(), "fields.filter.edit.props.状态.::not-exist")
}

func TestProcessUpload(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{
		"pet",
		"fields.table.相关图片.edit.props",
		"api",
		types.UploadFile{TempFile: tempFile(t)},
	}

	res, err := process.New("yao.table.Upload", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	file, ok := res.(string)
	assert.True(t, ok)
	assert.NotEmpty(t, file)
}

func TestProcessDownload(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	jwt := helper.JwtMake(1, map[string]interface{}{"id": 1}, map[string]interface{}{"sid": 1})
	fs := fs.MustGet("system")
	_, err := fs.WriteFile("/text.txt", []byte("Hello"), uint32(os.ModePerm))
	if err != nil {
		t.Fatal(err)
	}

	args := []interface{}{"pet", "images", "/text.txt", jwt.Token}
	res, err := process.New("yao.table.Download", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	body, ok := res.(map[string]interface{})
	reader, ok := body["content"].(io.ReadCloser)
	if !ok {
		t.Fatal("content not found")
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, ok)
	assert.Equal(t, []byte("Hello"), content)
	assert.Equal(t, "text/plain; charset=utf-8", body["type"])
}

func TestProcessSetting(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet"}
	res, err := process.New("yao.table.Setting", args...).Exec()
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
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet"}
	res, err := process.New("yao.table.Xgen", args...).Exec()
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

func TestProcessXgenWithPermissions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	session.Global().Set("__permissions", map[string]interface{}{
		"tables.pet": []string{
			"8ca9bdf0fa2cbc8f1018f8566ed6ab5e", // fields.table.消费金额
			"c5b1f06582e1dff3ac6d16822fdadd54", // fields.filter.状态
			"b1483ade34cd51261817558114e74e3f", // filter.actions[0] 添加宠物
			"e6a67850312980e8372e550c5b361097", // operation.actions[0] 查看
		},
	})

	args := []interface{}{"pet"}
	res, err := process.New("yao.table.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/xiang/import/pet", data.Get("header.preset.import.api.import"))
	assert.Equal(t, "查看详情1", data.Get("header.preset.import.actions[0].title"))
	assert.Equal(t, "查看详情2", data.Get("header.preset.import.actions[1].title"))
	assert.False(t, data.Has("fields.table.消费金额"))
	assert.False(t, data.Has("fields.filter.状态"))
	assert.False(t, data.Has("filter.actions[0]"))
	assert.Len(t, data.Get("table.columns"), 3)
	assert.Len(t, data.Get("table.operation.actions"), 4)

	session.Global().Set("__permissions", nil)
	res, err = process.New("yao.table.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data = any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/xiang/import/pet", data.Get("header.preset.import.api.import"))
	assert.Equal(t, "查看详情1", data.Get("header.preset.import.actions[0].title"))
	assert.Equal(t, "查看详情2", data.Get("header.preset.import.actions[1].title"))
	assert.True(t, data.Has("fields.table.消费金额"))
	assert.True(t, data.Has("fields.filter.状态"))
	assert.True(t, data.Has("filter.actions[0]"))
	assert.Len(t, data.Get("table.columns"), 4)
	assert.Len(t, data.Get("table.operation.actions"), 6)

}

func TestProcessExport(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet", model.QueryParam{Wheres: []model.QueryWhere{{Column: "mode", Value: "enabled"}}}, 2}
	response := process.New("yao.table.Export", args...).Run()
	assert.NotNil(t, response)
	fs := fs.MustGet("system")
	size, _ := fs.Size(response.(string))
	assert.Greater(t, size, 1000)

	// Export all data
	args = []interface{}{"pet", nil, 2}
	response = process.New("yao.table.Export", args...).Run()
	assert.NotNil(t, response)
	size, _ = fs.Size(response.(string))
	assert.Greater(t, size, 1000)
}

func TestProcessLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	source := `{
		"name": "Pet Admin Bind Model And Form",
		"action": {
		  "bind": { "model": "pet", "option": { "form": "pet" } },
		  "search": {
			"guard": "-",
			"process": "scripts.pet.Search",
			"default": [null, 1, 5]
		  }
		}
	  }
	`
	args := []interface{}{"dynamic.pet", "/tables/dynamic/pet.tab.yao", source}

	// Load
	assert.NotPanics(t, func() {
		process.New("yao.table.Load", args...).Run()
	})
	tab := MustGet("dynamic.pet")
	assert.Equal(t, "Pet Admin Bind Model And Form", tab.Name)
	assert.Equal(t, "pet", tab.Action.Bind.Model)

	// Exist
	res := process.New("yao.table.Exists", "dynamic.pet").Run()
	assert.True(t, res.(bool))

	// Reload
	assert.NotPanics(t, func() {
		process.New("yao.table.Reload", "dynamic.pet").Run()
	})
	tab = MustGet("dynamic.pet")
	assert.Equal(t, "Pet Admin Bind Model And Form", tab.Name)
	assert.Equal(t, "pet", tab.Action.Bind.Model)

	// Unload
	assert.NotPanics(t, func() {
		process.New("yao.table.Unload", "dynamic.pet").Run()
	})
	res = process.New("yao.table.Exists", "dynamic.pet").Run()
	assert.False(t, res.(bool))
}

func TestProcessRead(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	res := process.New("yao.table.Read", "pet").Run()
	assert.NotNil(t, res)
	assert.Equal(t, "::Pet Admin", res.(map[string]interface{})["name"])
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
