package table

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/widgets/field"
)

// BindModel cast model to fields
func (fields *FieldsDSL) BindModel(m *gou.Model) error {

	trans, err := field.ModelTransform()
	if err != nil {
		return err
	}

	for _, col := range m.Columns {
		data := col.Map()
		tableField, err := trans.Table(col.Type, data)
		if err != nil {
			return err
		}
		// append columns
		if _, has := fields.Table[tableField.Key]; !has {
			fields.Table[tableField.Key] = *tableField
		}

		// Index as filter
		if col.Index || col.Unique || col.Primary {
			filterField, err := trans.Filter(col.Type, data)
			if err != nil && !field.IsNotFound(err) {
				return err
			}
			if _, has := fields.Filter[filterField.Key]; !has {
				fields.Filter[tableField.Key] = *filterField
			}
		}
	}

	return nil
}

// Xgen trans to xgen setting
func (fields *FieldsDSL) Xgen(layout *LayoutDSL) (map[string]interface{}, error) {
	res := map[string]interface{}{}

	filters := map[string]interface{}{}
	tables := map[string]interface{}{}
	if layout.Filter != nil {
		for i, f := range layout.Filter.Columns {
			field, has := fields.Filter[f.Name]
			if !has {
				return nil, fmt.Errorf("fields.filter.%s not found, checking layout.filter.columns.%d.name", f.Name, i)
			}
			filters[f.Name] = field.Map()
		}
	}

	if layout.Table != nil {
		for i, f := range layout.Table.Columns {
			field, has := fields.Table[f.Name]
			if !has {
				return nil, fmt.Errorf("fields.table.%s not found, checking layout.table.columns.%d.name", f.Name, i)
			}
			tables[f.Name] = field.Map()
		}
	}

	res["filter"] = filters
	res["table"] = tables
	return res, nil
}
