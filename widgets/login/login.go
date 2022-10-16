package login

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
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
	var root = filepath.Join(cfg.Root, "logins")
	return LoadFrom(root, "")
}

// LoadFrom load from dir
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	messages := []string{}
	err := share.Walk(dir, ".json", func(root, filename string) {
		id := prefix + share.ID(root, filename)
		data := share.ReadFile(filename)
		dsl := &DSL{ID: id}
		err := jsoniter.Unmarshal(data, dsl)
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		Logins[id] = dsl
	})

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";"))
	}

	return err
}

// Export export login api
func Export() error {
	exportProcess()
	return exportAPI()
}

// exportAPI export login api
func exportAPI() error {

	http := gou.HTTP{
		Name:        "Widget Login API",
		Description: "Widget Login API",
		Version:     share.VERSION,
		Guard:       "bearer-jwt",
		Group:       "__yao/login",
		Paths:       []gou.Path{},
	}

	for _, dsl := range Logins {

		// login action
		process := "yao.login.Admin"
		args := []string{":payload"}
		if dsl.Action.Process != "" {
			process = dsl.Action.Process
			args = dsl.Action.Args
		}
		path := gou.Path{
			Label:       fmt.Sprintf("%s login", dsl.ID),
			Description: fmt.Sprintf("%s login", dsl.ID),
			Guard:       "-",
			Path:        fmt.Sprintf("/%s", dsl.ID),
			Method:      "POST",
			Process:     process,
			In:          args,
			Out:         gou.Out{Status: 200, Type: "application/json"},
		}
		http.Paths = append(http.Paths, path)

		// captcha
		process = "yao.utils.Captcha"
		args = []string{":query"}
		if dsl.Layout.Captcha != "" {
			process = dsl.Layout.Captcha
		}

		path = gou.Path{
			Label:       fmt.Sprintf("%s captcha", dsl.ID),
			Description: fmt.Sprintf("%s captcha", dsl.ID),
			Guard:       "-",
			Path:        fmt.Sprintf("/%s/captcha", dsl.ID),
			Method:      "GET",
			Process:     process,
			In:          args,
			Out:         gou.Out{Status: 200, Type: "application/json"},
		}
		http.Paths = append(http.Paths, path)

	}

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = gou.LoadAPIReturn(string(source), "widgets.login")
	return err
}
