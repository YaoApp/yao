package login

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

//
// API:
//   GET  /api/__yao/login/:id/captcha  -> Default process: yao.utils.Captcha :query
//  POST  /api/__yao/login/:id  		-> Default process: yao.login.Admin :payload
//

// Logins the loaded login widgets
var Logins map[string]*DSL = map[string]*DSL{}

// LoadAndExport load login
func LoadAndExport(cfg config.Config) error {
	err := Load(cfg)
	if err != nil {
		return err
	}
	return Export()
}

// Load load login
func Load(cfg config.Config) error {
	exts := []string{"*.login.yao", "*.login.json", "*.login.jsonc"}
	return application.App.Walk("logins", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		return LoadFile(root, file)
	}, exts...)
}

// LoadFile by dsl file
func LoadFile(root string, file string) error {

	id := share.ID(root, file)
	data, err := application.App.Read(file)
	if err != nil {
		return err
	}

	dsl := &DSL{ID: id}
	err = application.Parse(file, data, dsl)
	if err != nil {
		return fmt.Errorf("[%s] %s", id, err.Error())
	}

	Logins[id] = dsl
	return nil
}

// Export export login api
func Export() error {
	exportProcess()
	return exportAPI()
}

// exportAPI export login api
func exportAPI() error {

	http := api.HTTP{
		Name:        "Widget Login API",
		Description: "Widget Login API",
		Version:     share.VERSION,
		Guard:       "bearer-jwt",
		Group:       "__yao/login",
		Paths:       []api.Path{},
	}

	for _, dsl := range Logins {

		// login action
		process := "yao.login.Admin"
		args := []interface{}{":payload"}
		if dsl.Action.Process != "" {
			process = dsl.Action.Process
			args = dsl.Action.Args
		}
		path := api.Path{
			Label:       fmt.Sprintf("%s login", dsl.ID),
			Description: fmt.Sprintf("%s login", dsl.ID),
			Guard:       "-",
			Path:        fmt.Sprintf("/%s", dsl.ID),
			Method:      "POST",
			Process:     process,
			In:          args,
			Out:         api.Out{Status: 200, Type: "application/json"},
		}
		http.Paths = append(http.Paths, path)

		// captcha
		process = "utils.captcha.Make"
		args = []interface{}{":query"}
		if dsl.Layout.Captcha != "" {
			process = dsl.Layout.Captcha
		}

		path = api.Path{
			Label:       fmt.Sprintf("%s captcha", dsl.ID),
			Description: fmt.Sprintf("%s captcha", dsl.ID),
			Guard:       "-",
			Path:        fmt.Sprintf("/%s/captcha", dsl.ID),
			Method:      "GET",
			Process:     process,
			In:          args,
			Out:         api.Out{Status: 200, Type: "application/json"},
		}
		http.Paths = append(http.Paths, path)

	}

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = api.LoadSource("<widget.login>.yao", source, "widgets.login")
	return err
}
