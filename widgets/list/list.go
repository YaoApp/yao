package list

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/field"
)

//
// API:
//   GET  /api/__yao/list/:id/setting  						-> Default process: yao.list.Xgen
//   GET  /api/__yao/list/:id/get 							-> Default process: yao.list.Get $param.id :query
//   GET  /api/__yao/list/:id/component/:xpath/:method  	-> Default process: yao.list.Component $param.id $param.xpath $param.method :query
//  POST  /api/__yao/list/:id/save  						-> Default process: yao.list.Save $param.id :payload
//   GET  /api/__yao/list/:id/upload/:xpath/:method  		-> Default process: yao.list.Upload $param.id $param.xpath $param.method $file.file
//   GET  /api/__yao/list/:id/download/:field  				-> Default process: yao.list.Download $param.id $param.field $query.name $query.token
//
// Process:
// 	 yao.list.Setting Return the App DSL
// 	 yao.list.Xgen Return the Xgen setting
//   yao.list.Component Return the result defined in props.xProps
//   yao.list.Upload Upload file defined in props
//   yao.list.Download Download file defined in props
//   yao.list.Get Return the query record
//   yao.list.Save Save a record

//
// Hook:
//   before:get
//   after:get
//   before:save
//   after:save

// Lists the loaded list widgets
var Lists map[string]*DSL = map[string]*DSL{}

// New create a new DSL
func New(id string) *DSL {
	return &DSL{
		ID:     id,
		Fields: &FieldsDSL{List: field.Columns{}},
		Layout: &LayoutDSL{},
		CProps: field.CloudProps{},
		Config: map[string]interface{}{},
	}
}

// LoadAndExport load list
func LoadAndExport(cfg config.Config) error {
	err := Load(cfg)
	if err != nil {
		log.Error(err.Error())
	}
	return Export()
}

// Load load task
func Load(cfg config.Config) error {
	messages := []string{}
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	err := application.App.Walk("lists", func(root, file string, isdir bool) error {
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

// LoadFile load table dsl by file
func LoadFile(root string, file string) error {

	id := share.ID(root, file)
	data, err := application.App.Read(file)
	if err != nil {
		return err
	}

	dsl := New(id)
	err = application.Parse(file, data, dsl)
	if err != nil {
		return fmt.Errorf("[%s] %s", id, err.Error())
	}

	err = dsl.parse(id, root)
	if err != nil {
		return err
	}

	Lists[id] = dsl
	return nil
}

// LoadID load via id
func LoadID(id string, root string) error {
	file := filepath.Join("lists", share.File(id, ".yao"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("lists", file)
	}

	file = filepath.Join("lists", share.File(id, ".jsonc"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("lists", file)
	}

	file = filepath.Join("lists", share.File(id, ".json"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("lists", file)
	}

	return fmt.Errorf("list %s not found", id)
}

// LoadData load via data
func (dsl *DSL) parse(id string, root string) error {

	if dsl.Action == nil {
		dsl.Action = &ActionDSL{}
	}
	dsl.Action.SetDefaultProcess()

	if dsl.Layout == nil {
		dsl.Layout = &LayoutDSL{}
	}

	if dsl.Fields == nil {
		dsl.Fields = &FieldsDSL{}
	}

	// Bind model / store / list / ...
	err := dsl.Bind()
	if err != nil {
		return fmt.Errorf("[List] LoadData Bind %s %s", id, err.Error())
	}

	// Mapping
	err = dsl.mapping()
	if err != nil {
		return fmt.Errorf("[List] LoadData Mapping %s %s", id, err.Error())
	}

	// Validate
	err = dsl.Validate()
	if err != nil {
		return fmt.Errorf("[List] LoadData Validate %s %s", id, err.Error())
	}

	Lists[id] = dsl
	return nil
}

// Get list via process or id
func Get(list interface{}) (*DSL, error) {
	id := ""
	switch list.(type) {
	case string:
		id = list.(string)
	case *process.Process:
		id = list.(*process.Process).ArgsString(0)
	default:
		return nil, fmt.Errorf("%v type does not support", list)
	}

	t, has := Lists[id]
	if !has {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return t, nil
}

// MustGet Get list via process or id thow error
func MustGet(list interface{}) *DSL {
	t, err := Get(list)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return t
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen(data map[string]interface{}, excludes map[string]bool, query map[string]interface{}) (map[string]interface{}, error) {

	if dsl.Layout == nil {
		dsl.Layout = &LayoutDSL{List: &ViewLayoutDSL{}}
	}

	if dsl.Layout.List == nil {
		dsl.Layout.List = &ViewLayoutDSL{}
	}

	layout, err := dsl.Layout.Xgen(data, excludes, dsl.Mapping)
	if err != nil {
		return nil, err
	}

	fields, err := dsl.Fields.Xgen(layout, query)
	if err != nil {
		return nil, err
	}

	// ** WARNING **
	// set the full configuration by default
	// Temporary solution, Will be removed in the future
	// should be set when the list is created
	config := map[string]interface{}{}
	if dsl.Config != nil {
		for key, value := range dsl.Config {
			config[key] = value
		}
	}
	if _, has := config["full"]; !has {
		config["full"] = true
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

	onChange := map[string]interface{}{} // Hooks
	setting["fields"] = fields
	setting["config"] = config

	replacements := maps.Map{}
	if query != nil {
		replacements = maps.Of(map[string]interface{}{"$props": query}).Dot()
	}

	for _, cProp := range dsl.CProps {
		err := cProp.Replace(setting, func(cProp component.CloudPropsDSL) interface{} {

			if query != nil { // Replace Query
				newQuery := helper.Bind(cProp.Query, replacements)
				cProp.Query = newQuery.(map[string]interface{})
			}

			t := strings.ToLower(cProp.Type)
			if component.UploadComponents[t] {
				return fmt.Sprintf("/api/__yao/list/%s%s", dsl.ID, cProp.UploadPath())
			}

			return map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/list/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		})

		if err != nil {
			return nil, err
		}

		// hooks
		if cProp.Name == "on:change" {
			field := strings.TrimPrefix(cProp.Xpath, "fields.list.")
			field = strings.TrimSuffix(field, ".edit.props")
			onChange[field] = map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/list/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		}
	}
	setting["hooks"] = map[string]interface{}{"onChange": onChange}
	setting["name"] = dsl.Name
	return setting, nil
}
