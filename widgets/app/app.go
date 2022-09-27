package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/login"
)

//
// API:
//   GET /api/__yao/app/setting 	-> Default process: yao.app.Xgen
//   GET /api/__yao/app/menu  		-> Default process: yao.app.Menu
//
// Process:
// 	 yao.app.Setting Return the App DSL
// 	 yao.app.Xgen Return the Xgen setting ( merge app & login )
//   yao.app.Menu Return the menu list
//

// Setting the application setting
var Setting *DSL

// LoadAndExport load app
func LoadAndExport(cfg config.Config) error {
	err := Load(cfg)
	if err != nil {
		return err
	}
	return Export()
}

// Load the app DSL
func Load(cfg config.Config) error {

	file := filepath.Join(cfg.Root, "app.json")
	file, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	dsl := &DSL{}
	err = jsoniter.Unmarshal(data, dsl)
	if err != nil {
		return nil
	}

	// Apply a language pack
	if lang.Default != nil {
		lang.Default.Apply(dsl)
	}

	Setting = dsl
	return nil
}

// exportAPI export login api
func exportAPI() error {

	if Setting == nil {
		return fmt.Errorf("the app does not init")
	}

	http := gou.HTTP{
		Name:        "Widget App API",
		Description: "Widget App API",
		Version:     share.VERSION,
		Guard:       "bearer-jwt",
		Group:       "__yao/app",
		Paths:       []gou.Path{},
	}

	process := "yao.app.Xgen"
	if Setting.Optional.Setting != "" {
		process = Setting.Optional.Setting
	}

	path := gou.Path{
		Label:       "App Setting",
		Description: "App Setting",
		Guard:       "-",
		Path:        "/setting",
		Method:      "GET",
		Process:     process,
		In:          []string{},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	process = "yao.app.Menu"
	args := []string{}
	if Setting.Menu.Process != "" {
		process = Setting.Menu.Process
		args = Setting.Menu.Args
	}

	path = gou.Path{
		Label:       "App Menu",
		Description: "App Menu",
		Path:        "/menu",
		Method:      "GET",
		Process:     process,
		In:          args,
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = gou.LoadAPIReturn(string(source), "widgets.app")
	return err
}

// Export export login api
func Export() error {
	exportProcess()
	return exportAPI()
}

func exportProcess() {
	gou.RegisterProcessHandler("yao.app.setting", processSetting)
	gou.RegisterProcessHandler("yao.app.xgen", processXgen)
	gou.AliasProcess("flows.xiang.menu", "yao.app.menu")
}

func processSetting(process *gou.Process) interface{} {
	if Setting == nil {
		exception.New("the app does not init", 500).Throw()
		return nil
	}
	return *Setting
}

func processXgen(process *gou.Process) interface{} {

	if Setting == nil {
		exception.New("the app does not init", 500).Throw()
	}

	mode := os.Getenv("YAO_ENV")
	if mode == "" {
		mode = "production"
	}

	xgenLogin := map[string]map[string]interface{}{
		"entry": {"admin": "/x/Welcome"},
	}

	if admin, has := login.Logins["admin"]; has {
		xgenLogin["entry"]["admin"] = admin.Layout.Entry
		xgenLogin["admin"] = map[string]interface{}{
			"captcha": "/api/__yao/login/admin/captcha?type=digit",
			"login":   "/api/__yao/login/admin",
		}
	}

	if user, has := login.Logins["user"]; has {
		xgenLogin["entry"]["user"] = user.Layout.Entry
		xgenLogin["user"] = map[string]interface{}{
			"captcha": "/api/__yao/login/user/captcha?type=digit",
			"login":   "/api/__yao/login/user",
		}
	}

	xgenSetting := map[string]interface{}{
		"name":        Setting.Name,
		"description": Setting.Description,
		"mode":        mode,
		"apiPrefix":   "__yao",
		"token":       "localStorage",
		"optional":    Setting.Optional,
		"login":       xgenLogin,
	}

	return xgenSetting
}

// Lang for applying a language pack
func (dsl *DSL) Lang(trans func(widget string, inst string, value *string) bool) {
	widget := "app"
	trans(widget, "app", &dsl.Name)
	trans(widget, "app", &dsl.Short)
	trans(widget, "app", &dsl.Description)
}
