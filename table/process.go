package table

import (
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.table.Search", ProcessSearch)
	gou.RegisterProcessHandler("xiang.table.Find", ProcessFind)
	gou.RegisterProcessHandler("xiang.table.Select", ProcessSelect)
	gou.RegisterProcessHandler("xiang.table.Save", ProcessSave)
	gou.RegisterProcessHandler("xiang.table.Delete", ProcessDelete)
	gou.RegisterProcessHandler("xiang.table.Insert", ProcessInsert)
	gou.RegisterProcessHandler("xiang.table.UpdateWhere", ProcessUpdateWhere)
	gou.RegisterProcessHandler("xiang.table.DeleteWhere", ProcessDeleteWhere)
	gou.RegisterProcessHandler("xiang.table.QuickSave", ProcessQuickSave)
	gou.RegisterProcessHandler("xiang.table.UpdateIn", ProcessUpdateIn)
	gou.RegisterProcessHandler("xiang.table.DeleteIn", ProcessDeleteIn)
	gou.RegisterProcessHandler("xiang.table.Setting", ProcessSetting)
}

// ProcessSearch xiang.table.Search
// 按条件查询数据记录, 请求成功返回符合查询条件带有分页信息的数据对象
func ProcessSearch(process *gou.Process) interface{} {

	// 读取表格名称
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	table := Select(name)

	api := table.APIs["search"].ValidateLoop("xiang.table.search")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	// if process.NumOfArgsIs(5) && api.IsAllow(process.Args[4]) {
	// 	return nil
	// }

	// Before Hook
	process.Args = table.Before(table.Hooks.BeforeSearch, process.Args, process.Sid)

	// 参数表
	process.ValidateArgNums(4)
	param := api.MergeDefaultQueryParam(process.ArgsQueryParams(1), 0, process.Sid)
	log.With(log.F{"param": param}).Trace("==== ProcessSearch =============  SID: %s", process.Sid)
	page := process.ArgsInt(2, api.DefaultInt(1))
	pagesize := process.ArgsInt(3, api.DefaultInt(2))

	// 查询数据
	response := gou.NewProcess(api.Process, param, page, pagesize).Run()

	// After Hook
	return table.After(table.Hooks.AfterSearch, response, []interface{}{param, page, pagesize}, process.Sid)
}

// ProcessFind xiang.table.Find
// 按主键值查询单条数据, 请求成功返回对应主键的数据记录
func ProcessFind(process *gou.Process) interface{} {

	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["find"].ValidateLoop("xiang.table.find")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	// Before Hook
	process.Args = table.Before(table.Hooks.BeforeFind, process.Args, process.Sid)

	// 参数表
	process.ValidateArgNums(2)
	id := process.Args[1]
	param := api.MergeDefaultQueryParam(gou.QueryParam{}, 1, process.Sid)

	// 查询数据
	response := gou.NewProcess(api.Process, id, param).Run()

	// After Hook
	return table.After(table.Hooks.AfterFind, response, []interface{}{id, param}, process.Sid)
}

// ProcessSave xiang.table.Save
// 保存单条记录。如数据记录中包含主键字段则更新，不包含主键字段则创建记录；返回创建或更新的记录主键值
func ProcessSave(process *gou.Process) interface{} {

	// 读取表格名称
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["save"].ValidateLoop("xiang.table.save")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	// Before Hook
	process.Args = table.Before(table.Hooks.BeforeSave, process.Args, process.Sid)

	// 参数处理
	process.ValidateArgNums(2)

	// 查询数据
	response := gou.NewProcess(api.Process, process.Args[1]).Run()

	// After Hook
	return table.After(table.Hooks.AfterSave, response, []interface{}{process.Args[1]}, process.Sid)
}

// ProcessDelete xiang.table.Delete
// 删除指定主键值的数据记录, 请求成功返回null
func ProcessDelete(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["delete"].ValidateLoop("xiang.table.delete")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	id := process.Args[1]
	return gou.NewProcess(api.Process, id).Run()
}

// ProcessDeleteWhere xiang.table.DeleteWhere
// 按条件批量删除数据, 请求成功返回删除行数
func ProcessDeleteWhere(process *gou.Process) interface{} {

	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["delete-where"].ValidateLoop("xiang.table.DeleteWhere")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	// 批量删除
	param := api.MergeDefaultQueryParam(process.ArgsQueryParams(1), 0, process.Sid)
	if param.Limit == 0 { // 限定删除行
		param.Limit = 10
	}

	return gou.NewProcess(api.Process, param).Run()
}

// ProcessDeleteIn xiang.table.DeleteIn
// 按条件批量删除数据, 请求成功返回删除行数
func ProcessDeleteIn(process *gou.Process) interface{} {

	process.ValidateArgNums(3)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["delete-in"].ValidateLoop("xiang.table.DeleteIn")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	// 批量删除
	ids := strings.Split(process.ArgsString(1), ",")
	primary := process.ArgsString(2, "id")
	param := gou.QueryParam{
		Wheres: []gou.QueryWhere{
			{Column: primary, OP: "in", Value: ids},
		},
	}

	return gou.NewProcess(api.Process, param).Run()
}

// ProcessUpdateWhere xiang.table.UpdateWhere
// 按条件批量更新数据, 请求成功返回更新行数
func ProcessUpdateWhere(process *gou.Process) interface{} {

	process.ValidateArgNums(3)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["update-where"].ValidateLoop("xiang.table.UpdateWhere")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	// 批量更新
	param := api.MergeDefaultQueryParam(process.ArgsQueryParams(1), 0, process.Sid)
	if param.Limit == 0 { // 限定删除行
		param.Limit = 10
	}
	return gou.NewProcess(api.Process, param, process.Args[2]).Run()
}

// ProcessUpdateIn xiang.table.UpdateWhere
// 按条件批量更新数据, 请求成功返回更新行数
func ProcessUpdateIn(process *gou.Process) interface{} {

	process.ValidateArgNums(4)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["update-in"].ValidateLoop("xiang.table.UpdateIn")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	// 批量删除
	ids := strings.Split(process.ArgsString(1), ",")
	primary := process.ArgsString(2, "id")
	param := gou.QueryParam{
		Wheres: []gou.QueryWhere{
			{Column: primary, OP: "in", Value: ids},
		},
	}
	return gou.NewProcess(api.Process, param, process.Args[3]).Run()
}

// ProcessInsert xiang.table.Insert
// 插入多条数据记录，请求成功返回插入行数
func ProcessInsert(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["insert"].ValidateLoop("xiang.table.Insert")
	table.APIGuard(api.Guard, process.Sid, process.Global)
	return gou.NewProcess(api.Process, process.Args[1:]...).Run()
}

// ProcessSetting xiang.table.Setting
// 读取数据表格配置信息, 请求成功返回配置信息对象
func ProcessSetting(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	field := process.ArgsString(1)
	table := Select(name)
	api := table.APIs["setting"]
	table.APIGuard(api.Guard, process.Sid, process.Global)

	fields := strings.Split(field, ",")
	if api.ProcessIs("xiang.table.Setting") {

		setting := maps.Map{
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

	return gou.NewProcess(api.Process, fields).Run()
}

// ProcessQuickSave xiang.table.QuickSave
// 保存多条记录。如数据记录中包含主键字段则更新，不包含主键字段则创建记录；返回创建或更新的记录主键值
func ProcessQuickSave(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["quicksave"].ValidateLoop("xiang.table.quicksave")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	args := []interface{}{}
	payload := process.ArgsMap(1)
	ids := []int{}
	if payload.Has("delete") {
		if v, ok := payload["delete"].([]int); ok {
			ids = v
		} else if vany, ok := payload["delete"].([]interface{}); ok {
			for _, v := range vany {
				ids = append(ids, any.Of(v).CInt())
			}
		}
	}
	args = append(args, ids)
	args = append(args, payload.Get("data"))
	if payload.Has("query") {
		args = append(args, payload.Get("query"))
	}

	return gou.NewProcess(api.Process, args...).Run()
}

// ProcessSelect xiang.table.Select
// 单表数据查询，一般用于下拉菜单检索
func ProcessSelect(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	table := Select(name)
	api := table.APIs["select"].ValidateLoop("xiang.table.select")
	table.APIGuard(api.Guard, process.Sid, process.Global)

	// Before Hook
	process.Args = table.Before(table.Hooks.BeforeSelect, process.Args, process.Sid)

	response := gou.NewProcess(api.Process, process.Args[1:]...).Run()

	// After Hook
	return table.After(table.Hooks.AfterSelect, response, process.Args[1:], process.Sid)
}
