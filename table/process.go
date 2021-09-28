package table

import (
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/str"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.table.Search", ProcessSearch)
}

// ProcessSearch xiang.table.Search
// 按条件查询数据记录, 请求成功返回符合查询条件带有分页信息的数据对象
func ProcessSearch(process *gou.Process) interface{} {

	// 参数处理
	process.ValidateArgNums(5)
	name := any.Of(process.Args[0]).String()
	table := Select(name)
	api := table.APIs["search"]
	if api.IsGuard(process.Args[4]) {
		return nil
	}

	param := gou.QueryParam{}
	ok := false
	if process.Args[1] != nil {
		param, ok = process.Args[1].(gou.QueryParam)
		if !ok {
			exception.New("参数不是QueryParam", 400).Throw()
		}
	}

	// 查询校验
	if strings.ToLower(api.Process) == "xiang.table.search" {
		exception.New("循环引用 xiang.table.search", 400).Throw()
	}
	param = mergeQueryParam(param, api, 0)
	page := 1
	if str.Of(api.Default[1]) != "" {
		page = any.Of(api.Default[1]).CInt()
	}
	if str.Of(process.Args[2]) != "" {
		page = any.Of(process.Args[2]).CInt()
	}
	pagesize := 15
	if str.Of(api.Default[2]) != "" {
		pagesize = any.Of(api.Default[2]).CInt()
	}
	if str.Of(process.Args[3]) != "" {
		pagesize = any.Of(process.Args[3]).CInt()
	}

	return gou.NewProcess(api.Process, param, page, pagesize).Run()
}

// 合并默认查询参数
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
