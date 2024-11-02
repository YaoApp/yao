package form

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
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/test"
)

func TestProcessFind(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	args := []interface{}{"pet", 1}
	res, err := process.New("yao.form.find", args...).Exec()
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

	res, err := process.New("yao.form.Save", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "4", fmt.Sprintf("%v", res))

	res, err = process.New("yao.form.find", "pet", res).Exec()
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

	res, err := process.New("yao.form.Create", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "6", fmt.Sprintf("%v", res))

	res, err = process.New("yao.form.find", "pet", res).Exec()
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

	_, err := process.New("yao.form.Update", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := process.New("yao.form.find", "pet", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "New Pet", data.Get("name"))
}

func TestProcessDelete(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet", 1}

	_, err := process.New("yao.form.Delete", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	res, err := process.New("yao.form.find", "pet", 1).Exec()
	fmt.Println("err", res, err)
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
		"fields.form.状态.edit.props.xProps",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}

	res, err := process.New("yao.form.Component", args...).Exec()
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
		"pet",
		"fields.filter.edit.props.状态.::not-exist",
		"remote",
		map[string]interface{}{"select": []string{"name", "status"}, "limit": 2},
	}
	_, err := process.New("yao.form.Component", args...).Exec()
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
		"fields.form.相关图片.edit.props",
		"api",
		types.UploadFile{TempFile: tempFile(t)},
	}

	res, err := process.New("yao.form.Upload", args...).Exec()
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
	res, err := process.New("yao.form.Download", args...).Exec()
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
	res, err := process.New("yao.form.Setting", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("住院天数")+".edit.props/"+url.QueryEscape("on:change"), data.Get("hooks.onChange.住院天数.api"))
	assert.Equal(t, "开发者定义数据", data.Get("hooks.onChange.住院天数.params.extra"))
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.form.状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/form/pet/upload/fields.form."+url.QueryEscape("相关图片")+".edit.props/api", data.Get("fields.form.相关图片.edit.props.api"))
}

func TestProcessXgen(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)
	args := []interface{}{"pet"}
	res, err := process.New("yao.form.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("住院天数")+".edit.props/"+url.QueryEscape("on:change"), data.Get("hooks.onChange.住院天数.api"))
	assert.Equal(t, "开发者定义数据", data.Get("hooks.onChange.住院天数.params.extra"))
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.form.状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/form/pet/upload/fields.form."+url.QueryEscape("相关图片")+".edit.props/api", data.Get("fields.form.相关图片.edit.props.api"))
}

func TestProcessXgenWithPermissions(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	prepare(t)
	clear(t)
	testData(t)

	session.Global().Set("__permissions", map[string]interface{}{
		"forms.pet": []string{
			"b57eff5c9bac87d74e2a26596ed2b76f", // actions[0] 删除
			"773bee07c83276b4627b5bd7b99844ed", // fields.form.相关图片
		},
	})

	args := []interface{}{"pet"}
	res, err := process.New("yao.form.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("住院天数")+".edit.props/"+url.QueryEscape("on:change"), data.Get("hooks.onChange.住院天数.api"))
	assert.Equal(t, "开发者定义数据", data.Get("hooks.onChange.住院天数.params.extra"))
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.form.状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/form/pet/upload/fields.form."+url.QueryEscape("相关图片")+".edit.props/api", data.Get("fields.form.相关图片.edit.props.api"))
	assert.NotEqual(t, "删除", data.Get("actions[0].title"))

	session.Global().Set("__permissions", nil)
	res, err = process.New("yao.form.Xgen", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	data = any.Of(res).MapStr().Dot()
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("住院天数")+".edit.props/"+url.QueryEscape("on:change"), data.Get("hooks.onChange.住院天数.api"))
	assert.Equal(t, "开发者定义数据", data.Get("hooks.onChange.住院天数.params.extra"))
	assert.Equal(t, "/api/__yao/form/pet/component/fields.form."+url.QueryEscape("状态")+".edit.props.xProps/remote", data.Get("fields.form.状态.edit.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/form/pet/upload/fields.form."+url.QueryEscape("相关图片")+".edit.props/api", data.Get("fields.form.相关图片.edit.props.api"))
	assert.Equal(t, "删除", data.Get("actions[0].title"))

}

func TestProcessLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	source := `{
		"name": "Pet Admin Form Bind Model",
		"action": {
		  "bind": { "model": "pet" }
		}
	  }
	`
	args := []interface{}{"dynamic.pet", "/forms/dynamic/pet.form.yao", source}

	// Load
	assert.NotPanics(t, func() {
		process.New("yao.form.Load", args...).Run()
	})
	form := MustGet("dynamic.pet")
	assert.Equal(t, "Pet Admin Form Bind Model", form.Name)
	assert.Equal(t, "pet", form.Action.Bind.Model)

	// Exist
	res := process.New("yao.form.Exists", "dynamic.pet").Run()
	assert.True(t, res.(bool))

	// Reload
	assert.NotPanics(t, func() {
		process.New("yao.form.Reload", "dynamic.pet").Run()
	})
	form = MustGet("dynamic.pet")
	assert.Equal(t, "Pet Admin Form Bind Model", form.Name)
	assert.Equal(t, "pet", form.Action.Bind.Model)

	// Unload
	assert.NotPanics(t, func() {
		process.New("yao.form.Unload", "dynamic.pet").Run()
	})
	res = process.New("yao.form.Exists", "dynamic.pet").Run()
	assert.False(t, res.(bool))
}

func TestProcessRead(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	res := process.New("yao.form.Read", "pet").Run()
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
