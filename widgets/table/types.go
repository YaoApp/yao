package table

import "github.com/yaoapp/yao/widgets/component"

// DSL the table DSL
type DSL struct {
	ID          string                             `json:"id,omitempty"`
	Name        string                             `json:"name,omitempty"`
	Action      *ActionDSL                         `json:"action"`
	Layout      *LayoutDSL                         `json:"layout"`
	Fields      *FieldsDSL                         `json:"fields"`
	ComputesIn  map[string]string                  `json:"-"`
	ComputesOut map[string]string                  `json:"-"`
	CProps      map[string]component.CloudPropsDSL `json:"-"`
}

// ActionDSL the table action DSL
type ActionDSL struct {
	Bind              *BindActionDSL       `json:"bind,omitempty"`
	Setting           *ProcessActionDSL    `json:"setting,omitempty"`
	Component         *ProcessActionDSL    `json:"component,omitempty"`
	Search            *ProcessActionDSL    `json:"search,omitempty"`
	Get               *ProcessActionDSL    `json:"get,omitempty"`
	Find              *ProcessActionDSL    `json:"find,omitempty"`
	Save              *ProcessActionDSL    `json:"save,omitempty"`
	Create            *ProcessActionDSL    `json:"create,omitempty"`
	Insert            *ProcessActionDSL    `json:"insert,omitempty"`
	Delete            *ProcessActionDSL    `json:"delete,omitempty"`
	DeleteIn          *ProcessActionDSL    `json:"delete-in,omitempty"`
	DeleteWhere       *ProcessActionDSL    `json:"delete-where,omitempty"`
	Update            *ProcessActionDSL    `json:"update,omitempty"`
	UpdateIn          *ProcessActionDSL    `json:"update-in,omitempty"`
	UpdateWhere       *ProcessActionDSL    `json:"update-where,omitempty"`
	BeforeFind        *BeforeHookActionDSL `json:"before:find,omitempty"`
	AfterFind         *AfterHookActionDSL  `json:"after:find,omitempty"`
	BeforeSearch      *BeforeHookActionDSL `json:"before:search,omitempty"`
	AfterSearch       *AfterHookActionDSL  `json:"after:search,omitempty"`
	BeforeGet         *BeforeHookActionDSL `json:"before:get,omitempty"`
	AfterGet          *AfterHookActionDSL  `json:"after:get,omitempty"`
	BeforeSave        *BeforeHookActionDSL `json:"before:save,omitempty"`
	AfterSave         *AfterHookActionDSL  `json:"after:save,omitempty"`
	BeforeCreate      *BeforeHookActionDSL `json:"before:create,omitempty"`
	AfterCreate       *AfterHookActionDSL  `json:"after:create,omitempty"`
	BeforeInsert      *BeforeHookActionDSL `json:"before:insert,omitempty"`
	AfterInsert       *AfterHookActionDSL  `json:"after:insert,omitempty"`
	BeforeDelete      *BeforeHookActionDSL `json:"before:delete,omitempty"`
	AfterDelete       *AfterHookActionDSL  `json:"after:delete,omitempty"`
	BeforeDeleteIn    *BeforeHookActionDSL `json:"before:delete-in,omitempty"`
	AfterDeleteIn     *AfterHookActionDSL  `json:"after:delete-in,omitempty"`
	BeforeDeleteWhere *BeforeHookActionDSL `json:"before:delete-where,omitempty"`
	AfterDeleteWhere  *AfterHookActionDSL  `json:"after:delete-where,omitempty"`
	BeforeUpdate      *BeforeHookActionDSL `json:"before:update,omitempty"`
	AfterUpdate       *AfterHookActionDSL  `json:"after:update,omitempty"`
	BeforeUpdateIn    *BeforeHookActionDSL `json:"before:update-in,omitempty"`
	AfterUpdateIn     *AfterHookActionDSL  `json:"after:update-in,omitempty"`
	BeforeUpdateWhere *BeforeHookActionDSL `json:"before:update-where,omitempty"`
	AfterUpdateWhere  *AfterHookActionDSL  `json:"after:update-where,omitempty"`
}

// BindActionDSL action.bind
type BindActionDSL struct {
	Model  string                 `json:"model,omitempty"`  // bind model
	Store  string                 `json:"store,omitempty"`  // bind store
	Table  string                 `json:"table,omitempty"`  // bind table
	Option map[string]interface{} `json:"option,omitempty"` // bind option
}

// BeforeHookActionDSL action.before:search ...
type BeforeHookActionDSL string

// AfterHookActionDSL  action.after:search ...
type AfterHookActionDSL string

// ProcessActionDSL action.search ...
type ProcessActionDSL struct {
	Name        string               `json:"-"`
	Process     string               `json:"process,omitempty"`
	ProcessBind string               `json:"bind,omitempty"`
	Guard       string               `json:"guard,omitempty"`
	Default     []interface{}        `json:"default,omitempty"`
	Disable     bool                 `json:"disable,omitempty"`
	Before      *BeforeHookActionDSL `json:"-"`
	After       *AfterHookActionDSL  `json:"-"`
}

// LayoutDSL the table layout
type LayoutDSL struct {
	Primary string           `json:"primary,omitempty"`
	Header  *HeaderLayoutDSL `json:"header,omitempty"`
	Filter  *FilterLayoutDSL `json:"filter,omitempty"`
	Table   *ViewLayoutDSL   `json:"table,omitempty"`
}

// HeaderLayoutDSL layout.header
type HeaderLayoutDSL struct {
	Preset  *PresetHeaderDSL      `json:"preset,omitempty"`
	Actions []component.ActionDSL `json:"actions,omitempty"`
}

// PresetHeaderDSL layout.header.preset
type PresetHeaderDSL struct {
	Batch  *BatchPresetDSL  `json:"batch,omitempty"`
	Import *ImportPresetDSL `json:"import,omitempty"`
}

// BatchPresetDSL layout.header.preset.batch
type BatchPresetDSL struct {
	Columns   []component.InstanceDSL `json:"columns,omitempty"`
	Deletable bool                    `json:"deletable,omitempty"`
}

// ImportPresetDSL layout.header.preset.import
type ImportPresetDSL struct {
	Name      string               `json:"name,omitempty"`
	Operation []OperationImportDSL `json:"operation,omitempty"`
}

// OperationImportDSL  layout.header.preset.import.operation[*]
type OperationImportDSL struct {
	Title string `json:"title,omitempty"`
	Link  string `json:"link,omitempty"`
}

// FilterLayoutDSL layout.filter
type FilterLayoutDSL struct {
	BtnAddText string                  `json:"btnAddText,omitempty"`
	Columns    []component.InstanceDSL `json:"columns,omitempty"`
}

// ViewLayoutDSL layout.table
type ViewLayoutDSL struct {
	Props     component.PropsDSL      `json:"props,omitempty"`
	Columns   []component.InstanceDSL `json:"columns,omitempty"`
	Operation OperationTableDSL       `json:"operation,omitempty"`
}

// OperationTableDSL layout.table.operation
type OperationTableDSL struct {
	Fold    bool                  `json:"fold,omitempty"`
	Actions []component.ActionDSL `json:"actions,omitempty"`
}

// FieldsDSL the table fields DSL
type FieldsDSL struct {
	Filter map[string]FilterFiledsDSL `json:"filter,omitempty"`
	Table  map[string]ViewFiledsDSL   `json:"table,omitempty"`
}

// FilterFiledsDSL fields.filter
type FilterFiledsDSL struct {
	Bind string         `json:"bind,omitempty"`
	Edit *component.DSL `json:"edit,omitempty"`
}

// ViewFiledsDSL fields.table
type ViewFiledsDSL struct {
	Bind string         `json:"bind,omitempty"`
	In   string         `json:"in,omitempty"`
	Out  string         `json:"out,omitempty"`
	View *component.DSL `json:"view,omitempty"`
	Edit *component.DSL `json:"edit,omitempty"`
}
