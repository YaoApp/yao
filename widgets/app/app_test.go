package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/lang"
	"github.com/yaoapp/yao/widgets/login"
)

func TestLoad(t *testing.T) {
	os.Unsetenv("YAO_LANG")
	lang.Load(config.Conf)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Demo Application", Setting.Name)
	assert.Equal(t, "Demo", Setting.Short)
	assert.Equal(t, "Another yao application", Setting.Description)
	assert.Equal(t, []string{"demo"}, Setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", Setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional.HideNotification)
	assert.Equal(t, false, Setting.Optional.HideSetting)
}

func TestLoadHK(t *testing.T) {

	os.Setenv("YAO_LANG", "zh-hk")
	lang.Load(config.Conf)

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "示例應用", Setting.Name)
	assert.Equal(t, "演示", Setting.Short)
	assert.Equal(t, "又一個YAO應用", Setting.Description)
	assert.Equal(t, []string{"demo"}, Setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", Setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional.HideNotification)
	assert.Equal(t, false, Setting.Optional.HideSetting)
}

func TestLoadCN(t *testing.T) {

	os.Setenv("YAO_LANG", "zh-cn")
	lang.Load(config.Conf)

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "示例应用", Setting.Name)
	assert.Equal(t, "演示", Setting.Short)
	assert.Equal(t, "又一个 YAO 应用", Setting.Description)
	assert.Equal(t, []string{"demo"}, Setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", Setting.Menu.Process)
	assert.Equal(t, true, Setting.Optional.HideNotification)
	assert.Equal(t, false, Setting.Optional.HideSetting)
}

func TestExport(t *testing.T) {

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

	api, has := gou.APIs["widgets.app"]
	assert.True(t, has)
	assert.Equal(t, 2, len(api.HTTP.Paths))

	_, has = gou.ThirdHandlers["yao.app.setting"]
	assert.True(t, has)

	_, has = gou.ThirdHandlers["yao.app.xgen"]
	assert.True(t, has)

	_, has = gou.ThirdHandlers["yao.app.menu"]
	assert.True(t, has)
}

func TestProcessSetting(t *testing.T) {
	loadApp(t)
	res, err := gou.NewProcess("yao.app.Setting").Exec()
	if err != nil {
		t.Fatal(err)
	}

	setting, ok := res.(DSL)
	assert.True(t, ok)
	assert.Equal(t, "Demo Application", setting.Name)
	assert.Equal(t, "Demo", setting.Short)
	assert.Equal(t, "Another yao application", setting.Description)
	assert.Equal(t, []string{"demo"}, setting.Menu.Args)
	assert.Equal(t, "flows.app.menu", setting.Menu.Process)
	assert.Equal(t, true, setting.Optional.HideNotification)
	assert.Equal(t, false, setting.Optional.HideSetting)
}

func TestProcessXgen(t *testing.T) {
	loadApp(t)
	res, err := gou.NewProcess("yao.app.Xgen").Exec()
	if err != nil {
		t.Fatal(err)
	}

	xgen := any.Of(res).MapStr().Dot()
	assert.Equal(t, "__yao", xgen.Get("apiPrefix"))
	assert.Equal(t, "Another yao application", xgen.Get("description"))
	assert.Equal(t, "/api/__yao/login/admin/captcha?type=digit", xgen.Get("login.admin.captcha"))
	assert.Equal(t, "/api/__yao/login/admin", xgen.Get("login.admin.login"))
	assert.Equal(t, "/x/Table/pet", xgen.Get("login.entry.admin"))
	assert.Equal(t, "/x/Table/dash", xgen.Get("login.entry.user"))
	assert.Equal(t, "/api/__yao/login/user/captcha?type=digit", xgen.Get("login.user.captcha"))
	assert.Equal(t, "/api/__yao/login/user", xgen.Get("login.user.login"))
	assert.Equal(t, os.Getenv("YAO_ENV"), xgen.Get("mode"))
	assert.Equal(t, "Demo Application", xgen.Get("name"))
	assert.Equal(t, true, xgen.Get("optional.hideNotification"))
	assert.Equal(t, "localStorage", xgen.Get("token"))
}

func loadApp(t *testing.T) {

	os.Setenv("YAO_LANG", "")
	lang.Load(config.Conf)

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

}
