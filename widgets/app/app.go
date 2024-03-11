package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/login"
)

//
// API:
//   GET   /api/__yao/app/setting 			-> Default process: yao.app.Xgen
//   POST  /api/__yao/app/setting 			-> Default process: yao.app.Xgen  {"sid":"xxx", "lang":"zh-hk", "time": "2022-10-10 22:00:10"}
//   GET   /api/__yao/app/menu  			-> Default process: yao.app.Menu
//   POST  /api/__yao/app/check  			-> Default process: yao.app.Check
//   POST  /api/__yao/app/setup  			-> Default process: yao.app.Setup   {"sid":"xxxx", ...}
//   POST  /api/__yao/app/service/:name  	-> Default process: yao.app.Service {"method":"Bar", "args":["hello", "world"]}
//
// Process:
// 	 yao.app.Setting Return the App DSL
// 	 yao.app.Xgen Return the Xgen setting ( merge app & login )
//   yao.app.Menu Return the menu list
//

// Setting the application setting
var Setting *DSL
var regExcp = regexp.MustCompile("^Exception\\|([0-9]+):(.+)$")

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

	file, err := getAppFile()
	if err != nil {
		return err
	}

	data, err := application.App.Read(file)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	if data == nil {
		return fmt.Errorf("app.yao not found")
	}

	dsl := &DSL{Optional: OptionalDSL{}, Lang: cfg.Lang}
	err = application.Parse(file, data, dsl)
	if err != nil {
		return err
	}

	// Replace Admin Root
	err = dsl.replaceAdminRoot()
	if err != nil {
		return err
	}

	// Load icons
	dsl.icons(cfg)

	Setting = dsl
	return nil
}

func getAppFile() (string, error) {

	file := filepath.Join(string(os.PathSeparator), "app.yao")
	if has, _ := application.App.Exists(file); has {
		return file, nil
	}

	file = filepath.Join(string(os.PathSeparator), "app.jsonc")
	if has, _ := application.App.Exists(file); has {
		return file, nil
	}

	file = filepath.Join(string(os.PathSeparator), "app.json")
	if has, _ := application.App.Exists(file); has {
		return file, nil
	}

	return "", fmt.Errorf("app.yao not found")
}

// exportAPI export login api
func exportAPI() error {

	if Setting == nil {
		return fmt.Errorf("the app does not init")
	}

	http := api.HTTP{
		Name:        "Widget App API",
		Description: "Widget App API",
		Version:     share.VERSION,
		Guard:       "bearer-jwt",
		Group:       "__yao/app",
		Paths:       []api.Path{},
	}

	process := "yao.app.Xgen"
	if Setting.Setting != "" {
		process = Setting.Setting
	}

	path := api.Path{
		Label:       "App Setting",
		Description: "App Setting",
		Guard:       "-",
		Path:        "/setting",
		Method:      "GET",
		Process:     process,
		In:          []interface{}{},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	// POST
	path = api.Path{
		Label:       "App Setting",
		Description: "App Setting",
		Guard:       "-",
		Path:        "/setting",
		Method:      "POST",
		Process:     process,
		In:          []interface{}{":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	process = "yao.app.Menu"
	args := []interface{}{}
	if Setting.Menu.Process != "" {
		if Setting.Menu.Args != nil {
			args = Setting.Menu.Args
		}
	}

	path = api.Path{
		Label:       "App Menu",
		Description: "App Menu",
		Path:        "/menu",
		Method:      "GET",
		Process:     process,
		In:          args,
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	process = "yao.app.Icons"
	path = api.Path{
		Label:       "App Icons",
		Description: "App Icons",
		Path:        "/icons/:name",
		Guard:       "-",
		Method:      "GET",
		Process:     process,
		In:          []interface{}{"$param.name"},
		Out:         api.Out{Status: 200},
	}
	http.Paths = append(http.Paths, path)

	path = api.Path{
		Label:       "Setup",
		Description: "Setup",
		Path:        "/setup",
		Guard:       "-",
		Method:      "POST",
		Process:     "yao.app.Setup",
		In:          []interface{}{":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	path = api.Path{
		Label:       "Check",
		Description: "Check",
		Path:        "/check",
		Guard:       "-",
		Method:      "POST",
		Process:     "yao.app.Check",
		In:          []interface{}{":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	path = api.Path{
		Label:       "Serivce",
		Description: "Serivce",
		Path:        "/service/:name",
		Guard:       "bearer-jwt",
		Method:      "POST",
		Process:     "yao.app.Service",
		In:          []interface{}{"$param.name", ":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = api.LoadSource("<widget.app>.yao", source, "widgets.app")
	return err
}

// Export export login api
func Export() error {
	exportProcess()
	return exportAPI()
}

func exportProcess() {
	process.Register("yao.app.setting", processSetting)
	process.Register("yao.app.xgen", processXgen)
	process.Register("yao.app.menu", processMenu)
	process.Register("yao.app.icons", processIcons)
	process.Register("yao.app.setup", processSetup)
	process.Register("yao.app.check", processCheck)
	process.Register("yao.app.service", processService)
}

func processService(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	service := fmt.Sprintf("__yao_service.%s", process.ArgsString(0))
	payload := process.ArgsMap(1)
	if payload == nil || len(payload) == 0 {
		exception.New("content is required", 400).Throw()
	}

	method, ok := payload["method"].(string)
	if !ok || service == "" {
		exception.New("method is required", 400).Throw()
	}

	args := []interface{}{}
	if v, ok := payload["args"].([]interface{}); ok {
		args = v
	}

	//
	// Forward: Neo confirm Command
	// @file   neo/command/request.go
	// @method func (req *Request) confirm(args []interface{}, cb func(msg *message.JSON) int)
	//
	if service == "__yao_service.__neo" && method == "ExecCommand" {
		if len(args) < 4 {
			exception.New("args is required (%v)", 400, args).Throw()
		}

		id := args[0].(string)
		ctx := args[3].(map[string]interface{})
		processName := args[1].(string)
		processArgs := append(args[2].([]interface{}), ctx)
		result := forwardNeoExecCommand(process, processName, processArgs...)
		return map[string]interface{}{"id": id, "result": result, "context": ctx}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	script, err := v8.Select(service)
	if err != nil {
		exception.New("services.%s not loaded", 404, process.ArgsString(0)).Throw()
		return nil
	}

	v8ctx, err := script.NewContext(process.Sid, process.Global)
	if err != nil {
		message := fmt.Sprintf("services.%s failed to create context. %s", process.ArgsString(0), err.Error())
		log.Error("[V8] process error. %s", message)
		exception.New(message, 500).Throw()
		return nil
	}
	defer v8ctx.Close()

	res, err := v8ctx.CallWith(ctx, method, args...)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

func forwardNeoExecCommand(p *process.Process, name string, args ...interface{}) interface{} {
	new, err := process.Of(name, args...)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}

	res, err := new.WithGlobal(p.Global).WithSID(p.Sid).Exec()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

func processCheck(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	payload := process.ArgsMap(0)
	time.Sleep(3 * time.Second)

	if _, has := payload["error"]; has {
		exception.New("Something error", 500).Throw()
	}
	return nil
}

func processSetup(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	payload := process.ArgsMap(0)
	time.Sleep(3 * time.Second)

	if _, has := payload["error"]; has {
		exception.New("Something error", 500).Throw()
	}

	lang := session.Lang(process)
	if sid, has := payload["sid"].(string); has {
		lang, err := session.Global().ID(sid).Get("__yao_lang")
		if err != nil {
			lang = strings.ToLower(lang.(string))
		}
	}

	root := "yao"
	if Setting.AdminRoot != "" {
		root = Setting.AdminRoot
	}

	setting, err := i18n.Trans(lang, []string{"app.app"}, Setting)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return map[string]interface{}{
		"home":    fmt.Sprintf("http://127.0.0.1:%d", config.Conf.Port),
		"admin":   fmt.Sprintf("http://127.0.0.1:%d/%s/", config.Conf.Port, root),
		"setting": setting,
	}
}

func processIcons(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	file := filepath.Join("icons", name)
	content, err := application.App.Read(file)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return string(content)
}

func processMenu(p *process.Process) interface{} {

	if Setting.Menu.Process != "" {

		return process.
			New(Setting.Menu.Process, p.Args...).
			WithGlobal(p.Global).
			WithSID(p.Sid).
			Run()
	}

	args := map[string]interface{}{
		"select": []string{"id", "name", "icon", "parent", "path", "blocks", "visible_menu"},
		"withs": map[string]interface{}{
			"children": map[string]interface{}{
				"query": map[string]interface{}{
					"select": []string{"id", "name", "icon", "parent", "path", "blocks", "visible_menu"},
				},
			},
		},
		"wheres": []map[string]interface{}{
			{"column": "status", "value": "enabled"},
			{"column": "parent", "op": "null"},
		},
		"limit":  200,
		"orders": []map[string]interface{}{{"column": "rank", "option": "asc"}},
	}
	return process.
		New("models.xiang.menu.get", args).
		WithGlobal(p.Global).
		WithSID(p.Sid).
		Run()
}

func processSetting(process *process.Process) interface{} {
	if Setting == nil {
		exception.New("the app does not init", 500).Throw()
		return nil
	}

	sid := process.Sid
	if sid == "" {
		sid = session.ID()
	}

	// Set User ENV
	if process.NumOfArgs() > 0 {
		payload := process.ArgsMap(0, map[string]interface{}{
			"now":  time.Now().Unix(),
			"lang": "en-us",
			"sid":  "",
		})

		if v, ok := payload["sid"].(string); ok && v != "" {
			sid = v
		}

		lang := strings.ToLower(fmt.Sprintf("%v", payload["lang"]))
		session.Global().ID(sid).Set("__yao_lang", lang)
	}

	setting, err := i18n.Trans(session.Lang(process, config.Conf.Lang), []string{"app.app"}, Setting)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	setting.(*DSL).Sid = sid
	return *setting.(*DSL)
}

func processXgen(process *process.Process) interface{} {

	if Setting == nil {
		exception.New("the app does not init", 500).Throw()
	}

	sid := process.Sid
	if sid == "" {
		sid = session.ID()
	}

	// Set User ENV
	if process.NumOfArgs() > 0 {
		payload := process.ArgsMap(0, map[string]interface{}{
			"now":  time.Now().Unix(),
			"lang": "en-us",
			"sid":  "",
		})

		if v, ok := payload["sid"].(string); ok && v != "" {
			sid = v
		}

		lang := strings.ToLower(fmt.Sprintf("%v", payload["lang"]))
		session.Global().ID(sid).Set("__yao_lang", lang)
	}

	mode := os.Getenv("YAO_ENV")
	if mode == "" {
		mode = "production"
	}

	xgenLogin := map[string]map[string]interface{}{
		"entry": {"admin": "/x/Welcome"},
	}

	if admin, has := login.Logins["admin"]; has {
		layout := map[string]interface{}{}
		if admin.Layout.Site != "" {
			layout["site"] = admin.Layout.Site
		}

		if admin.Layout.Slogan != "" {
			layout["slogan"] = admin.Layout.Slogan
		}

		if admin.Layout.Cover != "" {
			layout["cover"] = admin.Layout.Cover
		}

		// Translate
		newLayout, err := i18n.Trans(session.Lang(process, config.Conf.Lang), []string{"login.admin"}, layout)
		if err != nil {
			log.Error("[Login] Xgen i18n.Trans login.admin %s", err.Error())
		}

		if new, ok := newLayout.(map[string]interface{}); ok {
			layout = new
		}

		xgenLogin["entry"]["admin"] = admin.Layout.Entry
		xgenLogin["admin"] = map[string]interface{}{
			"captcha": "/api/__yao/login/admin/captcha?type=digit",
			"login":   "/api/__yao/login/admin",
			"layout":  layout,
		}

		if admin.ThirdPartyLogin != nil && len(admin.ThirdPartyLogin) > 0 {
			xgenLogin["admin"]["thirdPartyLogin"] = admin.ThirdPartyLogin
		}

	}

	if user, has := login.Logins["user"]; has {
		layout := map[string]interface{}{}
		if user.Layout.Site != "" {
			layout["site"] = user.Layout.Site
		}

		if user.Layout.Slogan != "" {
			layout["slogan"] = user.Layout.Slogan
		}

		if user.Layout.Cover != "" {
			layout["cover"] = user.Layout.Cover
		}

		// Translate
		newLayout, err := i18n.Trans(session.Lang(process, config.Conf.Lang), []string{"login.user"}, layout)
		if err != nil {
			log.Error("[Login] Xgen %s", err.Error())
		}

		if new, ok := newLayout.(map[string]interface{}); ok {
			layout = new
		}
		xgenLogin["entry"]["user"] = user.Layout.Entry
		xgenLogin["user"] = map[string]interface{}{
			"captcha": "/api/__yao/login/user/captcha?type=digit",
			"login":   "/api/__yao/login/user",
			"layout":  layout,
		}

		if user.ThirdPartyLogin != nil && len(user.ThirdPartyLogin) > 0 {
			xgenLogin["user"]["thirdPartyLogin"] = user.ThirdPartyLogin
		}
	}

	xgenSetting := map[string]interface{}{
		"name":        Setting.Name,
		"description": Setting.Description,
		"version":     Setting.Version,
		"theme":       Setting.Theme,
		"lang":        Setting.Lang,
		"mode":        mode,
		"apiPrefix":   "__yao",
		"token":       Setting.Token,
		"optional":    Setting.Optional,
		"login":       xgenLogin,
	}

	if Setting.Logo != "" {
		xgenSetting["logo"] = Setting.Logo
	}

	if Setting.Favicon != "" {
		xgenSetting["favicon"] = Setting.Favicon
	}

	setting, err := i18n.Trans(session.Lang(process, config.Conf.Lang), []string{"app.app"}, xgenSetting)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	setting.(map[string]interface{})["sid"] = sid
	return setting.(map[string]interface{})
}

// replaceAdminRoot
func (dsl *DSL) replaceAdminRoot() error {

	if dsl.AdminRoot == "" {
		dsl.AdminRoot = "yao"
	}

	root := strings.TrimPrefix(dsl.AdminRoot, "/")
	root = strings.TrimSuffix(root, "/")
	// err := data.ReplaceXGen("/__yao_admin_root/", fmt.Sprintf("/%s/", root))
	// if err != nil {
	// 	return err
	// }

	return data.ReplaceXGen("__yao_admin_root", root)
}

// icons
func (dsl *DSL) icons(cfg config.Config) {

	dsl.Favicon = fmt.Sprintf("/api/__yao/app/icons/app.ico")
	dsl.Logo = fmt.Sprintf("/api/__yao/app/icons/app.png")

	// favicon := filepath.Join(cfg.Root, "icons", "app.ico")
	// if _, err := os.Stat(favicon); err == nil {
	// 	dsl.Favicon = fmt.Sprintf("/api/__yao/app/icons/app.ico")
	// }

	// logo := filepath.Join(cfg.Root, "icons", "app.png")
	// if _, err := os.Stat(logo); err == nil {
	// 	dsl.Logo = fmt.Sprintf("/api/__yao/app/icons/app.png")
	// }
}

// Permissions get the permission blacklist
// {"<widget>.<ID>":[<id...>]}
func Permissions(process *process.Process, widget string, id string) map[string]bool {
	permissions := map[string]bool{}
	sessionData, _ := session.Global().ID(process.Sid).Get("__permissions")
	data, ok := sessionData.(map[string]interface{})
	if !ok && sessionData != nil {
		log.Error("[Permissions] session data should be a map, but got %#v", sessionData)
		return permissions
	}

	switch values := data[fmt.Sprintf("%s.%s", widget, id)].(type) {
	case []interface{}:
		for _, value := range values {
			permissions[fmt.Sprintf("%v", value)] = true
		}
		break

	case []string:
		for _, value := range values {
			permissions[value] = true
		}
		break

	case map[string]interface{}:
		for key := range values {
			permissions[key] = true
		}
		break

	case map[string]bool:
		permissions = values
		break
	}

	return permissions
}
