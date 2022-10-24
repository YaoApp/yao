package form

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/widgets/field"
)

// BindModel bind model
func (fields *FieldsDSL) BindModel(m *gou.Model) error {

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

		// append columns
		if _, has := fields.formMap[formField.Key]; !has {
			fields.Form[formField.Key] = *formField
			fields.formMap[col.Name] = fields.Form[formField.Key]
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
