package table

import (
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.table.Search", processSearch)
}

// processSearch 用户检索
func processSearch(process *gou.Process) interface{} {
	process.ValidateArgNums(4)
	name := any.Of(process.Args[0]).String()
	page := any.Of(process.Args[2]).CInt()
	pagesize := any.Of(process.Args[3]).CInt()

	param := gou.QueryParam{}
	ok := false
	if process.Args[1] != nil {
		param, ok = process.Args[1].(gou.QueryParam)
		if !ok {
			exception.New("参数不是QueryParam", 400).Throw()
		}
	}

	table := Select(name)
	api := table.APIs["search"]
	param = mergeQueryParam(param, api, 0)

	if strings.ToLower(api.Process) == "xiang.table.search" {
		exception.New("循环引用 xiang.table.search", 400).Throw()
	}

	return gou.NewProcess(api.Process, param, page, pagesize).Run()
}

func mergeQueryParam(param gou.QueryParam, api API, i int) gou.QueryParam {
	if len(api.Default) > i && api.Default[i] != nil {
		defaults, ok := api.Default[i].(gou.QueryParam)
		if !ok {
			exception.New("参数默认值数据结构错误", 400).Throw()
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
