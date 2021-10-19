package chart

import (
	"github.com/yaoapp/xiang/share"
)

// SetupAPIs 设定API数据
func (chart *Chart) SetupAPIs() {

	defaults := map[string]share.API{
		"data":    apiDataDefault(),
		"setting": apiSettingDefault(),
	}

	// 开发者填写的规则
	for name := range chart.APIs {
		if _, has := defaults[name]; !has {
			delete(chart.APIs, name)
			continue
		}

		api := defaults[name]
		api.Name = name
		if chart.APIs[name].Process != "" {
			api.Process = chart.APIs[name].Process
		}

		if chart.APIs[name].Guard != "" {
			api.Guard = chart.APIs[name].Guard
		}

		if chart.APIs[name].Default != nil {
			api.Default = chart.APIs[name].Default
		}

		defaults[name] = api
	}

	chart.APIs = defaults
}

// apiSearchDefault data 接口默认值
func apiDataDefault() share.API {
	param := map[string]interface{}{}
	return share.API{
		Name:    "data",
		Guard:   "bearer-jwt",
		Process: "xiang.chart.data",
		Default: []interface{}{param},
	}
}

// apiSettingDefault setting 接口默认值
func apiSettingDefault() share.API {
	return share.API{
		Name:    "setting",
		Guard:   "bearer-jwt",
		Process: "xiang.chart.setting",
	}
}
