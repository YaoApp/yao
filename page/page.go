package page

import (
	"fmt"
	"path"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Pages 已载入页面
var Pages = map[string]*Page{}

// Load 加载页面
func Load(cfg config.Config) error {

	if share.BUILDIN {
		return LoadBuildIn("pages", "")
	}

	var errs error = nil
	if err := LoadFrom(path.Join(cfg.Root, "/kanban"), ""); err != nil {
		// log.Trace("load kanban error: %s ", err.Error())
		errs = err
	}
	if err := LoadFrom(path.Join(cfg.Root, "/screen"), ""); err != nil {
		// log.Trace("load screen error: %s", err.Error())
		errs = err
	}
	if err := LoadFrom(path.Join(cfg.Root, "/pages"), ""); err != nil {
		// log.Trace("load screen error: %s", err.Error())
		errs = err
	}

	return errs
}

// LoadBuildIn 从制品中读取
func LoadBuildIn(dir string, prefix string) error {
	return nil
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := LoadPage(content, name)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
		}
	})
	if err != nil {
		return err
	}

	// Load Script
	err = share.Walk(dir, ".js", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		page := Select(name)
		if page != nil {
			script := share.ScriptName(filename)
			content := share.ReadFile(filename)
			page.LoadScript(string(content), script)
		}
	})

	return err

}

// LoadPage 载入页面
func LoadPage(source []byte, name string) (*Page, error) {
	page := &Page{
		Flow: gou.Flow{
			Name: name,
		},
	}
	err := jsoniter.Unmarshal(source, page)
	if err != nil {
		log.With(log.F{"name": name, "source": source}).Error(err.Error())
		return nil, err
	}
	page.Prepare()
	page.SetupAPIs()
	Pages[name] = page

	// Apply a language pack
	if lang.Default != nil {
		lang.Default.Apply(Pages[name])
	}

	return page, nil
}

// Select 读取已加载页面
func Select(name string) *Page {
	page, has := Pages[name]
	if !has {
		exception.New(
			fmt.Sprintf("Page:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return page
}

// GetData 运行 flow 返回数值
func (page Page) GetData(params map[string]interface{}) interface{} {
	return page.Flow.Exec(params)
}
