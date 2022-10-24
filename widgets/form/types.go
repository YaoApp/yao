package form

import (
	"github.com/yaoapp/yao/widgets/action"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/compute"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/hook"
)

// DSL the form DSL
type DSL struct {
	ID     string                 `json:"id,omitempty"`
	Root   string                 `json:"-"`
	Name   string                 `json:"name,omitempty"`
	Action *ActionDSL             `json:"action"`
	Layout *LayoutDSL             `json:"layout"`
	Fields *FieldsDSL             `json:"fields"`
	Config map[string]interface{} `json:"config,omitempty"`
	CProps field.CloudProps       `json:"-"`
	compute.Computable
}

// ActionDSL the form action DSL
type ActionDSL struct {
	Bind         *BindActionDSL  `json:"bind,omitempty"`
	Setting      *action.Process `json:"setting,omitempty"`
	Component    *action.Process `json:"component,omitempty"`
	Find         *action.Process `json:"find,omitempty"`
	Save         *action.Process `json:"save,omitempty"`
	Update       *action.Process `json:"update,omitempty"`
	Create       *action.Process `json:"create,omitempty"`
	Delete       *action.Process `json:"delete,omitempty"`
	BeforeFind   *hook.Before    `json:"before:find,omitempty"`
	AfterFind    *hook.After     `json:"after:find,omitempty"`
	BeforeSave   *hook.Before    `json:"before:save,omitempty"`
	AfterSave    *hook.After     `json:"after:save,omitempty"`
	BeforeCreate *hook.Before    `json:"before:create,omitempty"`
	AfterCreate  *hook.After     `json:"after:create,omitempty"`
	BeforeDelete *hook.Before    `json:"before:delete,omitempty"`
	AfterDelete  *hook.After     `json:"after:delete,omitempty"`
	BeforeUpdate *hook.Before    `json:"before:update,omitempty"`
	AfterUpdate  *hook.After     `json:"after:update,omitempty"`
}

// BindActionDSL action.bind
type BindActionDSL struct {
	Model  string                 `json:"model,omitempty"`  // bind model
	Store  string                 `json:"store,omitempty"`  // bind store
	Table  string                 `json:"table,omitempty"`  // bind table
	Form   string                 `json:"form,omitempty"`   // bind form
	Option map[string]interface{} `json:"option,omitempty"` // bind option
}

// LayoutDSL the form layout DSL
type LayoutDSL struct {
	Primary   string                 `json:"primary,omitempty"`
	Operation *OperationLayoutDSL    `json:"operation,omitempty"`
	Form      *ViewLayoutDSL         `json:"form,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// OperationLayoutDSL layout.operation
type OperationLayoutDSL struct {
	Preset  map[string]map[string]interface{} `json:"preset,omitempty"`
	Actions []component.ActionDSL             `json:"actions,omitempty"`
}

// FieldsDSL the form fields DSL
type FieldsDSL struct {
	Form    field.Columns `json:"form,omitempty"`
	formMap map[string]field.ColumnDSL
}

// ViewLayoutDSL layout.form
type ViewLayoutDSL struct {
	Props    component.PropsDSL `json:"props,omitempty"`
	Sections []SectionDSL       `json:"sections,omitempty"`
}

// SectionDSL layout.form.sections[*]
type SectionDSL struct {
	Title   string   `json:"title,omitempty"`
	Desc    string   `json:"desc,omitempty"`
	Columns []Column `json:"columns,omitempty"`
}

// Column table columns
type Column struct {
	Tabs []SectionDSL `json:"tabs,omitempty"`
	component.InstanceDSL
}
