package form

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/widgets/component"
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
						"Table.delete": {"model": formID},
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
