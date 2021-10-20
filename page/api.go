package page

import (
	"github.com/yaoapp/xiang/share"
)

// SetupAPIs 设定API数据
func (page *Page) SetupAPIs() {

	defaults := map[string]share.API{
		"data":    apiDataDefault(),
		"setting": apiSettingDefault(),
	}

	// 开发者填写的规则
	for name := range page.APIs {
		if _, has := defaults[name]; !has {
			delete(page.APIs, name)
			continue
		}

		api := defaults[name]
		api.Name = name
		if page.APIs[name].Process != "" {
			api.Process = page.APIs[name].Process
		}

		if page.APIs[name].Guard != "" {
			api.Guard = page.APIs[name].Guard
		}

		if page.APIs[name].Default != nil {
			api.Default = page.APIs[name].Default
		}

		defaults[name] = api
	}

	page.APIs = defaults
}

// apiSearchDefault data 接口默认值
func apiDataDefault() share.API {
	param := map[string]interface{}{}
	return share.API{
		Name:    "data",
		Guard:   "bearer-jwt",
		Process: "xiang.page.data",
		Default: []interface{}{param},
	}
}

// apiSettingDefault setting 接口默认值
func apiSettingDefault() share.API {
	return share.API{
		Name:    "setting",
		Guard:   "bearer-jwt",
		Process: "xiang.page.setting",
	}
}
