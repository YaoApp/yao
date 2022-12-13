package chart

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/mapping"
)

// Xgen trans to Xgen setting
func (layout *LayoutDSL) Xgen(data map[string]interface{}, excludes map[string]bool, mapping *mapping.Mapping) (*LayoutDSL, error) {
	clone, err := layout.Clone()
	if err != nil {
		return nil, err
	}

	// Filter
	if clone.Filter != nil {
		if clone.Filter.Actions != nil {
			clone.Filter.Actions = clone.Filter.Actions.Filter(excludes)
		}

		if clone.Filter.Columns != nil {
			columns := []component.InstanceDSL{}
			for _, column := range clone.Filter.Columns {
				id, has := mapping.Filters[column.Name]
				if !has {
					continue
				}

				if _, has := excludes[id]; has {
					continue
				}

				columns = append(columns, column)
			}
			clone.Filter.Columns = columns
		}
	}

	// Operations
	if clone.Operation != nil && clone.Operation.Actions != nil {
		clone.Operation.Actions = clone.Operation.Actions.Filter(excludes)
	}

	// Columns
	if clone.Chart != nil && clone.Chart.Columns != nil {
		columns := []component.InstanceDSL{}
		for _, column := range clone.Chart.Columns {
			id, has := mapping.Columns[column.Name]
			if !has {
				continue
			}

			if _, has := excludes[id]; has {
				continue
			}

			columns = append(columns, column)
		}
		clone.Chart.Columns = columns
	}

	return clone, nil
}

// Clone layout for output
func (layout *LayoutDSL) Clone() (*LayoutDSL, error) {
	new := LayoutDSL{}
	bytes, err := jsoniter.Marshal(layout)
	if err != nil {
		return nil, err
	}
	err = jsoniter.Unmarshal(bytes, &new)
	if err != nil {
		return nil, err
	}
	return &new, nil
}
