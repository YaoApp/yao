package chart

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
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
		_, err := LoadChart(content, name)
		if err != nil {
			exception.New("%s 图表格式错误", 400, name).Ctx(filename).Throw()
		}
	})
}

// LoadChart 载入数据表格
func LoadChart(source []byte, name string) (*Chart, error) {
	chart := &Chart{
		Flow: gou.Flow{
			Name: name,
		},
	}
	err := jsoniter.Unmarshal(source, chart)
	if err != nil {
		xlog.Println(name)
		xlog.Println(err.Error())
		xlog.Println(string(source))
		return nil, err
	}
	chart.Prepare()
	chart.SetupAPIs()
	Charts[name] = chart
	return chart, nil
}

// Select 读取已加载图表
func Select(name string) *Chart {
	chart, has := Charts[name]
	if !has {
		exception.New(
			fmt.Sprintf("Chart:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return chart
}

// GetData 运行 flow 返回数值
func (chart Chart) GetData(params map[string]interface{}) interface{} {
	return chart.Flow.Exec(params)
}
