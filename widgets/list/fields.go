package list

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/widgets/table"
)

// BindModel bind model
func (fields *FieldsDSL) BindModel(m *model.Model) error {

	// fields.listMap = map[string]field.ColumnDSL{}

	// trans, err := field.ModelTranslist()
	// if err != nil {
	// 	return err
	// }

	// for _, col := range m.Columns {
	// 	data := col.Map()
	// 	listField, err := trans.List(col.Type, data)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	// append columns
	// 	if _, has := fields.List[listField.Key]; !has {
	// 		fields.List[listField.Key] = *listField

	// 		// PASSWORD Fields
	// 		if col.Crypt == "PASSWORD" {
	// 			if fields.List[listField.Key].View != nil {
	// 				fields.List[listField.Key].View.Compute = &component.Compute{
	// 					Process: "Hide",
	// 					Args:    []component.CArg{component.NewExp("value")},
	// 				}
	// 			}

	// 			if fields.List[listField.Key].Edit != nil {
	// 				fields.List[listField.Key].Edit.Props["type"] = "password"
	// 			}
	// 		}
	// 		fields.listMap[col.Name] = fields.List[listField.Key]
	// 	}
	// }

	// return nil
	return nil
}

// BindTable bind table
func (fields *FieldsDSL) BindTable(tab *table.DSL) error {

	return nil

	// Bind tab
	// if fields.List == nil || len(fields.List) == 0 {
	// 	fields.List = field.Columns{}
	// 	fields.listMap = map[string]field.ColumnDSL{}
	// }

	// if tab.Fields.Table != nil {
	// 	for key, list := range tab.Fields.Table {
	// 		if list.Edit == nil {
	// 			continue
	// 		}

	// 		if _, has := fields.List[key]; !has {
	// 			edit := *list.Edit
	// 			fields.List[key] = field.ColumnDSL{Key: key, Bind: list.Bind, Edit: &edit}
	// 		}
	// 	}

	// 	mapping := tab.Fields.TableMap()
	// 	for name, list := range mapping {
	// 		if _, has := fields.listMap[name]; !has {
	// 			if list.Edit == nil {
	// 				continue
	// 			}
	// 			fields.listMap[name] = fields.List[name]
	// 		}
	// 	}

	// }
	// return nil
}

// Xgen trans to xgen setting
func (fields *FieldsDSL) Xgen(layout *LayoutDSL, query map[string]interface{}) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	lists := map[string]interface{}{}
	messages := []string{}
	replacements := maps.Map{}
	if query != nil {
		replacements = maps.Of(map[string]interface{}{"$props": query}).Dot()
	}

	if layout.List != nil && layout.List.Columns != nil {

		for i, f := range layout.List.Columns {
			name := f.Name
			field, has := fields.List[name]
			if !has {
				if strings.HasPrefix(f.Name, "::") {
					name = fmt.Sprintf("$L(%s)", strings.TrimPrefix(f.Name, "::"))
					if field, has = fields.List[name]; has {
						lists[name] = field.Map()
						continue
					}
				}

				path := fmt.Sprintf("layout.columns[%d]", i)
				messages = append(messages, fmt.Sprintf("fields.list.%s not found, checking %s", f.Name, path))
			}

			if field.Edit != nil && field.Edit.Props != nil {
				if _, has := field.Edit.Props["$on:change"]; has {
					delete(field.Edit.Props, "$on:change")
				}

			}

			lists[name] = field.Map()

			// Bind Parent Data
			if query != nil {
				if field.Edit != nil && field.Edit.Props != nil {
					lists[name] = helper.Bind(lists[name], replacements)
				}
				if field.View != nil && field.View.Props != nil {
					lists[name] = helper.Bind(lists[name], replacements)
				}
			}
		}
	}

	if len(messages) > 0 {
		return nil, fmt.Errorf(strings.Join(messages, ";\n"))
	}
	res["list"] = lists
	return res, nil
}
