package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/test"
	_ "github.com/yaoapp/yao/utils"
	"github.com/yaoapp/yao/widgets/login"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "::Demo Application", Setting.Name)
	assert.Equal(t, "::Demo", Setting.Short)
	assert.Equal(t, "::Another yao application", Setting.Description)
	assert.Equal(t, []interface{}{"demo"}, Setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", Setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional["hideNotification"])
	assert.Equal(t, false, Setting.Optional["hideSetting"])
}

func TestLoadHK(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := i18n.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	newSetting, err := i18n.Trans("zh-hk", []string{"app.app"}, Setting)
	if err != nil {
		t.Fatal(err)
	}
	setting := newSetting.(*DSL)

	assert.Equal(t, "示例應用", setting.Name)
	assert.Equal(t, "演示", setting.Short)
	assert.Equal(t, "又一個YAO應用", setting.Description)
	assert.Equal(t, []interface{}{"demo"}, setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional["hideNotification"])
	assert.Equal(t, false, Setting.Optional["hideSetting"])

	assert.Equal(t, "::Demo Application", Setting.Name)
	assert.Equal(t, "::Demo", Setting.Short)
	assert.Equal(t, "::Another yao application", Setting.Description)
	assert.Equal(t, []interface{}{"demo"}, Setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", Setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional["hideNotification"])
	assert.Equal(t, false, Setting.Optional["hideSetting"])
}

func TestLoadCN(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := i18n.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	newSetting, err := i18n.Trans("zh-cn", []string{"app.app"}, Setting)
	if err != nil {
		t.Fatal(err)
	}
	setting := newSetting.(*DSL)

	assert.Equal(t, "示例应用", setting.Name)
	assert.Equal(t, "演示", setting.Short)
	assert.Equal(t, "又一个 YAO 应用", setting.Description)
	assert.Equal(t, []interface{}{"demo"}, setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional["hideNotification"])
	assert.Equal(t, false, Setting.Optional["hideSetting"])

	assert.Equal(t, "::Demo Application", Setting.Name)
	assert.Equal(t, "::Demo", Setting.Short)
	assert.Equal(t, "::Another yao application", Setting.Description)
	assert.Equal(t, []interface{}{"demo"}, Setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", Setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional["hideNotification"])
	assert.Equal(t, false, Setting.Optional["hideSetting"])
}

func TestExport(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := login.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = Export()
	if err != nil {
		t.Fatal(err)
	}

	// api, has := gou.APIs["widgets.app"]
	// assert.True(t, has)
	// assert.Equal(t, 7, len(api.HTTP.Paths))

	// _, has = gou.ThirdHandlers["yao.app.setting"]
	// assert.True(t, has)

	// _, has = gou.ThirdHandlers["yao.app.xgen"]
	// assert.True(t, has)

	// _, has = gou.ThirdHandlers["yao.app.menu"]
	// assert.True(t, has)

	// _, has = gou.ThirdHandlers["yao.app.check"]
	// assert.True(t, has)

	// _, has = gou.ThirdHandlers["yao.app.setup"]
	// assert.True(t, has)

	// _, has = gou.ThirdHandlers["yao.app.service"]
	// assert.True(t, has)
}

func TestProcessSetting(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	loadApp(t)
	backup := config.Conf.Lang
	config.Conf.Lang = "en-us"
	res, err := process.New("yao.app.Setting").Exec()
	if err != nil {
		t.Fatal(err)
	}

	setting, ok := res.(DSL)
	assert.True(t, ok)
	assert.Equal(t, "Demo Application", setting.Name)
	assert.Equal(t, "Demo", setting.Short)
	assert.Equal(t, "Another yao application", setting.Description)
	assert.Equal(t, []interface{}{"demo"}, setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional["hideNotification"])
	assert.Equal(t, false, Setting.Optional["hideSetting"])
	assert.Equal(t, true, setting.Sid != "")

	// Set
	res, err = process.New("yao.app.Setting", map[string]interface{}{"lang": "zh-hk", "sid": setting.Sid}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	setting2, ok := res.(DSL)
	assert.Equal(t, setting.Sid, setting2.Sid)

	p := process.New("yao.app.Setting").WithSID(setting.Sid)
	lang := session.Lang(p)
	assert.Equal(t, "zh-hk", lang)

	config.Conf.Lang = backup
}

func TestProcessXgen(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	loadApp(t)
	backup := config.Conf.Lang
	config.Conf.Lang = "en-us"
	res, err := process.New("yao.app.Xgen").Exec()
	if err != nil {
		t.Fatal(err)
	}

	xgen := any.Of(res).MapStr().Dot()
	assert.Equal(t, "__yao", xgen.Get("apiPrefix"))
	assert.Equal(t, "Another yao application", xgen.Get("description"))
	assert.Equal(t, "/api/__yao/login/admin/captcha?type=digit", xgen.Get("login.admin.captcha"))
	assert.Equal(t, "/api/__yao/login/admin", xgen.Get("login.admin.login"))
	assert.Equal(t, "/x/Chart/dashboard", xgen.Get("login.entry.admin"))
	assert.Equal(t, "/x/Table/pet", xgen.Get("login.entry.user"))
	assert.Equal(t, "/api/__yao/login/user/captcha?type=digit", xgen.Get("login.user.captcha"))
	assert.Equal(t, "/api/__yao/login/user", xgen.Get("login.user.login"))
	assert.Equal(t, "/assets/images/login/cover.svg", xgen.Get("login.user.layout.cover"))
	assert.Equal(t, "/assets/images/login/cover.svg", xgen.Get("login.admin.layout.cover"))
	assert.Equal(t, "/api/__yao/app/icons/app.ico", xgen.Get("favicon"))
	assert.Equal(t, "/api/__yao/app/icons/app.png", xgen.Get("logo"))
	assert.Equal(t, os.Getenv("YAO_ENV"), xgen.Get("mode"))
	assert.Equal(t, "Demo Application", xgen.Get("name"))
	assert.Equal(t, true, xgen.Get("optional.hideNotification"))
	// assert.Equal(t, "localStorage", xgen.Get("token"))
	assert.Equal(t, true, xgen.Get("sid").(string) != "")

	// Set
	res, err = process.New("yao.app.Xgen", map[string]interface{}{"lang": "zh-hk", "sid": xgen.Get("sid")}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	xgen2 := any.Of(res).MapStr().Dot()
	assert.Equal(t, xgen.Get("sid"), xgen2.Get("sid"))

	p := process.New("yao.app.Setting").WithSID(xgen2.Get("sid").(string))
	lang := session.Lang(p)
	assert.Equal(t, "zh-hk", lang)
	config.Conf.Lang = backup
}

func TestProcessMenu(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	loadApp(t)
	res, err := process.New("yao.app.Menu").Exec()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(res).MapStr()
	assert.True(t, data.Has("items"))
	assert.True(t, data.Has("setting"))
}

func TestProcessIcons(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	loadApp(t)
	res, err := process.New("yao.app.Icons", "app.png").Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(res.(string)), 10)
}

func TestProcessCheck(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	loadApp(t)
	res, err := process.New("yao.app.Check", map[string]interface{}{}).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	_, err = process.New("yao.app.Check", map[string]interface{}{"error": "1"}).Exec()
	assert.NotNil(t, err)
}

func TestProcessSetup(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	loadApp(t)
	res, err := process.New("yao.app.Setup", map[string]interface{}{"sid": "hello"}).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "http://127.0.0.1:5099/admin/", res.(map[string]interface{})["admin"])
	_, err = process.New("yao.app.Setup", map[string]interface{}{"error": "1"}).Exec()
	assert.NotNil(t, err)
}

func TestProcessService(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	loadApp(t)
	res, err := process.New(
		"yao.app.Service",
		"foo",
		map[string]interface{}{"method": "Bar", "args": []interface{}{"hello", "world"}},
	).Exec()

	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []interface{}{"hello", "world"}, res.(map[string]interface{})["args"])
}

func loadApp(t *testing.T) {

	err := script.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = i18n.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	lang.Pick("en-us").AsDefault()

	err = login.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = flow.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = Export()
	if err != nil {
		t.Fatal(err)
	}

}
