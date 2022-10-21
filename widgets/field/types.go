package field

import (
	"github.com/yaoapp/yao/widgets/component"
)

// Filters the filters DSL
type Filters map[string]FilterDSL

// Columns the columns DSL
type Columns map[string]ColumnDSL

// ComputeFields the Compute filelds
type ComputeFields map[string]string

// CloudProps the cloud props
type CloudProps map[string]component.CloudPropsDSL

// ColumnDSL the field column dsl
type ColumnDSL struct {
	Key  string         `json:"key,omitempty"`
	Bind string         `json:"bind,omitempty"`
	Link string         `json:"link,omitempty"`
	In   Compute        `json:"in,omitempty"`
	Out  Compute        `json:"out,omitempty"`
	View *component.DSL `json:"view,omitempty"`
	Edit *component.DSL `json:"edit,omitempty"`
}

// FilterDSL the field filter dsl
type FilterDSL struct {
	Key  string         `json:"key,omitempty"`
	Bind string         `json:"bind,omitempty"`
	Edit *component.DSL `json:"edit,omitempty"`
}

// Compute the compute filed
type Compute string

// Transform the field transform
type Transform struct {
	Variables map[string]interface{}    `json:"variables,omitempty"`
	Aliases   map[string]string         `json:"aliases,omitempty"`
	Fields    map[string]TransformField `json:"fields,omitempty"`
}

// TransformField the transform.types[*]
type TransformField struct {
	Filter *FilterDSL `json:"filter,omitempty"`
	Form   *ColumnDSL `json:"form,omitempty"`
	Table  *ColumnDSL `json:"table,omitempty"`
}
