package share

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/xlog"
)

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

// ProcessIs 检查处理器名称
func (api API) ProcessIs(name string) bool {
	return strings.ToLower(api.Process) == strings.ToLower(name)
}

// DefaultInt 读取参数 Int
func (api API) DefaultInt(i int, defaults ...int) int {
	value := 0
	ok := false
	if len(defaults) > 0 {
		value = defaults[0]
	}

	if len(api.Default) <= i || api.Default[i] == nil {
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
func (api API) MergeDefaultQueryParam(param gou.QueryParam, i int, sid string) gou.QueryParam {
	if len(api.Default) > i && api.Default[i] != nil {

		defaults := GetQueryParam(api.Default[i], sid)

		if defaults.Withs != nil {
			param.Withs = defaults.Withs
		}

		if defaults.Select != nil {
			param.Select = defaults.Select
			utils.Dump(param.Select)
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

// GetQueryParam 解析参数
func GetQueryParam(v interface{}, sid string) gou.QueryParam {
	fmt.Println("\n==== GetQueryParam =====  SID:", sid)
	data := map[string]interface{}{}
	if sid != "" {
		var err error
		ss := session.Global().ID(sid)
		data, err = ss.Dump()
		utils.Dump(data)
		if err != nil {
			xlog.Printf("读取会话信息出错 %s", err.Error())
		}
	}
	fmt.Println("==== GetQueryParam========================")
	fmt.Println("")

	v = share.Bind(v, maps.Of(data).Dot())
	param, ok := gou.AnyToQueryParam(v)
	if !ok {
		exception.New("参数默认值数据结构错误", 400).Ctx(v).Throw()
	}
	return param
}
