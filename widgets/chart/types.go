package chart

import (
	"github.com/yaoapp/yao/widgets/action"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/compute"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/hook"
	"github.com/yaoapp/yao/widgets/mapping"
)

// DSL the chart DSL
type DSL struct {
	ID     string                 `json:"id,omitempty"`
	Name   string                 `json:"name,omitempty"`
	Action *ActionDSL             `json:"action"`
	Layout *LayoutDSL             `json:"layout"`
	Fields *FieldsDSL             `json:"fields"`
	Config map[string]interface{} `json:"config,omitempty"`
	CProps field.CloudProps       `json:"-"`
	compute.Computable
	*mapping.Mapping
}

// ActionDSL the chart action DSL
type ActionDSL struct {
	Setting    *action.Process `json:"setting,omitempty"`
	Component  *action.Process `json:"-"`
	Data       *action.Process `json:"data,omitempty"`
	BeforeData *hook.Before    `json:"before:data,omitempty"`
	AfterData  *hook.After     `json:"after:data,omitempty"`
}

// FieldsDSL the chart fields DSL
type FieldsDSL struct {
	Filter    field.Filters `json:"filter,omitempty"`
	Chart     field.Columns `json:"chart,omitempty"`
	filterMap map[string]field.FilterDSL
	chartMap  map[string]field.ColumnDSL
}

// LayoutDSL the chart layout DSL
type LayoutDSL struct {
	Operation *OperationLayoutDSL `json:"operation,omitempty"`
	Chart     *ViewLayoutDSL      `json:"chart,omitempty"`
	Filter    *FilterLayoutDSL    `json:"filter,omitempty"`
}

// FilterLayoutDSL layout.filter
type FilterLayoutDSL struct {
	Actions component.Actions   `json:"actions,omitempty"`
	Columns component.Instances `json:"columns,omitempty"`
}

// OperationLayoutDSL layout.operation
type OperationLayoutDSL struct {
	Actions component.Actions `json:"actions,omitempty"`
}

// ViewLayoutDSL layout.form
type ViewLayoutDSL struct {
	Columns component.Instances `json:"columns,omitempty"`
}
