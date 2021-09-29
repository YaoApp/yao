package table

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

// loadAPIs 加载数据管理 API
func (table *Table) loadAPIs() {
	if table.Bind.Model == "" {
		return
	}
	defaults := getDefaultAPIs(table.Bind)
	defaults["setting"] = apiDefaultSetting(table)

	for name := range table.APIs {
		if _, has := defaults[name]; !has {
			delete(table.APIs, name)
			continue
		}

		api := defaults[name]
		api.Name = name
		if table.APIs[name].Process != "" {
			api.Process = table.APIs[name].Process
		}

		if table.APIs[name].Guard != "" {
			api.Guard = table.APIs[name].Guard
		}

		if table.APIs[name].Default != nil {
			api.Default = table.APIs[name].Default
		}
		defaults[name] = api
	}

	table.APIs = defaults
}

// getDefaultAPIs 读取数据模型绑定的APIs
func getDefaultAPIs(bind Bind) map[string]API {
	name := bind.Model
	model := gou.Select(name)
	apis := map[string]API{
		"search":       apiSearchDefault(model, bind.Withs),
		"find":         apiFindDefault(model, bind.Withs),
		"save":         apiDefault(model, "save", "Save"),
		"delete":       apiDefault(model, "delete", "Delete"),
		"insert":       apiDefault(model, "insert", "Insert"),
		"delete-in":    apiDefault(model, "delete-in", "DeleteWhere"),
		"delete-where": apiDefaultWhere(model, bind.Withs, "delete-where", "DeleteWhere"),
		"update-in":    apiDefault(model, "update-in", "UpdateWhere"),
		"update-where": apiDefaultWhere(model, bind.Withs, "update-where", "UpdateWhere"),
	}

	return apis
}

// apiSearchDefault search 接口默认值
func apiSearchDefault(model *gou.Model, withs map[string]gou.With) API {
	query := gou.QueryParam{}
	if model.MetaData.Option.Timestamps {
		query.Orders = []gou.QueryOrder{
			{Column: "created_at", Option: "desc"},
		}
	}

	if withs != nil {
		query.Withs = withs
	}

	return API{
		Name:    "search",
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("models.%s.Paginate", model.Name),
		Default: []interface{}{query, 1, 20},
	}
}

// apiFindDefault find 接口默认值
func apiFindDefault(model *gou.Model, withs map[string]gou.With) API {

	query := gou.QueryParam{}
	if withs != nil {
		query.Withs = withs
	}

	return API{
		Name:    "find",
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("models.%s.Find", model.Name),
		Default: []interface{}{nil, query},
	}
}

// apiFindDefault 接口默认值
func apiDefault(model *gou.Model, name string, process string) API {
	return API{
		Name:    name,
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("models.%s.%s", model.Name, process),
	}
}

// apiFindDefault 接口默认值
func apiDefaultWhere(model *gou.Model, withs map[string]gou.With, name string, process string) API {

	query := gou.QueryParam{}
	if withs != nil {
		query.Withs = withs
	}

	return API{
		Name:    name,
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("models.%s.%s", model.Name, process),
		Default: []interface{}{query},
	}
}

// apiDefaultSetting 数据表格配置默认值
func apiDefaultSetting(table *Table) API {
	return API{
		Name:    "setting",
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("tables.%s.Setting", table.Table),
	}
}

// IsAllow 鉴权处理程序
func (api API) IsAllow(v interface{}) bool {
	c, ok := v.(*gin.Context)
	if !ok {
		return false
	}

	guards := strings.Split(api.Guard, ",")
	for _, guard := range guards {
		guard = strings.TrimSpace(guard)
		handler, has := gou.HTTPGuards[guard]
		if has {
			handler(c)
			fmt.Println(api.Guard, c.IsAborted())
			return c.IsAborted()
		}
	}
	return false
}

// ValidateLoop 循环引用校验
func (api API) ValidateLoop(name string) API {
	if strings.ToLower(api.Process) == strings.ToLower(name) {
		exception.New("循环引用 %s", 400, name).Throw()
	}
	return api
}

// DefaultQueryParams 读取参数 QueryParam
func (api API) DefaultQueryParams(i int, defaults ...gou.QueryParam) gou.QueryParam {
	param := gou.QueryParam{}
	if len(defaults) > 0 {
		param = defaults[0]
	}

	if api.Default[i] == nil || len(api.Default) <= i {
		return param
	}

	param, ok := api.Default[i].(gou.QueryParam)
	if !ok {
		param, ok = gou.AnyToQueryParam(api.Default[i])
	}

	return param
}

// DefaultInt 读取参数 Int
func (api API) DefaultInt(i int, defaults ...int) int {
	value := 0
	ok := false
	if len(defaults) > 0 {
		value = defaults[0]
	}

	if api.Default[i] == nil || len(api.Default) <= i {
		return value
	}

	value, ok = api.Default[i].(int)
	if !ok {
		value = any.Of(api.Default[i]).CInt()
	}

	return value
}

// DefaultString 读取参数 String
func (api API) DefaultString(i int, defaults ...string) string {
	value := ""
	ok := false
	if len(defaults) > 0 {
		value = defaults[0]
	}

	if api.Default[i] == nil || len(api.Default) <= i {
		return value
	}

	value, ok = api.Default[i].(string)
	if !ok {
		value = any.Of(api.Default[i]).CString()
	}
	return value
}

// MergeDefaultQueryParam 合并默认查询参数
func (api API) MergeDefaultQueryParam(param gou.QueryParam, i int) gou.QueryParam {
	if len(api.Default) > i && api.Default[i] != nil {
		defaults, ok := gou.AnyToQueryParam(api.Default[i])
		if !ok {
			exception.New("参数默认值数据结构错误", 400).Ctx(api.Default[i]).Throw()
		}
		if defaults.Withs != nil {
			param.Withs = defaults.Withs
		}
		if defaults.Wheres != nil {
			if param.Wheres == nil {
				param.Wheres = []gou.QueryWhere{}
			}
			param.Wheres = append(param.Wheres, defaults.Wheres...)
		}

		if defaults.Orders != nil {
			param.Orders = append(param.Orders, defaults.Orders...)
		}
	}
	return param
}
