package page

import (
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/maps"
)

// 注册处理器
func init() {
	gou.RegisterProcessHandler("xiang.page.data", ProcessData)
	gou.RegisterProcessHandler("xiang.page.setting", ProcessSetting)
}

// ProcessData xiang.page.data
// 查询数据分析页面中定义的数据
func ProcessData(process *gou.Process) interface{} {

	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	params := process.ArgsMap(1)
	page := Select(name)
	api := page.APIs["data"]

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
	page.Flow.WithGlobal(process.Global)
	page.Flow.WithSID(process.Sid)

	return page.GetData(params)
}

// ProcessSetting xiang.page.setting
// 查询数据分析页面中定义的数据
func ProcessSetting(process *gou.Process) interface{} {

	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	field := process.ArgsString(1)
	page := Select(name)

	fields := strings.Split(field, ",")
	setting := maps.Map{
		"name":        page.Name,
		"label":       page.Label,
		"version":     page.Version,
		"description": page.Description,
		"filters":     page.Filters,
		"page":        page.Page,
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
