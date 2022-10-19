package chart

import (
	"github.com/yaoapp/yao/widgets/action"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/hook"
)

// DSL the chart DSL
type DSL struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Action      *ActionDSL             `json:"action"`
	Layout      *LayoutDSL             `json:"layout"`
	Fields      *FieldsDSL             `json:"fields"`
	Config      map[string]interface{} `json:"config,omitempty"`
	ComputesIn  field.ComputeFields    `json:"-"`
	ComputesOut field.ComputeFields    `json:"-"`
	CProps      field.CloudProps       `json:"-"`
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
	Filter field.Filters `json:"filter,omitempty"`
	Chart  field.Columns `json:"chart,omitempty"`
}

// LayoutDSL the chart layout DSL
type LayoutDSL struct {
	Operation *OperationLayoutDSL `json:"operation,omitempty"`
	Chart     *ViewLayoutDSL      `json:"chart,omitempty"`
}

// OperationLayoutDSL layout.operation
type OperationLayoutDSL struct {
	Actions []component.ActionDSL `json:"actions,omitempty"`
}

// ViewLayoutDSL layout.form
type ViewLayoutDSL struct {
	Columns component.Instances `json:"columns,omitempty"`
}
