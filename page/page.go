package page

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/xlog"
)

// Pages 已载入页面
var Pages = map[string]*Page{}

// Load 加载页面
func Load(cfg config.Config) {
	LoadFrom(cfg.RootPage, "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) {

	if share.DirNotExists(dir) {
		return
	}

	share.Walk(dir, ".json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := LoadPage(content, name)
		if err != nil {
			exception.New("%s 页面格式错误", 400, name).Ctx(filename).Throw()
		}
	})

	// Load Script
	share.Walk(dir, ".js", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		page := Select(name)
		if page != nil {
			script := share.ScriptName(filename)
			content := share.ReadFile(filename)
			page.LoadScript(string(content), script)
		}
	})

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
		xlog.Println(name)
		xlog.Println(err.Error())
		xlog.Println(string(source))
		return nil, err
	}
	page.Prepare()
	page.SetupAPIs()
	Pages[name] = page
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
