package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
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

	file := filepath.Join(cfg.Root, "app.json")
	file, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	dsl := &DSL{Optional: OptionalDSL{}, Lang: cfg.Lang}
	err = jsoniter.Unmarshal(data, dsl)
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
	if Setting.Setting != "" {
		process = Setting.Setting
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

	// POST
	path = gou.Path{
		Label:       "App Setting",
		Description: "App Setting",
		Guard:       "-",
		Path:        "/setting",
		Method:      "POST",
		Process:     process,
		In:          []string{":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	process = "yao.app.Menu"
	args := []string{}
	if Setting.Menu.Process != "" {
		if Setting.Menu.Args != nil {
			args = Setting.Menu.Args
		}
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

	process = "yao.app.Icons"
	path = gou.Path{
		Label:       "App Icons",
		Description: "App Icons",
		Path:        "/icons/:name",
		Guard:       "-",
		Method:      "GET",
		Process:     process,
		In:          []string{"$param.name"},
		Out:         gou.Out{Status: 200},
	}
	http.Paths = append(http.Paths, path)

	path = gou.Path{
		Label:       "Setup",
		Description: "Setup",
		Path:        "/setup",
		Guard:       "-",
		Method:      "POST",
		Process:     "yao.app.Setup",
		In:          []string{":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	path = gou.Path{
		Label:       "Check",
		Description: "Check",
		Path:        "/check",
		Guard:       "-",
		Method:      "POST",
		Process:     "yao.app.Check",
		In:          []string{":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	path = gou.Path{
		Label:       "Serivce",
		Description: "Serivce",
		Path:        "/service/:name",
		Guard:       "bearer-jwt",
		Method:      "POST",
		Process:     "yao.app.Serivce",
		In:          []string{"$param.name", ":payload"},
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
	gou.RegisterProcessHandler("yao.app.menu", processMenu)
	gou.RegisterProcessHandler("yao.app.icons", processIcons)
	gou.RegisterProcessHandler("yao.app.setup", processSetup)
	gou.RegisterProcessHandler("yao.app.check", processCheck)
	gou.RegisterProcessHandler("yao.app.service", processService)
}

func processService(process *gou.Process) interface{} {
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

	req := gou.Yao.New(service, method)
	if process.Sid != "" {
		req.WithSid(process.Sid)
	}

	res, err := req.Call(args...)
	if err != nil {
		// parse Exception
		code := 500
		message := err.Error()
		match := regExcp.FindStringSubmatch(message)
		if len(match) > 0 {
			code, err = strconv.Atoi(match[1])
			if err == nil {
				message = strings.TrimSpace(match[2])
			}
		}
		exception.New(message, code).Throw()
	}

	return res
}

func processCheck(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	payload := process.ArgsMap(0)
	time.Sleep(3 * time.Second)

	if _, has := payload["error"]; has {
		exception.New("Something error", 500).Throw()
	}
	return nil
}

func processSetup(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	payload := process.ArgsMap(0)
	time.Sleep(3 * time.Second)

	if _, has := payload["error"]; has {
		exception.New("Something error", 500).Throw()
	}

	lang := process.Lang()
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

func processIcons(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	file, err := filepath.Abs(filepath.Join(config.Conf.Root, "icons", name))
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	content, err := ioutil.ReadFile(file)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return string(content)
}

func processMenu(process *gou.Process) interface{} {

	if Setting.Menu.Process != "" {
		return gou.
			NewProcess(Setting.Menu.Process, process.Args...).
			WithGlobal(process.Global).
			WithSID(process.Sid).
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
	return gou.
		NewProcess("models.xiang.menu.get", args).
		WithGlobal(process.Global).
		WithSID(process.Sid).
		Run()
}

func processSetting(process *gou.Process) interface{} {
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

	setting, err := i18n.Trans(process.Lang(config.Conf.Lang), []string{"app.app"}, Setting)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	setting.(*DSL).Sid = sid
	return *setting.(*DSL)
}

func processXgen(process *gou.Process) interface{} {

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
		newLayout, err := i18n.Trans(process.Lang(config.Conf.Lang), []string{"login.admin"}, layout)
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
		newLayout, err := i18n.Trans(process.Lang(config.Conf.Lang), []string{"login.user"}, layout)
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
	}

	xgenSetting := map[string]interface{}{
		"name":        Setting.Name,
		"description": Setting.Description,
		"theme":       Setting.Theme,
		"lang":        Setting.Lang,
		"mode":        mode,
		"apiPrefix":   "__yao",
		"token":       "localStorage",
		"optional":    Setting.Optional,
		"login":       xgenLogin,
	}

	if Setting.Logo != "" {
		xgenSetting["logo"] = Setting.Logo
	}

	if Setting.Favicon != "" {
		xgenSetting["favicon"] = Setting.Favicon
	}

	setting, err := i18n.Trans(process.Lang(config.Conf.Lang), []string{"app.app"}, xgenSetting)
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
	err := data.ReplaceXGen("/__yao_admin_root/", fmt.Sprintf("/%s/", root))
	if err != nil {
		return err
	}

	return data.ReplaceXGen("\"__yao_admin_root\"", fmt.Sprintf("\"%s\"", root))
}

// icons
func (dsl *DSL) icons(cfg config.Config) {

	favicon := filepath.Join(cfg.Root, "icons", "app.ico")
	if _, err := os.Stat(favicon); err == nil {
		dsl.Favicon = fmt.Sprintf("/api/__yao/app/icons/app.ico")
	}

	logo := filepath.Join(cfg.Root, "icons", "app.png")
	if _, err := os.Stat(logo); err == nil {
		dsl.Logo = fmt.Sprintf("/api/__yao/app/icons/app.png")
	}
}
