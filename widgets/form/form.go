package form

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
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
		return err
	}
	return Export()
}

// Load load task
func Load(cfg config.Config) error {
	var root = filepath.Join(cfg.Root, "forms")
	return LoadFrom(root, "")
}

// LoadFrom load from dir
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	messages := []string{}
	err := share.Walk(dir, ".json", func(root, filename string) {
		id := prefix + share.ID(root, filename)
		data := share.ReadFile(filename)
		dsl := New(id)
		err := jsoniter.Unmarshal(data, dsl)
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

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
		err = dsl.Bind()
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		// Parse
		err = dsl.Parse()
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		// Validate
		err = dsl.Validate()
		if err != nil {
			messages = append(messages, fmt.Sprintf("[%s] %s", id, err.Error()))
			return
		}

		Forms[id] = dsl
	})

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";"))
	}

	return err
}

// Get form via process or id
func Get(form interface{}) (*DSL, error) {
	id := ""
	switch form.(type) {
	case string:
		id = form.(string)
	case *gou.Process:
		id = form.(*gou.Process).ArgsString(0)
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

// Parse Layout
func (dsl *DSL) Parse() error {

	// ComputeFields
	err := dsl.computeMapping()
	if err != nil {
		return err
	}

	// Columns
	return dsl.Fields.Form.CPropsMerge(dsl.CProps, func(name string, kind string, column field.ColumnDSL) (xpath string) {
		return fmt.Sprintf("fields.form.%s.%s.props", name, kind)
	})
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen() (map[string]interface{}, error) {

	if dsl.Layout == nil {
		dsl.Layout = &LayoutDSL{Form: &ViewLayoutDSL{}}
	}

	if dsl.Layout.Form == nil {
		dsl.Layout.Form = &ViewLayoutDSL{}
	}

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
			return map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/form/%s/component/%s/%s", dsl.ID, cProp.Xpath, cProp.Name),
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
