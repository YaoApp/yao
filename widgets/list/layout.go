package list

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/widgets/table"
)

// BindModel bind model
func (layout *LayoutDSL) BindModel(m *gou.Model, listID string, fields *FieldsDSL, option map[string]interface{}) {
	// if layout.Primary == "" {
	// 	layout.Primary = m.PrimaryKey
	// }

	// if layout.Operation == nil {
	// 	layout.Operation = &OperationLayoutDSL{
	// 		Preset: map[string]map[string]interface{}{"save": {}, "back": {}},
	// 		Actions: []component.ActionDSL{
	// 			{
	// 				Title: "::Delete",
	// 				Icon:  "icon-trash-2",
	// 				Style: "danger",
	// 				Action: map[string]component.ParamsDSL{
	// 					"List.delete": {"model": listID},
	// 				},
	// 				Confirm: &component.ConfirmActionDSL{
	// 					Title: "::Confirm",
	// 					Desc:  "::Please confirm, the data cannot be recovered",
	// 				},
	// 			},
	// 		},
	// 	}
	// }

	// if layout.List == nil && len(fields.List) > 0 {
	// 	layout.List = &ViewLayoutDSL{
	// 		Props:    component.PropsDSL{},
	// 		Sections: []SectionDSL{{Columns: []Column{}}},
	// 	}

	// 	columns := []Column{}
	// 	for _, namev := range m.ColumnNames {
	// 		name, ok := namev.(string)
	// 		if ok && name != "deleted_at" {
	// 			if col, has := fields.listMap[name]; has {
	// 				width := 12
	// 				if col.Edit != nil && (col.Edit.Type == "TextArea" || col.Edit.Type == "Upload") {
	// 					width = 24
	// 				}
	// 				// if c, has := m.Columns[name]; has {
	// 				// 	typ := strings.ToLower(c.Type)
	// 				// 	if typ == "id" || strings.Contains(typ, "integer") || strings.Contains(typ, "float") {
	// 				// 		width = 6
	// 				// 	}
	// 				// }
	// 				columns = append(columns, Column{InstanceDSL: component.InstanceDSL{Name: col.Key, Width: width}})
	// 			}
	// 		}
	// 	}
	// 	layout.List.Sections = []SectionDSL{{Columns: columns}}
	// }
}

// BindTable bind table
func (layout *LayoutDSL) BindTable(tab *table.DSL, listID string, fields *FieldsDSL) error {

	// if layout.Primary == "" {
	// 	layout.Primary = tab.Layout.Primary
	// }

	// if layout.Operation == nil {
	// 	layout.Operation = &OperationLayoutDSL{
	// 		Preset: map[string]map[string]interface{}{"save": {}, "back": {}},
	// 		Actions: []component.ActionDSL{
	// 			{
	// 				Title: "::Delete",
	// 				Icon:  "icon-trash-2",
	// 				Style: "danger",
	// 				Action: map[string]component.ParamsDSL{
	// 					"List.delete": {"model": listID},
	// 				},
	// 				Confirm: &component.ConfirmActionDSL{
	// 					Title: "::Confirm",
	// 					Desc:  "::Please confirm, the data cannot be recovered",
	// 				},
	// 			},
	// 		},
	// 	}
	// }

	// if layout.List == nil &&
	// 	tab.Layout != nil && tab.Layout.Table != nil && tab.Layout.Table.Columns != nil &&
	// 	len(tab.Layout.Table.Columns) > 0 {

	// 	layout.List = &ViewLayoutDSL{
	// 		Props:    component.PropsDSL{},
	// 		Sections: []SectionDSL{{Columns: []Column{}}},
	// 	}

	// 	columns := []Column{}
	// 	for _, column := range tab.Fields.Table {
	// 		if column.Edit == nil {
	// 			continue
	// 		}

	// 		name := column.Key
	// 		if col, has := fields.List[name]; has && column.Bind != "deleted_at" {
	// 			width := 12
	// 			if col.Edit != nil && (col.Edit.Type == "TextArea" || col.Edit.Type == "Upload") {
	// 				width = 24
	// 			}
	// 			columns = append(columns, Column{InstanceDSL: component.InstanceDSL{Name: col.Key, Width: width}})
	// 		}
	// 	}
	// 	layout.List.Sections = []SectionDSL{{Columns: columns}}
	// }

	return nil
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
