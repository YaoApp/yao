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
	ID        string                   `json:"id,omitempty"`
	Data      *component.CloudPropsDSL `json:"$data,omitempty"`
	Key       string                   `json:"key,omitempty"`
	Bind      string                   `json:"bind,omitempty"`
	Link      string                   `json:"link,omitempty"`
	HideLabel bool                     `json:"hideLabel,omitempty"`
	View      *component.DSL           `json:"view,omitempty"`
	Edit      *component.DSL           `json:"edit,omitempty"`
}

type aliasColumnDSL ColumnDSL

// FilterDSL the field filter dsl
type FilterDSL struct {
	ID   string         `json:"id,omitempty"`
	Key  string         `json:"key,omitempty"`
	Bind string         `json:"bind,omitempty"`
	Edit *component.DSL `json:"edit,omitempty"`
}

type aliasFilterDSL FilterDSL

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
