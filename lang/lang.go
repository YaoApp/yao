package lang

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"gopkg.in/yaml.v3"
)

func init() {
	lang.RegisterWidget("logins", "login")
	lang.RegisterWidget("tables", "table")
	lang.RegisterWidget("forms", "form")
	lang.RegisterWidget("charts", "chart")
	lang.RegisterWidget("kanban", "page")
	lang.RegisterWidget("screen", "page")
	lang.RegisterWidget("pages", "page")
}

// Load language packs
func Load(cfg config.Config) error {
	err := loadFromBin()
	if err != nil {
		return err
	}

	root := filepath.Join(cfg.Root, "langs")
	err = lang.Load(root)
	if err != nil {
		return err
	}

	// Set default
	lang.Pick("default").AsDefault()
	name := os.Getenv("YAO_LANG")
	if name != "" {
		if _, has := lang.Dicts[name]; !has {
			log.Error("The language pack %s does not found", name)
			return nil
		}
		lang.Pick(name).AsDefault()
	}

	return nil
}

func loadFromBin() error {

	dirs := map[string][]struct {
		File     string
		Widget   string
		Instance string
		IsGlobal bool
	}{
		"zh-cn": {
			{File: "yao/langs/zh-cn/global.yml", IsGlobal: true},
			{File: "yao/langs/zh-cn/logins/admin.login.yml", Widget: "login", Instance: "admin"},
			{File: "yao/langs/zh-cn/logins/user.login.yml", Widget: "login", Instance: "user"},
		},
		"zh-hk": {
			{File: "yao/langs/zh-hk/global.yml", IsGlobal: true},
			{File: "yao/langs/zh-hk/logins/admin.login.yml", Widget: "login", Instance: "admin"},
			{File: "yao/langs/zh-hk/logins/user.login.yml", Widget: "login", Instance: "user"},
		},
	}

	for langName, files := range dirs {

		dict := &lang.Dict{
			Name:    langName,
			Global:  lang.Words{},
			Widgets: map[string]lang.Widget{},
		}

		for _, f := range files {
			data, err := data.Read(f.File)
			if err != nil {
				return fmt.Errorf("%s: %s", f.File, err.Error())
			}

			words := lang.Words{}
			err = yaml.Unmarshal(data, &words)
			if err != nil {
				return fmt.Errorf("%s: %s", f.File, err.Error())
			}

			if f.IsGlobal {
				dict.Global = words
				continue
			}

			if _, has := dict.Widgets[f.Widget]; !has {
				dict.Widgets[f.Widget] = map[string]lang.Words{}
			}
			dict.Widgets[f.Widget][f.Instance] = words
		}

		if _, has := lang.Dicts[langName]; has {
			lang.Dicts[langName].Merge(dict)
			continue
		}

		lang.Dicts[langName] = dict
	}
	return nil
}
