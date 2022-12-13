package list

import (
	"github.com/yaoapp/yao/widgets/action"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/compute"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/hook"
	"github.com/yaoapp/yao/widgets/mapping"
)

// DSL the list DSL
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
	*mapping.Mapping
}

// ActionDSL the list action DSL
type ActionDSL struct {
	Bind       *BindActionDSL  `json:"bind,omitempty"`
	Setting    *action.Process `json:"setting,omitempty"`
	Component  *action.Process `json:"component,omitempty"`
	Upload     *action.Process `json:"upload,omitempty"`
	Download   *action.Process `json:"download,omitempty"`
	Get        *action.Process `json:"get,omitempty"`
	Save       *action.Process `json:"save,omitempty"`
	BeforeGet  *hook.Before    `json:"before:find,omitempty"`
	AfterGet   *hook.After     `json:"after:find,omitempty"`
	BeforeSave *hook.Before    `json:"before:save,omitempty"`
	AfterSave  *hook.After     `json:"after:save,omitempty"`
}

// BindActionDSL action.bind
type BindActionDSL struct {
	Model  string                 `json:"model,omitempty"`  // bind model
	Store  string                 `json:"store,omitempty"`  // bind store
	Table  string                 `json:"table,omitempty"`  // bind table
	Option map[string]interface{} `json:"option,omitempty"` // bind option
}

// LayoutDSL the list layout DSL
type LayoutDSL struct {
	List   *ViewLayoutDSL         `json:"list,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// OperationLayoutDSL layout.operation
type OperationLayoutDSL struct {
	Preset  map[string]map[string]interface{} `json:"preset,omitempty"`
	Actions []component.ActionDSL             `json:"actions,omitempty"`
}

// FieldsDSL the list fields DSL
type FieldsDSL struct {
	List    field.Columns `json:"list,omitempty"`
	listMap map[string]field.ColumnDSL
}

// ViewLayoutDSL layout.list
type ViewLayoutDSL struct {
	Props   component.PropsDSL      `json:"props,omitempty"`
	Columns []component.InstanceDSL `json:"columns,omitempty"`
}
