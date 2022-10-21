package chart

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
//   GET  /api/__yao/chart/:id/setting  					-> Default process: yao.chart.Xgen
//   GET  /api/__yao/chart/:id/data 						-> Default process: yao.chart.Data $param.id :query
//   GET  /api/__yao/chart/:id/component/:xpath/:method  	-> Default process: yao.chart.Component $param.id $param.xpath $param.method :query
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

// Charts the loaded chart widgets
var Charts map[string]*DSL = map[string]*DSL{}

// New create a new DSL
func New(id string) *DSL {
	return &DSL{
		ID:          id,
		Fields:      &FieldsDSL{Chart: field.Columns{}, Filter: field.Filters{}},
		CProps:      field.CloudProps{},
		ComputesIn:  field.ComputeFields{},
		ComputesOut: field.ComputeFields{},
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
	var root = filepath.Join(cfg.Root, "charts")
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

		Charts[id] = dsl
	})

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";"))
	}

	return err
}

// Get chart via process or id
func Get(chart interface{}) (*DSL, error) {
	id := ""
	switch chart.(type) {
	case string:
		id = chart.(string)
	case *gou.Process:
		id = chart.(*gou.Process).ArgsString(0)
	default:
		return nil, fmt.Errorf("%v type does not support", chart)
	}

	t, has := Charts[id]
	if !has {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return t, nil
}

// MustGet Get chart via process or id thow error
func MustGet(chart interface{}) *DSL {
	t, err := Get(chart)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}
	return t
}

// Parse Layout
func (dsl *DSL) Parse() error {

	// ComputeFields
	// dsl.Fields.Chart.ComputeFieldsMerge(dsl.ComputesIn, dsl.ComputesOut)

	// Filters
	err := dsl.Fields.Filter.CPropsMerge(dsl.CProps, func(name string, filter field.FilterDSL) (xpath string) {
		return fmt.Sprintf("fields.filter.%s.edit.props", name)
	})

	if err != nil {
		return err
	}

	// Columns
	return dsl.Fields.Chart.CPropsMerge(dsl.CProps, func(name string, kind string, column field.ColumnDSL) (xpath string) {
		return fmt.Sprintf("fields.chart.%s.%s.props", name, kind)
	})
}

// Xgen trans to xgen setting
func (dsl *DSL) Xgen() (map[string]interface{}, error) {

	setting, err := dsl.Layout.Xgen()
	if err != nil {
		return nil, err
	}

	fields, err := dsl.Fields.Xgen()
	if err != nil {
		return nil, err
	}

	setting["fields"] = fields
	setting["config"] = dsl.Config
	for _, cProp := range dsl.CProps {
		err := cProp.Replace(setting, func(cProp component.CloudPropsDSL) interface{} {
			return map[string]interface{}{
				"api":    fmt.Sprintf("/api/__yao/chart/%s/component/%s/%s", dsl.ID, cProp.Xpath, cProp.Name),
				"params": cProp.Query,
			}
		})

		if err != nil {
			return nil, err
		}
	}

	return setting, nil
}
