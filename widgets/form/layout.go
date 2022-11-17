package form

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/table"
)

// BindModel bind model
func (layout *LayoutDSL) BindModel(m *gou.Model, formID string, fields *FieldsDSL, option map[string]interface{}) {
	if layout.Primary == "" {
		layout.Primary = m.PrimaryKey
	}

	if layout.Operation == nil {
		layout.Operation = &OperationLayoutDSL{
			Preset: map[string]map[string]interface{}{"save": {}, "back": {}},
			Actions: []component.ActionDSL{
				{
					Title: "::Delete",
					Icon:  "icon-trash-2",
					Style: "danger",
					Action: map[string]component.ParamsDSL{
						"Form.delete": {"model": formID},
					},
					Confirm: &component.ConfirmActionDSL{
						Title: "::Confirm",
						Desc:  "::Please confirm, the data cannot be recovered",
					},
				},
			},
		}
	}

	if layout.Form == nil && len(fields.Form) > 0 {
		layout.Form = &ViewLayoutDSL{
			Props:    component.PropsDSL{},
			Sections: []SectionDSL{{Columns: []Column{}}},
		}

		columns := []Column{}
		for _, namev := range m.ColumnNames {
			name, ok := namev.(string)
			if ok && name != "deleted_at" {
				if col, has := fields.formMap[name]; has {
					width := 12
					if col.Edit != nil && (col.Edit.Type == "TextArea" || col.Edit.Type == "Upload") {
						width = 24
					}
					// if c, has := m.Columns[name]; has {
					// 	typ := strings.ToLower(c.Type)
					// 	if typ == "id" || strings.Contains(typ, "integer") || strings.Contains(typ, "float") {
					// 		width = 6
					// 	}
					// }
					columns = append(columns, Column{InstanceDSL: component.InstanceDSL{Name: col.Key, Width: width}})
				}
			}
		}
		layout.Form.Sections = []SectionDSL{{Columns: columns}}
	}
}

// BindForm bind form
func (layout *LayoutDSL) BindForm(form *DSL, fields *FieldsDSL) error {

	if layout.Primary == "" {
		layout.Primary = form.Layout.Primary
	}

	if layout.Operation == nil && form.Layout.Operation != nil {
		layout.Operation = &OperationLayoutDSL{
			Actions: []component.ActionDSL{},
			Preset:  map[string]map[string]interface{}{},
		}
	}

	if (layout.Operation.Actions == nil || len(layout.Operation.Actions) == 0) &&
		form.Layout.Operation.Actions != nil {
		layout.Operation.Actions = form.Layout.Operation.Actions
	}

	if layout.Operation.Preset == nil || len(layout.Operation.Preset) == 0 &&
		form.Layout.Operation.Preset != nil {
		layout.Operation.Preset = form.Layout.Operation.Preset
	}

	if layout.Form == nil && form.Layout.Form != nil {
		layout.Form = &ViewLayoutDSL{}
		*layout.Form = *form.Layout.Form
	}
	return nil
}

// BindTable bind table
func (layout *LayoutDSL) BindTable(tab *table.DSL, formID string, fields *FieldsDSL) error {

	if layout.Primary == "" {
		layout.Primary = tab.Layout.Primary
	}

	if layout.Operation == nil {
		layout.Operation = &OperationLayoutDSL{
			Preset: map[string]map[string]interface{}{"save": {}, "back": {}},
			Actions: []component.ActionDSL{
				{
					Title: "::Delete",
					Icon:  "icon-trash-2",
					Style: "danger",
					Action: map[string]component.ParamsDSL{
						"Form.delete": {"model": formID},
					},
					Confirm: &component.ConfirmActionDSL{
						Title: "::Confirm",
						Desc:  "::Please confirm, the data cannot be recovered",
					},
				},
			},
		}
	}

	if layout.Form == nil &&
		tab.Layout != nil && tab.Layout.Table != nil && tab.Layout.Table.Columns != nil &&
		len(tab.Layout.Table.Columns) > 0 {

		layout.Form = &ViewLayoutDSL{
			Props:    component.PropsDSL{},
			Sections: []SectionDSL{{Columns: []Column{}}},
		}

		columns := []Column{}
		for _, column := range tab.Fields.Table {
			if column.Edit == nil {
				continue
			}

			name := column.Key
			if col, has := fields.Form[name]; has && column.Bind != "deleted_at" {
				width := 12
				if col.Edit != nil && (col.Edit.Type == "TextArea" || col.Edit.Type == "Upload") {
					width = 24
				}
				columns = append(columns, Column{InstanceDSL: component.InstanceDSL{Name: col.Key, Width: width}})
			}
		}
		layout.Form.Sections = []SectionDSL{{Columns: columns}}
	}

	return nil
}

func (layout *LayoutDSL) listColumns(fn func(string, Column), path string, sections []SectionDSL) {
	if layout.Form == nil || layout.Form.Sections == nil {
		return
	}

	if sections == nil {
		sections = layout.Form.Sections
		path = "layout.sections"
	}

	for i := range sections {
		if sections[i].Columns != nil {
			for j := range sections[i].Columns {

				if sections[i].Columns[j].Tabs != nil {
					for k := range sections[i].Columns[j].Tabs {
						layout.listColumns(
							fn,
							fmt.Sprintf("%s[%d].Columns[%d].tabs[%d]", path, i, j, k),
							[]SectionDSL{sections[i].Columns[j].Tabs[k]},
						)
					}
					continue
				}
				if path == "layout.sections" {
					fn(fmt.Sprintf("%s[%d].Columns[%d]", path, i, j), sections[i].Columns[j])
				} else {
					fn(fmt.Sprintf("%s.Columns[%d]", path, j), sections[i].Columns[j])
				}
			}
		}
	}
}

// Xgen trans to Xgen setting
func (layout *LayoutDSL) Xgen() (map[string]interface{}, error) {
	res := map[string]interface{}{}
	data, err := jsoniter.Marshal(layout)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
