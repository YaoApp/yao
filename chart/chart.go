package chart

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/xlog"
)

// Charts 已载入图表
var Charts = map[string]*Chart{}

// Load 加载数据表格
func Load(cfg config.Config) {
	LoadFrom(cfg.RootChart, "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) {

	if share.DirNotExists(dir) {
		return
	}

	share.Walk(dir, ".json", func(root, filename string) {
		name := share.SpecName(root, filename)
		content := share.ReadFile(filename)
		chart, err := LoadChart(content, name)
		if err != nil {
			exception.New("%s 图表格式错误", 400, name).Ctx(filename).Throw()
		}
		Charts[name] = chart
	})
}

// LoadChart 载入数据表格
func LoadChart(source []byte, name string) (*Chart, error) {
	chart := Chart{}
	err := jsoniter.Unmarshal(source, &chart)
	if err != nil {
		xlog.Println(name)
		xlog.Println(err.Error())
		xlog.Println(string(source))
		return nil, err
	}
	chart.Prepare()
	return &chart, nil
}
