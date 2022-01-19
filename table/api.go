package table

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/share"
)

// apiSearchDefault search 接口默认值
func apiSearchDefault(model *gou.Model, withs map[string]gou.With) share.API {
	query := gou.QueryParam{}
	if model.MetaData.Option.Timestamps {
		query.Orders = []gou.QueryOrder{
			{Column: "created_at", Option: "desc"},
		}
	}

	if withs != nil {
		query.Withs = withs
	}

	return share.API{
		Name:    "search",
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("models.%s.Paginate", model.Name),
		Default: []interface{}{query, 1, 20},
	}
}

// apiFindDefault find 接口默认值
func apiFindDefault(model *gou.Model, withs map[string]gou.With) share.API {

	query := gou.QueryParam{}
	if withs != nil {
		query.Withs = withs
	}

	return share.API{
		Name:    "find",
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("models.%s.Find", model.Name),
		Default: []interface{}{nil, query},
	}
}

// apiDefault 接口默认值
func apiDefault(model *gou.Model, name string, process string) share.API {
	return share.API{
		Name:    name,
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("models.%s.%s", model.Name, process),
	}
}

// apiFindDefault 接口默认值
func apiDefaultWhere(model *gou.Model, withs map[string]gou.With, name string, process string) share.API {

	query := gou.QueryParam{}
	if withs != nil {
		query.Withs = withs
	}

	return share.API{
		Name:    name,
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("models.%s.%s", model.Name, process),
		Default: []interface{}{query},
	}
}

// apiDefaultSetting 数据表格配置默认值
func apiDefaultSetting() share.API {
	return share.API{
		Name:    "setting",
		Guard:   "bearer-jwt",
		Process: fmt.Sprintf("xiang.table.setting"),
	}
}
