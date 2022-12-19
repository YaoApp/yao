package dashboard

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

	// Actions
	if clone.Actions != nil {
		clone.Actions = clone.Actions.Filter(excludes)
	}

	// Columns
	if clone.Dashboard != nil && clone.Dashboard.Columns != nil {
		columns := []component.InstanceDSL{}
		for _, column := range clone.Dashboard.Columns {

			if column.Rows != nil {
				new := component.InstanceDSL{Rows: []component.InstanceDSL{}}
				if column.Width != nil {
					new.Width = column.Width
				}

				for _, column := range column.Rows {
					id, has := mapping.Columns[column.Name]
					if !has {
						continue
					}

					if _, has := excludes[id]; has {
						continue
					}
					new.Rows = append(new.Rows, column)
				}

				if len(new.Rows) > 0 {
					columns = append(columns, new)
				}
				continue
			}

			id, has := mapping.Columns[column.Name]
			if !has {
				continue
			}

			if _, has := excludes[id]; has {
				continue
			}

			columns = append(columns, column)
		}
		clone.Dashboard.Columns = columns
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
