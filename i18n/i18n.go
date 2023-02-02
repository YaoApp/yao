package i18n

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"gopkg.in/yaml.v3"
)

type langCache = struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

var cache langCache = langCache{
	data: map[string]interface{}{},
	mu:   sync.RWMutex{},
}

var timezone *time.Location

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

	// Load langs
	err = lang.Load("langs")
	if err != nil {
		return err
	}
	if _, has := lang.Dicts[cfg.Lang]; !has {
		log.Error("The language pack %s does not found", cfg.Lang)
		return nil
	}
	lang.Pick(cfg.Lang).AsDefault()

	// Load Timezone
	if cfg.TimeZone != "" {
		loc, err := time.LoadLocation(cfg.TimeZone)
		if err != nil {
			log.Error("Load timezone error %s", cfg.TimeZone)
		}
		timezone = loc
	}

	// Clear lang cache
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.data = map[string]interface{}{}

	return nil
}

// Trans translate dsl
func Trans(langName string, widgets []string, data interface{}) (interface{}, error) {

	// Get From cache
	hash := sha256.Sum256([]byte(fmt.Sprintf("%v", data)))
	key := fmt.Sprintf("%s::%s::%s", langName, strings.Join(widgets, "::"), hash)

	if res, has := cache.data[key]; has {
		return res, nil
	}

	var dict *lang.Dict = lang.Default
	if langName != "" {
		dict = lang.Pick(langName)
	}

	res, err := dict.ReplaceClone(widgets, data)
	if err != nil {
		return nil, err
	}

	cacheSet(dict, widgets, res)
	return res, nil
}

// cacheSet cache set
func cacheSet(dict *lang.Dict, widgets []string, value interface{}) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	hash := sha256.Sum256([]byte(fmt.Sprintf("%v", value)))
	key := fmt.Sprintf("%s::%s::%s", dict.Name, strings.Join(widgets, "::"), hash)
	cache.data[key] = value
}

func loadFromBin() error {

	dirs := map[string][]struct {
		File     string
		Widget   string
		IsGlobal bool
	}{
		"zh-cn": {
			{File: "yao/langs/zh-cn/global.yml", IsGlobal: true},
			{File: "yao/langs/zh-cn/logins/admin.login.yml", Widget: "login.admin"},
			{File: "yao/langs/zh-cn/logins/user.login.yml", Widget: "login.user"},
		},
		"zh-hk": {
			{File: "yao/langs/zh-hk/global.yml", IsGlobal: true},
			{File: "yao/langs/zh-hk/logins/admin.login.yml", Widget: "login.admin"},
			{File: "yao/langs/zh-hk/logins/user.login.yml", Widget: "login.user"},
		},
	}

	for langName, files := range dirs {

		dict := &lang.Dict{
			Name:    langName,
			Global:  lang.Words{},
			Widgets: map[string]lang.Words{},
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
				dict.Widgets[f.Widget] = lang.Words{}
			}
			dict.Widgets[f.Widget] = words
		}

		if _, has := lang.Dicts[langName]; has {
			lang.Dicts[langName].Merge(dict)
			continue
		}

		lang.Dicts[langName] = dict
	}
	return nil
}
