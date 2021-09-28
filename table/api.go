package table

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou"
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
