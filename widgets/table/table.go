package table

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/field"
)

//
// API:
//   GET  /api/__yao/table/:id/setting  					-> Default process: yao.table.Xgen
//   GET  /api/__yao/table/:id/search  						-> Default process: yao.table.Search $param.id :query $query.page  $query.pagesize
//   GET  /api/__yao/table/:id/get  						-> Default process: yao.table.Get $param.id :query
//   GET  /api/__yao/table/:id/find/:primary  				-> Default process: yao.table.Find $param.id $param.primary :query
//   GET  /api/__yao/table/:id/component/:xpath/:method  	-> Default process: yao.table.Component $param.id $param.xpath $param.method :query
//   GET  /api/__yao/table/:id/upload/:xpath/:method  		-> Default process: yao.table.Upload $param.id $param.xpath $param.method $file.file
//   GET  /api/__yao/table/:id/download/:field  			-> Default process: yao.table.Download $param.id $param.field $query.name $query.token
//  POST  /api/__yao/table/:id/save  						-> Default process: yao.table.Save $param.id :payload
//  POST  /api/__yao/table/:id/create  						-> Default process: yao.table.Create $param.id :payload
//  POST  /api/__yao/table/:id/insert  						-> Default process: yao.table.Insert :payload
//  POST  /api/__yao/table/:id/update/:primary  			-> Default process: yao.table.Update $param.id $param.primary :payload
//  POST  /api/__yao/table/:id/update/where  				-> Default process: yao.table.UpdateWhere $param.id :query :payload
//  POST  /api/__yao/table/:id/update/in  					-> Default process: yao.table.UpdateIn $param.id $query.ids :payload
//  POST  /api/__yao/table/:id/delete/:primary  			-> Default process: yao.table.Delete $param.id $param.primary
//  POST  /api/__yao/table/:id/delete/where  				-> Default process: yao.table.DeleteWhere $param.id :query
//  POST  /api/__yao/table/:id/delete/in  					-> Default process: yao.table.DeleteIn $param.id $query.ids
//
// Process:
// 	 yao.table.Setting Return the App DSL
// 	 yao.table.Xgen Return the Xgen setting
//   yao.table.Search Return the records with pagination
//   yao.table.Get  Return the records without pagination
//   yao.table.Find Return the record via the given primary key
//   yao.table.Component Return the result defined in props
//   yao.table.Upload Upload file defined in props
//   yao.table.Download Download file defined in props
//   yao.table.Save Save a record, if given a primary key update, else insert
//   yao.table.Create Create a record
//   yao.table.Insert Insert records
//   yao.table.Update update record via the given primary key
//   yao.table.UpdateWhere update record via the given query params
//   yao.table.UpdateIn update record via the given primary key list
//   yao.table.Delete delete record via the given primary key
//   yao.table.DeleteWhere delete record via the given query params
//   yao.table.DeleteIn delete record via the given primary key list
//
// Hook:
//   before:find
//   after:find
//   before:search
//   after:search
//   before:get
//   after:get
//   before:save
//   after:save
//   before:create
//   after:create
//   before:delete
//   after:delete
//   before:insert
//   after:insert
//   before:delete-in
//   after:delete-in
//   before:delete-where
//   after:delete-where
//   before:update-in
//   after:update-in
//   before:update-where
//   after:update-where
//

// Tables the loaded table widgets
var Tables map[string]*DSL = map[string]*DSL{}
var lock sync.Mutex

// New create a new DSL
func New(id string, file string, source []byte) *DSL {
	return &DSL{
		ID:     id,
		file:   file,
		source: source,
		Fields: &FieldsDSL{Filter: field.Filters{}, Table: field.Columns{}},
		CProps: field.CloudProps{},
		Config: map[string]interface{}{},
	}
}

// LoadAndExport load table
func LoadAndExport(cfg config.Config) error {
	err := Export()
	if err != nil {
		log.Error(err.Error())
	}
	return Load(cfg)
}

// Load load table dsl
func Load(cfg config.Config) error {
	messages := []string{}
	exts := []string{"*.tab.yao", "*.tab.json", "*.tab.jsonc"}
	err := application.App.Walk("tables", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		if err := LoadFile(root, file); err != nil {
			messages = append(messages, err.Error())
		}

		return nil
	}, exts...)

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";\n"))
	}

	return err
}

// Unload unload the table
func Unload(id string) {
	delete(Tables, id)
}

// LoadID load table dsl by id
func LoadID(id string) error {

	file := filepath.Join("tables", share.File(id, ".tab.yao"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("tables", file)
	}

	file = filepath.Join("tables", share.File(id, ".tab.jsonc"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("tables", file)
	}

	file = filepath.Join("tables", share.File(id, ".tab.json"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("tables", file)
	}

	return fmt.Errorf("table %s not found", id)
}

// LoadFileSync load table dsl by file
func LoadFileSync(root string, file string) error {
	lock.Lock()
	defer lock.Unlock()
	return LoadFile(root, file)
}

// LoadFile load table dsl by file
func LoadFile(root string, file string) error {

	id := share.ID(root, file)
	data, err := application.App.Read(file)
	if err != nil {
		return err
	}

	_, err = load(data, id, file)
	if err != nil {
		return err
	}
	return nil
}

// LoadSourceSync load table dsl by source
func LoadSourceSync(source []byte, id string) (*DSL, error) {
	lock.Lock()
	defer lock.Unlock()
	return LoadSource(source, id)
}

// LoadSource load table dsl by source
func LoadSource(source []byte, id string) (*DSL, error) {
	file := filepath.Join("tables", share.File(id, ".tab.yao"))
	return load(source, id, file)
}

// LoadSource load table dsl by source
func load(source []byte, id string, file string) (*DSL, error) {
	dsl := New(id, file, source)
	err := application.Parse(file, source, dsl)
	if err != nil {
		return nil, fmt.Errorf("[%s] %s", id, err.Error())
	}

	err = dsl.parse(id)
	if err != nil {
		return nil, err
	}

	Tables[id] = dsl
	return dsl, nil
}

// parse parse table dsl source
func (dsl *DSL) parse(id string) error {

	if dsl.Action == nil {
		dsl.Action = &ActionDSL{}
	}

	dsl.Action.SetDefaultProcess()
	if dsl.Layout == nil {
		dsl.Layout = &LayoutDSL{
			Header: &HeaderLayoutDSL{
				Preset:  &PresetHeaderDSL{},
				Actions: []component.ActionDSL{},
			},
		}
	}

	if dsl.Fields == nil {
		dsl.Fields = &FieldsDSL{
			Table:     field.Columns{},
			Filter:    field.Filters{},
			filterMap: map[string]field.FilterDSL{},
			tableMap:  map[string]field.ColumnDSL{},
		}
	}

	// Bind model / store / table / ...
	err := dsl.Bind()
	if err != nil {
		return fmt.Errorf("[Table] LoadData Bind %s %s", id, err.Error())
	}

	// Mapping
	err = dsl.mapping()
	if err != nil {
		return fmt.Errorf("[Table] LoadData Mapping %s %s", id, err.Error())
	}

	// Validate
	err = dsl.Validate()
	if err != nil {
		return fmt.Errorf("[Table] LoadData Validate %s %s", id, err.Error())
	}

	Tables[id] = dsl
	return nil
}

// Get table via process or id
func Get(table interface{}) (*DSL, error) {
	id := ""
	switch table.(type) {
	case string:
		id = table.(string)
	case *process.Process:
		id = table.(*process.Process).ArgsString(0)
	default:
		return nil, fmt.Errorf("%v type does not support", table)
	}

	t, has := Tables[id]
	if !has {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return t, nil
}

// MustGet Get table via process or id thow error
func MustGet(table interface{}) *DSL {
	t, err := Get(table)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return t
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen(data map[string]interface{}, excludes map[string]bool) (map[string]interface{}, error) {

	if dsl.Config == nil {
		dsl.Config = map[string]interface{}{}
	}

	layout, err := dsl.Layout.Xgen(data, excludes, dsl.Mapping)
	if err != nil {
		return nil, err
	}

	fields, err := dsl.Fields.Xgen(layout)
	if err != nil {
		return nil, err
	}

	// full width default value
	if _, has := dsl.Config["full"]; !has {
		dsl.Config["full"] = true
	}

	// Merge the layout config
	if layout.Config != nil {
		for key, value := range layout.Config {
			dsl.Config[key] = value
		}
	}

	setting := map[string]interface{}{}
	bytes, err := jsoniter.Marshal(layout)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &setting)
	if err != nil {
		return nil, err
	}

	// replace import
	if layout.Header != nil && layout.Header.Preset != nil && layout.Header.Preset.Import != nil {
		name := layout.Header.Preset.Import.Name
		setting["header"].(map[string]interface{})["preset"].(map[string]interface{})["import"] = map[string]interface{}{
			"api": map[string]interface{}{
				"setting":               fmt.Sprintf("/api/xiang/import/%s/setting", name),
				"mapping":               fmt.Sprintf("/api/xiang/import/%s/mapping", name),
				"preview":               fmt.Sprintf("/api/xiang/import/%s/data", name),
				"import":                fmt.Sprintf("/api/xiang/import/%s", name),
				"mapping_setting_model": fmt.Sprintf("import_%s_mapping", name),
				"preview_setting_model": fmt.Sprintf("import_%s_preview", name),
			},
			"actions": layout.Header.Preset.Import.Actions,
		}
	}

	// Set Fields
	setting["fields"] = fields
	setting["config"] = dsl.Config
	for _, cProp := range dsl.CProps {
		err := cProp.Replace(setting, func(cProp component.CloudPropsDSL) interface{} {

			t := strings.ToLower(cProp.Type)
			if component.UploadComponents[t] {
				return fmt.Sprintf("/api/__yao/table/%s%s", dsl.ID, cProp.UploadPath())
			}

			return map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/table/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		})

		if err != nil {
			return nil, err
		}
	}

	setting["name"] = dsl.Name
	return setting, nil
}

// Actions get the table actions
func (dsl *DSL) Actions() []component.ActionsExport {

	res := []component.ActionsExport{}

	// layout.header.preset.import.actions
	if dsl.Layout != nil &&
		dsl.Layout.Header != nil &&
		dsl.Layout.Header.Preset != nil &&
		dsl.Layout.Header.Preset.Import != nil &&
		dsl.Layout.Header.Preset.Import.Actions != nil &&
		len(dsl.Layout.Header.Preset.Import.Actions) > 0 {

		res = append(res, component.ActionsExport{
			Type:    "import",
			Xpath:   "layout.header.preset.import.actions",
			Actions: dsl.Layout.Header.Preset.Import.Actions,
		})
	}

	// layout.filter.actions
	if dsl.Layout != nil &&
		dsl.Layout.Filter != nil &&
		dsl.Layout.Filter.Actions != nil &&
		len(dsl.Layout.Filter.Actions) > 0 {
		res = append(res, component.ActionsExport{
			Type:    "filter",
			Xpath:   "layout.filter.actions",
			Actions: dsl.Layout.Filter.Actions,
		})
	}

	// layout.table.operation.actions
	if dsl.Layout != nil &&
		dsl.Layout.Table != nil &&
		dsl.Layout.Table.Operation.Actions != nil &&
		len(dsl.Layout.Table.Operation.Actions) > 0 {
		res = append(res, component.ActionsExport{
			Type:    "operation",
			Xpath:   "layout.table.operation.actions",
			Actions: dsl.Layout.Table.Operation.Actions,
		})
	}

	return res
}

// Reload reload the table
func (dsl *DSL) Reload() (*DSL, error) {
	return LoadSourceSync(dsl.source, dsl.ID)
}

// Read read the source
func (dsl *DSL) Read() []byte {
	return dsl.source
}

// Exists check the table exists
func Exists(id string) bool {
	_, has := Tables[id]
	return has
}
