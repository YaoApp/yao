package table

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/widgets/component"
)

// BindModel bind model
func (layout *LayoutDSL) BindModel(m *gou.Model, fields *FieldsDSL, option map[string]interface{}) error {

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
					Action: map[string]component.ParamsDSL{
						"Common.historyPush": {"pathname": fmt.Sprintf("/x/Form/%s/0/edit", formName)},
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
			layout.Table.Operation.Width = 160
			layout.Table.Operation.Hide = false
			layout.Table.Operation.Actions = append(
				layout.Table.Operation.Actions,

				component.ActionDSL{
					Title: "查看",
					Icon:  "icon-eye",
					Action: map[string]component.ParamsDSL{
						"Common.openModal": {
							"width": 640,
							"Form":  map[string]interface{}{"type": "view", "model": formName},
						},
					},
				},

				component.ActionDSL{
					Title: "编辑",
					Icon:  "icon-edit-2",
					Action: map[string]component.ParamsDSL{
						"Common.openModal": {
							"width": 640,
							"Form":  map[string]interface{}{"type": "edit", "model": formName},
						},
					},
				},

				component.ActionDSL{
					Title: "删除",
					Icon:  "icon-trash-2",
					Style: "danger",
					Action: map[string]component.ParamsDSL{
						"Table.delete": {"model": formName},
					},
					Confirm: &component.ConfirmActionDSL{
						Title: "提示",
						Desc:  "确认删除，删除后数据无法恢复？",
					},
				},
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

	// replace import
	if layout.Header != nil && layout.Header.Preset != nil && layout.Header.Preset.Import != nil {
		name := layout.Header.Preset.Import.Name
		operation := layout.Header.Preset.Import.Operation
		res["header"].(map[string]interface{})["preset"].(map[string]interface{})["import"] = map[string]interface{}{
			"api": map[string]interface{}{
				"setting":               fmt.Sprintf("/api/xiang/import/%s/setting", name),
				"mapping":               fmt.Sprintf("/api/xiang/import/%s/mapping", name),
				"preview":               fmt.Sprintf("/api/xiang/import/%s/data", name),
				"import":                fmt.Sprintf("/api/xiang/import/%s", name),
				"mapping_setting_model": fmt.Sprintf("import_%s_mapping", name),
				"preview_setting_model": fmt.Sprintf("import_%s_preview", name),
			},
			"operation": operation,
		}
	}

	return res, nil
}
