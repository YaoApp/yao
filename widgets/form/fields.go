package form

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/table"
)

// BindModel bind model
func (fields *FieldsDSL) BindModel(m *model.Model) error {

	fields.formMap = map[string]field.ColumnDSL{}

	trans, err := field.ModelTransform()
	if err != nil {
		return err
	}

	for _, col := range m.Columns {
		data := col.Map()
		formField, err := trans.Form(col.Type, data)
		if err != nil {
			return err
		}

		if fields.Form == nil {
			fields.Form = field.Columns{}
		}

		// append columns
		if _, has := fields.Form[formField.Key]; !has {
			fields.Form[formField.Key] = *formField

			// PASSWORD Fields
			if col.Crypt == "PASSWORD" {
				if fields.Form[formField.Key].View != nil {
					fields.Form[formField.Key].View.Compute = &component.Compute{
						Process: "Hide",
						Args:    []component.CArg{component.NewExp("value")},
					}
				}

				if fields.Form[formField.Key].Edit != nil {
					fields.Form[formField.Key].Edit.Props["type"] = "password"
				}
			}
			fields.formMap[col.Name] = fields.Form[formField.Key]
		}
	}

	return nil
}

// BindForm bind form
func (fields *FieldsDSL) BindForm(form *DSL) error {
	// Bind Form
	if fields.Form == nil || len(fields.Form) == 0 {
		fields.Form = form.Fields.Form
	} else if form.Fields.Form != nil {
		for key, form := range form.Fields.Form {
			if _, has := fields.Form[key]; !has {
				fields.Form[key] = form
			}
		}
	}
	return nil
}

// BindTable bind table
func (fields *FieldsDSL) BindTable(tab *table.DSL) error {

	// Bind tab
	if fields.Form == nil || len(fields.Form) == 0 {
		fields.Form = field.Columns{}
	}

	if fields.formMap == nil {
		fields.formMap = map[string]field.ColumnDSL{}
	}

	if tab.Fields.Table != nil {
		for key, form := range tab.Fields.Table {
			if form.Edit == nil {
				continue
			}

			if _, has := fields.Form[key]; !has {
				edit := *form.Edit
				fields.Form[key] = field.ColumnDSL{Key: key, Bind: form.Bind, Edit: &edit}
			}
		}

		mapping := tab.Fields.TableMap()
		for name, form := range mapping {
			if _, has := fields.formMap[name]; !has {
				if form.Edit == nil {
					continue
				}
				fields.formMap[name] = fields.Form[name]
			}
		}

	}
	return nil
}

// Xgen trans to xgen setting
func (fields *FieldsDSL) Xgen(layout *LayoutDSL) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	forms := map[string]interface{}{}
	messages := []string{}
	if layout.Form != nil && layout.Form.Sections != nil {

		layout.listColumns(func(path string, f Column) {

			name := f.Name
			field, has := fields.Form[name]
			if !has {
				if strings.HasPrefix(f.Name, "::") {
					name = fmt.Sprintf("$L(%s)", strings.TrimPrefix(f.Name, "::"))
					if field, has = fields.Form[name]; has {
						forms[name] = field.Map()
						return
					}
				}
				messages = append(messages, fmt.Sprintf("fields.form.%s not found, checking %s", f.Name, path))
				return
			}

			if field.Edit != nil && field.Edit.Props != nil {
				if _, has := field.Edit.Props["$on:change"]; has {
					delete(field.Edit.Props, "$on:change")
				}
			}

			forms[name] = field.Map()
		}, "", nil)
	}

	if len(messages) > 0 {
		return nil, fmt.Errorf(strings.Join(messages, ";\n"))
	}
	res["form"] = forms
	return res, nil
}
