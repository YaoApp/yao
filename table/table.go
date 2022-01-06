package table

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
	"github.com/yaoapp/xiang/xlog"
)

// Tables 已载入模型
var Tables = map[string]*Table{}

// Load 加载数据表格
func Load(cfg config.Config) {
	LoadFrom(cfg.RootTable, "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) {

	if share.DirNotExists(dir) {
		return
	}

	share.Walk(dir, ".json", func(root, filename string) {
		name := share.SpecName(root, filename)
		content := share.ReadFile(filename)
		LoadTable(string(content), name)
	})
}

// LoadTable 载入数据表格
func LoadTable(source string, name string) *Table {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") || strings.HasPrefix(source, "fs://") {
		filename := strings.TrimPrefix(source, "file://")
		filename = strings.TrimPrefix(filename, "fs://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	table := Table{
		Source: source,
		Table:  name,
	}
	err := helper.UnmarshalFile(input, &table)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	table.loadColumns()
	table.loadFilters()
	table.loadAPIs()
	Tables[name] = &table
	return Tables[name]
}

// Select 读取已加载表格配置
func Select(name string) *Table {
	tab, has := Tables[name]
	if !has {
		exception.New(
			fmt.Sprintf("Table:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return tab
}

// Reload 更新数据表格配置
func (table *Table) Reload() *Table {
	*table = *LoadTable(table.Source, table.Name)
	return table
}

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
			// fmt.Printf("\n%s.APIs[%s].Default: entry\n", table.Table, name)
			if len(table.APIs[name].Default) == len(api.Default) {
				for i := range table.APIs[name].Default {
					// fmt.Printf("%s.APIs[%s].Default[%d]:%v\n", table.Table, name, i, table.APIs[name].Default[i])
					if table.APIs[name].Default[i] != nil {
						api.Default[i] = table.APIs[name].Default[i]
					}
				}
			}
		}

		defaults[name] = api
	}

	table.APIs = defaults
}

// getDefaultAPIs 读取数据模型绑定的APIs
func getDefaultAPIs(bind Bind) map[string]share.API {
	name := bind.Model
	model := gou.Select(name)
	apis := map[string]share.API{
		"search":       apiSearchDefault(model, bind.Withs),
		"find":         apiFindDefault(model, bind.Withs),
		"save":         apiDefault(model, "save", "Save"),
		"delete":       apiDefault(model, "delete", "Delete"),
		"insert":       apiDefault(model, "insert", "Insert"),
		"delete-in":    apiDefault(model, "delete-in", "DeleteWhere"),
		"delete-where": apiDefaultWhere(model, bind.Withs, "delete-where", "DeleteWhere"),
		"update-in":    apiDefault(model, "update-in", "UpdateWhere"),
		"update-where": apiDefaultWhere(model, bind.Withs, "update-where", "UpdateWhere"),
		"quicksave":    apiDefault(model, "quicksave", "EachSaveAfterDelete"), // 批量保存
		"select":       apiDefault(model, "select", "SelectOption"),           // 选择
	}

	return apis
}

// Before 运行 Before hook
func (table *Table) Before(process string, processArgs []interface{}) []interface{} {
	if process == "" {
		return processArgs
	}
	args := []interface{}{}
	res := []interface{}{}
	if len(processArgs) > 0 {
		args = processArgs[1:]
		res = append(res, processArgs[0])
	}

	response := gou.NewProcess(process, args...).Run()
	if fixedArgs, ok := response.([]interface{}); ok {
		res = append(res, fixedArgs...)
		return res
	}

	xlog.Println("无效的处理器", maps.StrAny{"process": process, "response": response})
	return processArgs
}

// After 运行 After hook
func (table *Table) After(process string, data interface{}) interface{} {
	if process == "" {
		return data
	}

	fmt.Println("After", process)
	return gou.NewProcess(process, data).Run()
}

// loadFilters 加载查询过滤器
func (table *Table) loadFilters() {
	if table.Bind.Model == "" {
		return
	}
	defaults := share.GetDefaultFilters(table.Bind.Model)
	for name, filter := range table.Filters {
		defaults[name] = filter
	}
	table.Filters = defaults
}

// loadColumns 加载字段呈现方式
func (table *Table) loadColumns() {
	if table.Bind.Model == "" {
		return
	}
	defaults := share.GetDefaultColumns(table.Bind.Model)
	for name, column := range table.Columns {
		defaults[name] = column
	}
	table.Columns = defaults
}
