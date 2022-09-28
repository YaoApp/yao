package field

import "github.com/yaoapp/yao/widgets/component"

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
	Bind string         `json:"bind,omitempty"`
	Link string         `json:"link,omitempty"`
	In   string         `json:"in,omitempty"`
	Out  string         `json:"out,omitempty"`
	View *component.DSL `json:"view,omitempty"`
	Edit *component.DSL `json:"edit,omitempty"`
}

// FilterDSL the field filter dsl
type FilterDSL struct {
	Bind string         `json:"bind,omitempty"`
	Edit *component.DSL `json:"edit,omitempty"`
}
