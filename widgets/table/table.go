package table

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/environment"
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

// New create a new DSL
func New(id string) *DSL {
	return &DSL{
		ID:     id,
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

// Load load table
func Load(cfg config.Config) error {
	var root = filepath.Join(cfg.Root, "tables")
	return LoadFrom(root, "")
}

// LoadID load by id
func LoadID(id string, root string) error {
	dirs := strings.Split(id, ".")
	name := fmt.Sprintf("%s.tab.json", dirs[len(dirs)-1])
	elems := []string{root}
	elems = append(elems, dirs[0:len(dirs)-1]...)
	elems = append(elems, "tables", name)
	filename := filepath.Join(elems...)
	data, err := environment.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("[table] LoadID %s root=%s %s", id, root, err.Error())
	}
	return LoadData(data, id, root)
}

// LoadFrom load from dir
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	messages := []string{}
	err := share.Walk(dir, ".json", func(root, filename string) {
		id := prefix + share.ID(root, filename)
		data, err := environment.ReadFile(filename)
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		err = LoadData(data, id, filepath.Dir(dir))
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}
	})

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";"))
	}

	return err
}

// LoadData load table from source
func LoadData(data []byte, id string, root string) error {
	dsl := New(id)
	dsl.Root = root

	err := jsoniter.Unmarshal(data, dsl)
	if err != nil {
		return fmt.Errorf("[Table] LoadData %s %s", id, err.Error())
	}

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
		dsl.Fields = &FieldsDSL{}
	}

	// Bind model / store / table / ...
	err = dsl.Bind()
	if err != nil {
		return fmt.Errorf("[Table] LoadData Bind %s %s", id, err.Error())
	}

	// Parse
	err = dsl.Parse()
	if err != nil {
		return fmt.Errorf("[Table] LoadData Parse %s %s", id, err.Error())
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
	case *gou.Process:
		id = table.(*gou.Process).ArgsString(0)
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

// Parse Layout
func (dsl *DSL) Parse() error {

	// ComputeFields
	err := dsl.computeMapping()
	if err != nil {
		return err
	}

	// Filters
	err = dsl.Fields.Filter.CPropsMerge(dsl.CProps, func(name string, filter field.FilterDSL) (xpath string) {
		return fmt.Sprintf("fields.filter.%s.edit.props", name)
	})
	if err != nil {
		return err
	}

	// Columns
	return dsl.Fields.Table.CPropsMerge(dsl.CProps, func(name string, kind string, column field.ColumnDSL) (xpath string) {
		return fmt.Sprintf("fields.table.%s.%s.props", name, kind)
	})
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen() (map[string]interface{}, error) {

	setting, err := dsl.Layout.Xgen()
	if err != nil {
		return nil, err
	}

	fields, err := dsl.Fields.Xgen(dsl.Layout)
	if err != nil {
		return nil, err
	}

	// full width default value
	if _, has := dsl.Config["full"]; !has {
		dsl.Config["full"] = true
	}

	setting["fields"] = fields
	setting["config"] = dsl.Config
	for _, cProp := range dsl.CProps {
		err := cProp.Replace(setting, func(cProp component.CloudPropsDSL) interface{} {

			if cProp.Type == "Upload" {
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
