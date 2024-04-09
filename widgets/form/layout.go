package form

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/mapping"
	"github.com/yaoapp/yao/widgets/table"
)

// BindModel bind model
func (layout *LayoutDSL) BindModel(m *model.Model, formID string, fields *FieldsDSL, option map[string]interface{}) {
	if layout.Primary == "" {
		layout.Primary = m.PrimaryKey
	}

	if layout.Actions == nil {
		layout.Actions = []component.ActionDSL{
			{
				Title:       "::Save",
				Icon:        "icon-check",
				Style:       "primary",
				ShowWhenAdd: true,
				Action: component.ActionNodes{{
					"name":    "Submit",
					"type":    "Form.submit",
					"payload": map[string]interface{}{},
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
					"type":    "Form.delete",
					"payload": map[string]interface{}{"model": formID},
				}, {
					"name":    "Close",
					"type":    "Common.closeModal",
					"payload": map[string]interface{}{},
				}},
			}, {
				Title:        "::Close",
				Icon:         "icon-arrow-left",
				ShowWhenAdd:  true,
				ShowWhenView: true,
				Action: component.ActionNodes{{
					"name":    "Close",
					"type":    "Common.closeModal",
					"payload": map[string]interface{}{},
				}},
			},
		}
	}

	if layout.Form == nil && len(fields.Form) > 0 {
		layout.Form = &ViewLayoutDSL{
			Props:    component.PropsDSL{},
			Sections: []SectionDSL{{Columns: []Column{}}},
		}

		columns := []Column{}
		ignoreFields := map[string]bool{"deleted_at": m.MetaData.Option.SoftDeletes}
		for _, namev := range m.ColumnNames {
			name, ok := namev.(string)
			if !ok {
				continue
			}

			ignore, has := ignoreFields[name]
			if has && ignore {
				continue
			}

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
		layout.Form.Sections = []SectionDSL{{Columns: columns}}
	}
}

// BindForm bind form
func (layout *LayoutDSL) BindForm(form *DSL, fields *FieldsDSL) error {

	if layout.Primary == "" {
		layout.Primary = form.Layout.Primary
	}

	if (layout.Actions == nil || len(layout.Actions) == 0) &&
		form.Layout.Actions != nil {
		layout.Actions = form.Layout.Actions
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

	if layout.Actions == nil {
		layout.Actions = []component.ActionDSL{
			{
				Title:       "::Save",
				Icon:        "icon-check",
				Style:       "primary",
				ShowWhenAdd: true,
				Action: component.ActionNodes{{
					"name":    "Submit",
					"type":    "Form.submit",
					"payload": map[string]interface{}{},
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
					"type":    "Form.delete",
					"payload": map[string]interface{}{"model": formID},
				}, {
					"name":    "Close",
					"type":    "Common.closeModal",
					"payload": map[string]interface{}{},
				}},
			}, {
				Title:        "::Close",
				Icon:         "icon-arrow-left",
				ShowWhenAdd:  true,
				ShowWhenView: true,
				Action: component.ActionNodes{{
					"name":    "Close",
					"type":    "Common.closeModal",
					"payload": map[string]interface{}{},
				}},
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
						tab := sections[i].Columns[j].Tabs[k]
						layout.listColumns(
							fn,
							fmt.Sprintf("%s[%d].Columns[%d].tabs[%d]", path, i, j, k),
							[]SectionDSL{tab},
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
func (layout *LayoutDSL) Xgen(data map[string]interface{}, excludes map[string]bool, mapping *mapping.Mapping) (*LayoutDSL, error) {
	clone, err := layout.Clone()
	if err != nil {
		return nil, err
	}

	// layout.actions
	if clone.Actions != nil && len(clone.Actions) > 0 {
		clone.Actions = clone.Actions.Filter(excludes)
	}

	// layout.form.sections
	if clone.Form != nil && clone.Form.Sections != nil {
		sections := []SectionDSL{}
		for _, section := range clone.Form.Sections {
			new, err := section.Filter(excludes, mapping)
			if err != nil {
				return nil, err
			}

			if len(new.Columns) > 0 {
				sections = append(sections, new)
			}
		}
		clone.Form.Sections = sections
	}

	return clone, nil
}

// Filter exclude filter
func (section SectionDSL) Filter(excludes map[string]bool, mapping *mapping.Mapping) (SectionDSL, error) {
	new := SectionDSL{Columns: []Column{}, Title: section.Title, Desc: section.Desc, Icon: section.Icon, Weight: section.Weight, Color: section.Color}
	columns, err := section.filterColumns(section.Columns, excludes, mapping)
	if err != nil {
		return new, err
	}
	new.Columns = columns
	return new, nil
}

func (section SectionDSL) filterColumns(columns []Column, excludes map[string]bool, mapping *mapping.Mapping) ([]Column, error) {

	new := []Column{}
	for i, column := range columns {

		if column.Tabs != nil {
			for j, tab := range column.Tabs {
				tabColumns, err := tab.filterColumns(columns[i].Tabs[j].Columns, excludes, mapping)
				if err != nil {
					return nil, err
				}
				column.Tabs[j].Columns = tabColumns
			}

			new = append(new, column)
			continue
		}

		id, has := mapping.Columns[column.Name]
		if !has {
			continue
		}

		if _, has := excludes[id]; has {
			continue
		}

		new = append(new, column)
	}
	return new, nil
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
