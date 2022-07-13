package chart

import (
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/maps"
)

// 注册处理器
func init() {
	gou.RegisterProcessHandler("xiang.chart.data", ProcessData)
	gou.RegisterProcessHandler("xiang.chart.setting", ProcessSetting)
}

// ProcessData xiang.chart.data
// 查询数据分析图表中定义的数据
func ProcessData(process *gou.Process) interface{} {

	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	params := process.ArgsMap(1)
	chart := Select(name)
	api := chart.APIs["data"]

	if len(api.Default) > 0 {
		if defaults, ok := api.Default[0].(map[string]interface{}); ok {
			for key, value := range defaults {
				if !params.Has(key) {
					params.Set(key, value)
				}
			}
		}
	}

	// with Session
	chart.Flow.WithGlobal(process.Global)
	chart.Flow.WithSID(process.Sid)

	return chart.GetData(params)
}

// ProcessSetting xiang.chart.setting
// 查询数据分析图表中定义的数据
func ProcessSetting(process *gou.Process) interface{} {

	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	field := process.ArgsString(1)
	chart := Select(name)

	fields := strings.Split(field, ",")
	setting := maps.Map{
		"name":        chart.Name,
		"label":       chart.Label,
		"version":     chart.Version,
		"description": chart.Description,
		"filters":     chart.Filters,
		"page":        chart.Page,
	}

	if len(fields) == 1 && setting.Has(fields[0]) {
		field := strings.TrimSpace(fields[0])
		return setting.Get(field)
	}

	if len(fields) > 1 {
		res := maps.Map{}
		for _, field := range fields {
			field = strings.TrimSpace(field)
			if setting.Has(field) {
				res.Set(field, setting.Get(field))
			}
		}
		return res
	}
	return setting

}
