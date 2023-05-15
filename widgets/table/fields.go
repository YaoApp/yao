package table

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/field"
)

// TableMap get table maps
func (fields *FieldsDSL) TableMap() map[string]field.ColumnDSL {
	return fields.tableMap
}

// BindModel cast model to fields
func (fields *FieldsDSL) BindModel(m *model.Model) error {

	fields.filterMap = map[string]field.FilterDSL{}
	fields.tableMap = map[string]field.ColumnDSL{}

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

			if fields.Table == nil {
				continue
			}

			fields.Table[tableField.Key] = *tableField

			// PASSWORD Fields
			if col.Crypt == "PASSWORD" {
				if fields.Table[tableField.Key].View != nil {
					fields.Table[tableField.Key].View.Compute = &component.Compute{
						Process: "Hide",
						Args:    []component.CArg{component.NewExp("value")},
					}
				}

				if fields.Table[tableField.Key].Edit != nil {
					fields.Table[tableField.Key].Edit.Props["type"] = "password"
				}
			}

			fields.tableMap[col.Name] = fields.Table[tableField.Key]
		}

		// Index as filter
		if col.Index || col.Unique || col.Primary {

			filterField, err := trans.Filter(col.Type, data)
			if err != nil && !field.IsNotFound(err) {
				return err
			}

			if filterField == nil {
				log.Warn("[Fields.BindModel] %s.%s (%s) No matching filters found", m.Name, col.Name, col.Type)
			}

			if filterField != nil && fields.Filter != nil {
				if _, has := fields.Filter[filterField.Key]; !has {
					fields.Filter[tableField.Key] = *filterField
					fields.filterMap[col.Name] = fields.Filter[tableField.Key]
				}
			}
		}

	}

	return nil
}

// BindTable bind table
func (fields *FieldsDSL) BindTable(tab *DSL) error {

	// Bind filter
	if fields.Filter == nil || len(fields.Filter) == 0 {
		fields.Filter = tab.Fields.Filter

	} else if tab.Fields.Filter != nil {
		for key, filter := range tab.Fields.Filter {
			if _, has := fields.Filter[key]; !has {
				fields.Filter[key] = filter
			}
		}
	}

	// Bind Table
	if fields.Table == nil || len(fields.Table) == 0 {
		fields.Table = tab.Fields.Table

	} else if tab.Fields.Table != nil {
		for key, table := range tab.Fields.Table {
			if _, has := fields.Table[key]; !has {
				fields.Table[key] = table
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
			name := f.Name
			field, has := fields.Filter[name]
			if !has {
				if strings.HasPrefix(f.Name, "::") {
					name = fmt.Sprintf("$L(%s)", strings.TrimPrefix(f.Name, "::"))
					if field, has = fields.Filter[name]; has {
						filters[name] = field.Map()
						continue
					}
				}
				return nil, fmt.Errorf("fields.filter.%s not found, checking layout.filter.columns.%d.name", f.Name, i)
			}

			filters[name] = field.Map()
		}
	}

	if layout.Table != nil {
		for i, f := range layout.Table.Columns {
			name := f.Name
			field, has := fields.Table[name]
			if !has {
				if strings.HasPrefix(f.Name, "::") {
					name = fmt.Sprintf("$L(%s)", strings.TrimPrefix(f.Name, "::"))
					if field, has = fields.Table[name]; has {
						tables[name] = field.Map()
						continue
					}
				}
				return nil, fmt.Errorf("fields.table.%s not found, checking layout.table.columns.%d.name", f.Name, i)
			}
			tables[name] = field.Map()
		}
	}

	res["filter"] = filters
	res["table"] = tables
	return res, nil
}
