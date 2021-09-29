package table

import (
	"github.com/yaoapp/gou"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.table.Search", ProcessSearch)
	gou.RegisterProcessHandler("xiang.table.Find", ProcessFind)
}

// ProcessSearch xiang.table.Search
// 按条件查询数据记录, 请求成功返回符合查询条件带有分页信息的数据对象
func ProcessSearch(process *gou.Process) interface{} {

	process.ValidateArgNums(4)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["search"].ValidateLoop("xiang.table.search")
	if process.NumOfArgsIs(5) && api.IsAllow(process.Args[4]) {
		return nil
	}

	// 参数表
	param := api.MergeDefaultQueryParam(process.ArgsQueryParams(1), 0)
	page := process.ArgsInt(2, api.DefaultInt(1))
	pagesize := process.ArgsInt(3, api.DefaultInt(2))

	return gou.NewProcess(api.Process, param, page, pagesize).Run()
}

// ProcessFind xiang.table.Find
// 按主键值查询单条数据, 请求成功返回对应主键的数据记录
func ProcessFind(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["find"].ValidateLoop("xiang.table.find")
	if process.NumOfArgsIs(3) && api.IsAllow(process.Args[2]) {
		return nil
	}

	// 参数表
	id := process.Args[1]
	param := api.MergeDefaultQueryParam(gou.QueryParam{}, 1)
	return gou.NewProcess(api.Process, id, param).Run()
}
