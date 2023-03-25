package dashboard

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/field"
)

//
// API:
//   GET  /api/__yao/dashboard/:id/setting  					-> Default process: yao.dashboard.Xgen
//   GET  /api/__yao/dashboard/:id/data 						-> Default process: yao.dashboard.Data $param.id :query
//   GET  /api/__yao/dashboard/:id/component/:xpath/:method  	-> Default process: yao.dashboard.Component $param.id $param.xpath $param.method :query
//
// Process:
// 	 yao.form.Setting Return the App DSL
// 	 yao.form.Xgen Return the Xgen setting
//   yao.form.Data Return the query data
//   yao.form.Component Return the result defined in props.xProps
//
// Hook:
//   before:data
//   after:data
//

// Dashboards the loaded dashboard widgets
var Dashboards map[string]*DSL = map[string]*DSL{}

// New create a new DSL
func New(id string) *DSL {
	return &DSL{
		ID:     id,
		Fields: &FieldsDSL{Dashboard: field.Columns{}, Filter: field.Filters{}},
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
	messages := []string{}
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	err := application.App.Walk("dashboards", func(root, file string, isdir bool) error {
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

	Dashboards[id] = dsl
	return nil
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

	// mapping
	err := dsl.mapping()
	if err != nil {
		return err
	}

	// Validate
	err = dsl.Validate()
	if err != nil {
		return err
	}

	return nil
}

// Get dashboard via process or id
func Get(dashboard interface{}) (*DSL, error) {
	id := ""
	switch dashboard.(type) {
	case string:
		id = dashboard.(string)
	case *process.Process:
		id = dashboard.(*process.Process).ArgsString(0)
	default:
		return nil, fmt.Errorf("%v type does not support", dashboard)
	}

	t, has := Dashboards[id]
	if !has {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return t, nil
}

// MustGet Get dashboard via process or id thow error
func MustGet(dashboard interface{}) *DSL {
	t, err := Get(dashboard)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return t
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen(data map[string]interface{}, excludes map[string]bool) (map[string]interface{}, error) {

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

	setting := map[string]interface{}{}
	bytes, err := jsoniter.Marshal(layout)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &setting)
	if err != nil {
		return nil, err
	}

	setting["name"] = dsl.Name
	setting["fields"] = fields
	setting["config"] = dsl.Config

	onChange := map[string]interface{}{} // Hooks
	for _, cProp := range dsl.CProps {
		err := cProp.Replace(setting, func(cProp component.CloudPropsDSL) interface{} {
			return map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/dashboard/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		})
		if err != nil {
			return nil, err
		}

		// hooks
		if cProp.Name == "on:change" {
			field := strings.TrimPrefix(cProp.Xpath, "fields.dashboard.")
			field = strings.TrimSuffix(field, ".view.props")
			field = strings.TrimSuffix(field, ".edit.props")
			onChange[field] = map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/dashboard/%s%s", dsl.ID, cProp.Path()),
				"params": cProp.Query,
			}
		}
	}
	setting["hooks"] = map[string]interface{}{"onChange": onChange}
	return setting, nil
}

// Actions get the dashboard actions
func (dsl *DSL) Actions() []component.ActionsExport {

	res := []component.ActionsExport{}

	// layout.actions
	if dsl.Layout != nil &&
		dsl.Layout.Actions != nil &&
		len(dsl.Layout.Actions) > 0 {

		res = append(res, component.ActionsExport{
			Type:    "operation",
			Xpath:   "layout.actions",
			Actions: dsl.Layout.Actions,
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
	return res
}
