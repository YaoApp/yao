package table

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Tables 已载入模型
var Tables = map[string]*Table{}

// Guard Table guard
func Guard(c *gin.Context) {

	if !strings.HasPrefix(c.FullPath(), "/api/xiang/table") {
		c.Next()
		return
	}

	routes := strings.Split(c.FullPath(), "/")
	path := routes[len(routes)-1]
	name, has := c.Params.Get("name")
	if !has {
		c.Next()
		return
	}

	table, has := Tables[name]
	if !has {
		c.Next()
		return
	}

	api, has := table.APIs[path]
	if !has {
		c.Next()
		return
	}

	if api.Guard == "-" {
		c.Next()
		return
	}

	guards := strings.Split(api.Guard, ",")
	for _, guard := range guards {
		guard = strings.TrimSpace(guard)
		log.Trace("Guard Table %s %s", name, guard)
		if middleware, has := gou.HTTPGuards[guard]; has {
			middleware(c)
			continue
		}
		gou.ProcessGuard(guard)(c)
	}
}

// Load 加载数据表格
func Load(cfg config.Config) error {
	if share.BUILDIN {
		return LoadBuildIn("tables", "")
	}
	return LoadFrom(filepath.Join(cfg.Root, "tables"), "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".json", func(root, filename string) {
		name := share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := LoadTable(string(content), name)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
		}
	})

	return err
}

// LoadBuildIn 从制品中读取
func LoadBuildIn(dir string, prefix string) error {
	return nil
}

// LoadTable 载入数据表格
func LoadTable(source string, name string) (*Table, error) {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") || strings.HasPrefix(source, "fs://") {
		filename := strings.TrimPrefix(source, "file://")
		filename = strings.TrimPrefix(filename, "fs://")
		file, err := os.Open(filename)
		if err != nil {
			return nil, err
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
		// exception.Err(err, 400).Ctx(maps.Map{"name": name}).Throw()
		return &table, err
	}

	table.loadColumns()
	table.loadFilters()
	table.loadAPIs()
	Tables[name] = &table
	return Tables[name], nil
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
func (table *Table) Reload() (*Table, error) {
	new, err := LoadTable(table.Source, table.Name)
	if err != nil {
		return nil, err
	}
	*table = *new
	return table, nil
}

// loadAPIs 加载数据管理 API
func (table *Table) loadAPIs() {
	if table.Bind.Model == "" {
		return
	}
	defaults := getDefaultAPIs(table.Bind)
	defaults["setting"] = apiDefaultSetting()

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

		api.Guard = "bearer-jwt"
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
func (table *Table) Before(process string, processArgs []interface{}, sid string) []interface{} {
	if process == "" {
		return processArgs
	}
	args := []interface{}{}
	res := []interface{}{}
	if len(processArgs) > 0 {
		args = processArgs[1:]
		res = append(res, processArgs[0])
	}

	response, err := gou.NewProcess(process, args...).WithSID(sid).Exec()
	if err != nil {
		log.With(log.F{"process": process, "args": args}).Warn("Hook执行失败: %s", err.Error())
		return processArgs
	}

	if fixedArgs, ok := response.([]interface{}); ok {
		res = append(res, fixedArgs...)
		return res
	}

	log.With(log.F{"process": process, "args": args}).Warn("Hook执行失败: 无效的处理器")
	return processArgs
}

// After 运行 After hook
func (table *Table) After(process string, data interface{}, args []interface{}, sid string) interface{} {
	if process == "" {
		return data
	}
	args = append([]interface{}{data}, args...)
	response, err := gou.NewProcess(process, args...).WithSID(sid).Exec()
	if err != nil {
		log.With(log.F{"process": process, "args": args}).Warn("Hook执行失败: %s", err.Error())
		return data
	}
	return response
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
