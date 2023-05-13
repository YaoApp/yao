package table

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/mapping"
)

// BindModel bind model
func (layout *LayoutDSL) BindModel(m *model.Model, fields *FieldsDSL, option map[string]interface{}) error {

	if option == nil {
		option = map[string]interface{}{}
	}

	formName, hasForm := option["form"]

	if layout.Primary == "" {
		layout.Primary = m.PrimaryKey
	}

	if layout.Filter == nil && len(fields.Filter) > 0 {

		layout.Filter = &FilterLayoutDSL{Columns: component.Instances{}}
		if hasForm {
			layout.Filter.Actions = component.Actions{
				{
					Title: "::Create",
					Icon:  "icon-plus",
					Width: 3,
					Action: component.ActionNodes{
						{
							"name": "OpenModal",
							"type": "Common.openModal",
							"payload": map[string]interface{}{
								"Form": map[string]interface{}{"type": "edit", "model": formName},
							},
						},
					},
				},
			}
		}

		max := 3
		curr := 0
		for _, namev := range m.ColumnNames {
			name, ok := namev.(string)

			if ok {

				if fli, has := fields.filterMap[name]; has {
					curr++
					if curr >= max {
						break
					}
					layout.Filter.Columns = append(layout.Filter.Columns, component.InstanceDSL{
						Name: fli.Key,
					})
				}
			}
		}
	}

	if layout.Table == nil && len(fields.Table) > 0 {
		layout.Table = &ViewLayoutDSL{
			Props:   component.PropsDSL{"scroll": map[string]interface{}{"x": "max-content"}},
			Columns: component.Instances{},
			Operation: OperationTableDSL{
				Hide:    true,
				Fold:    false,
				Actions: component.Actions{},
			},
		}

		if hasForm {
			layout.Table.Operation.Width = 140
			layout.Table.Operation.Hide = false
			layout.Table.Operation.Actions = append(
				layout.Table.Operation.Actions,
				[]component.ActionDSL{{
					Title: "::View",
					Icon:  "icon-eye",
					Action: component.ActionNodes{{
						"name": "OpenModal",
						"type": "Common.openModal",
						"payload": map[string]interface{}{
							"Form": map[string]interface{}{"type": "view", "model": formName},
						},
					}},
				}, {
					Title: "::Edit",
					Icon:  "icon-edit-2",
					Action: component.ActionNodes{{
						"name": "OpenModal",
						"type": "Common.openModal",
						"payload": map[string]interface{}{
							"Form": map[string]interface{}{"type": "edit", "model": formName},
						},
					}},
				}, {
					Title: "::Delete",
					Icon:  "icon-trash-2",
					Style: "danger",
					Action: component.ActionNodes{{
						"name":    "Confirm",
						"type":    "Common.confirm",
						"payload": map[string]interface{}{"title": "::Confirm", "content": "::Please confirm, the data cannot be recovered"},
					}, {
						"name":    "Delete",
						"type":    "Table.delete",
						"payload": map[string]interface{}{"model": formName},
					}},
				}}...,
			)
		}

		for _, namev := range m.ColumnNames {
			name, ok := namev.(string)
			if ok && name != "deleted_at" {
				if col, has := fields.tableMap[name]; has {
					width := 160
					if c, has := m.Columns[name]; has {
						typ := strings.ToLower(c.Type)
						if typ == "id" || strings.Contains(typ, "integer") || strings.Contains(typ, "float") {
							width = 100
						}
					}
					layout.Table.Columns = append(layout.Table.Columns, component.InstanceDSL{
						Name:  col.Key,
						Width: width,
					})
				}
			}
		}
	}

	return nil

}

// BindTable bind table
func (layout *LayoutDSL) BindTable(tab *DSL, fields *FieldsDSL) error {

	if layout.Primary == "" {
		layout.Primary = tab.Layout.Primary
	}

	if layout.Filter == nil && tab.Layout.Filter != nil {
		layout.Filter = &FilterLayoutDSL{}
		*layout.Filter = *tab.Layout.Filter
	}

	if layout.Table == nil && tab.Layout.Table != nil {
		layout.Table = &ViewLayoutDSL{}
		*layout.Table = *tab.Layout.Table
	}

	return nil
}

// Xgen trans to Xgen setting
func (layout *LayoutDSL) Xgen(data map[string]interface{}, excludes map[string]bool, mapping *mapping.Mapping) (*LayoutDSL, error) {

	clone, err := layout.Clone()
	if err != nil {
		return nil, err
	}

	if clone.Table != nil {
		if clone.Table.Props == nil {
			clone.Table.Props = component.PropsDSL{}
		}

		if _, has := clone.Table.Props["scroll"]; !has {
			clone.Table.Props["scroll"] = map[string]interface{}{"x": "max-content"}
		}
	}

	// layout.header.preset.import.actions
	if clone.Header != nil &&
		clone.Header.Preset != nil {
		if clone.Header.Preset.Import != nil &&
			clone.Header.Preset.Import.Actions != nil &&
			len(clone.Header.Preset.Import.Actions) > 0 {
			clone.Header.Preset.Import.Actions = clone.Header.Preset.Import.Actions.Filter(excludes)
		}

		// layout.header.preset.batch.columns
		if clone.Header.Preset.Batch != nil && clone.Header.Preset.Batch.Columns != nil {
			columns := []component.InstanceDSL{}
			for _, column := range clone.Header.Preset.Batch.Columns {
				id, has := mapping.Filters[column.Name]
				if !has {
					continue
				}

				if _, has := excludes[id]; has {
					continue
				}

				columns = append(columns, column)
			}
			clone.Header.Preset.Batch.Columns = columns
		}
	}

	// layout.filter.actions
	if clone.Filter != nil {
		if clone.Filter.Actions != nil && len(clone.Filter.Actions) > 0 {
			clone.Filter.Actions = clone.Filter.Actions.Filter(excludes)
		}

		if clone.Filter.Columns != nil && len(clone.Filter.Columns) > 0 {
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

	// layout.table.operation.actions
	if clone.Table != nil {
		if clone.Table.Operation.Actions != nil && len(clone.Table.Operation.Actions) > 0 {
			clone.Table.Operation.Actions = clone.Table.Operation.Actions.Filter(excludes)
		}

		if clone.Table.Columns != nil && len(clone.Table.Columns) > 0 {
			columns := []component.InstanceDSL{}
			for _, column := range clone.Table.Columns {
				id, has := mapping.Columns[column.Name]
				if !has {
					continue
				}
				if _, has := excludes[id]; has {
					continue
				}
				columns = append(columns, column)
			}
			clone.Table.Columns = columns
		}
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
