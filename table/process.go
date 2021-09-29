package table

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/maps"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.table.Search", ProcessSearch)
	gou.RegisterProcessHandler("xiang.table.Find", ProcessFind)
	gou.RegisterProcessHandler("xiang.table.Save", ProcessSave)
	gou.RegisterProcessHandler("xiang.table.Delete", ProcessDelete)
	gou.RegisterProcessHandler("xiang.table.Setting", ProcessSetting)
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

// ProcessSave xiang.table.Save
// 保存单条记录。如数据记录中包含主键字段则更新，不包含主键字段则创建记录；返回创建或更新的记录主键值
func ProcessSave(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["save"].ValidateLoop("xiang.table.save")
	if process.NumOfArgsIs(3) && api.IsAllow(process.Args[2]) {
		return nil
	}
	return gou.NewProcess(api.Process, process.Args[1]).Run()
}

// ProcessDelete xiang.table.Delete
// 删除指定主键值的数据记录, 请求成功返回null
func ProcessDelete(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["delete"].ValidateLoop("xiang.table.delete")
	if process.NumOfArgsIs(3) && api.IsAllow(process.Args[2]) {
		return nil
	}

	id := process.Args[1]
	return gou.NewProcess(api.Process, id).Run()
}

// ProcessSetting xiang.table.Setting
// 删除指定主键值的数据记录, 请求成功返回null
func ProcessSetting(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["setting"]
	if process.NumOfArgsIs(2) && api.IsAllow(process.Args[1]) {
		return nil
	}
	if api.ProcessIs("xiang.table.Setting") {
		return maps.Map{
			"name":       table.Name,
			"title":      table.Title,
			"decription": table.Decription,
			"columns":    table.Columns,
			"filters":    table.Filters,
			"list":       table.List,
			"edit":       table.Edit,
			"view":       table.View,
			"insert":     table.Insert,
		}
	}

	return gou.NewProcess(api.Process).Run()
}
