package form

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
//   GET  /api/__yao/form/:id/setting  						-> Default process: yao.form.Xgen
//   GET  /api/__yao/form/:id/find/:primary  				-> Default process: yao.form.Find $param.id $param.primary :query
//   GET  /api/__yao/form/:id/component/:xpath/:method  	-> Default process: yao.form.Component $param.id $param.xpath $param.method :query
//  POST  /api/__yao/form/:id/save  						-> Default process: yao.form.Save $param.id :payload
//  POST  /api/__yao/form/:id/create  						-> Default process: yao.form.Create $param.id :payload
//  POST  /api/__yao/form/:id/update/:primary  				-> Default process: yao.form.Update $param.id $param.primary :payload
//  POST  /api/__yao/form/:id/delete/:primary  				-> Default process: yao.form.Delete $param.id $param.primary
//
// Process:
// 	 yao.form.Setting Return the App DSL
// 	 yao.form.Xgen Return the Xgen setting
//   yao.form.Find Return the record via the given primary key
//   yao.form.Component Return the result defined in props.xProps
//   yao.form.Save Save a record, if given a primary key update, else insert
//   yao.form.Create Create a record
//   yao.form.Update update record via the given primary key
//   yao.form.Delete delete record via the given primary key
//
// Hook:
//   before:find
//   after:find
//   before:save
//   after:save
//   before:create
//   after:create
//   before:delete
//   after:delete
//   before:update
//   after:update
//

// Forms the loaded form widgets
var Forms map[string]*DSL = map[string]*DSL{}
var lock sync.Mutex

// New create a new DSL
func New(id string) *DSL {
	return &DSL{
		ID:     id,
		Fields: &FieldsDSL{Form: field.Columns{}},
		Layout: &LayoutDSL{},
		CProps: field.CloudProps{},
		Config: map[string]interface{}{},
	}
}

// LoadAndExport load table
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
	exts := []string{"*.form.yao", "*.form.json", "*.form.jsonc"}
	err := application.App.Walk("forms", func(root, file string, isdir bool) error {
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

// LoadFileSync load form dsl by file
func LoadFileSync(root string, file string) error {
	lock.Lock()
	defer lock.Unlock()
	return LoadFile(root, file)
}

// LoadFile load form dsl by file
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

	Forms[id] = dsl
	return nil
}

// LoadID load via id
func LoadID(id string, root string) error {

	file := filepath.Join("forms", share.File(id, ".form.yao"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("forms", file)
	}

	file = filepath.Join("forms", share.File(id, ".form.jsonc"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("forms", file)
	}

	file = filepath.Join("forms", share.File(id, ".form.json"))
	if exists, _ := application.App.Exists(file); exists {
		return LoadFile("forms", file)
	}

	return fmt.Errorf("form %s not found", id)
}

// LoadData load via data
func (dsl *DSL) parse(id string, root string) error {

	dsl.Root = root
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

	// Bind model / store / table / ...
	err := dsl.Bind()
	if err != nil {
		return fmt.Errorf("[Form] LoadData Bind %s %s", id, err.Error())
	}

	// mapping
	err = dsl.mapping()
	if err != nil {
		return fmt.Errorf("[Form] LoadData Mapping %s %s", id, err.Error())
	}

	// Validate
	err = dsl.Validate()
	if err != nil {
		return fmt.Errorf("[Form] LoadData Validate %s %s", id, err.Error())
	}

	Forms[id] = dsl
	return nil
}

// Get form via process or id
func Get(form interface{}) (*DSL, error) {
	id := ""
	switch form.(type) {
	case string:
		id = form.(string)
	case *process.Process:
		id = form.(*process.Process).ArgsString(0)
	default:
		return nil, fmt.Errorf("%v type does not support", form)
	}

	t, has := Forms[id]
	if !has {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return t, nil
}

// MustGet Get form via process or id thow error
func MustGet(form interface{}) *DSL {
	t, err := Get(form)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return t
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen(data map[string]interface{}, excludes map[string]bool) (map[string]interface{}, error) {

	if dsl.Layout == nil {
		dsl.Layout = &LayoutDSL{Form: &ViewLayoutDSL{}}
	}

	if dsl.Layout.Form == nil {
		dsl.Layout.Form = &ViewLayoutDSL{}
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

	onChange := map[string]interface{}{} // Hooks
	setting["fields"] = fields
	setting["config"] = dsl.Config

	for _, cProp := range dsl.CProps {
		err := cProp.Replace(setting, func(cProp component.CloudPropsDSL) interface{} {

			if cProp.Type == "Upload" || cProp.Type == "WangEditor" {
				return fmt.Sprintf("/api/__yao/form/%s%s", dsl.ID, cProp.UploadPath())
			}

			return map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/form/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		})
		if err != nil {
			return nil, err
		}

		// hooks
		if cProp.Name == "on:change" {
			field := strings.TrimPrefix(cProp.Xpath, "fields.form.")
			field = strings.TrimSuffix(field, ".edit.props")
			onChange[field] = map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/form/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		}
	}

	setting["hooks"] = map[string]interface{}{"onChange": onChange}
	setting["name"] = dsl.Name
	return setting, nil
}

// Actions get the form actions
func (dsl *DSL) Actions() []component.ActionsExport {

	res := []component.ActionsExport{}

	// layout.operation.actions
	if dsl.Layout != nil &&
		dsl.Layout.Actions != nil &&
		len(dsl.Layout.Actions) > 0 {
		res = append(res, component.ActionsExport{
			Type:    "operation",
			Xpath:   "layout.actions",
			Actions: dsl.Layout.Actions,
		})
	}
	return res
}
