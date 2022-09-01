package chart

import (
	"fmt"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Charts 已载入图表
var Charts = map[string]*Chart{}

// Load 加载数据表格
func Load(cfg config.Config) error {
	if share.BUILDIN {
		return LoadBuildIn("charts", "")
	}
	return LoadFrom(filepath.Join(cfg.Root, "charts"), "")
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
		_, err := LoadChart(content, name)
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
		chart := Select(name)
		if chart != nil {
			script := share.ScriptName(filename)
			content := share.ReadFile(filename)
			chart.LoadScript(string(content), script)
		}
	})

	return err
}

// LoadChart 载入数据表格
func LoadChart(source []byte, name string) (*Chart, error) {
	chart := &Chart{
		Flow: &gou.Flow{Name: name},
	}
	err := jsoniter.Unmarshal(source, chart)
	if err != nil {
		log.With(log.F{"name": name, "source": source}).Error(err.Error())
		return nil, err
	}
	chart.Prepare()
	chart.SetupAPIs()
	Charts[name] = chart

	// Apply a language pack
	if lang.Default != nil {
		lang.Default.Apply(Charts[name])
	}

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
