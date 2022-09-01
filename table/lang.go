package table

// Lang for applying a language pack
func (table *Table) Lang(trans func(widget string, inst string, value *string) bool) {
	inst := table.Table
	widget := "table"

	trans(widget, inst, &table.Name)
	trans(widget, inst, &table.Decription)

	// Columns
	for name, column := range table.Columns {
		new := name
		trans(widget, inst, &new)
		trans(widget, inst, &column.Label)

		// Props
		transMap(widget, inst, column.Edit.Props, trans)
		transMap(widget, inst, column.View.Props, trans)
		transMap(widget, inst, column.Form.Props, trans)

		delete(table.Columns, name)
		table.Columns[new] = column
	}

	// Filters
	for name, filter := range table.Filters {
		new := name
		trans(widget, inst, &new)
		trans(widget, inst, &filter.Label)
		delete(table.Filters, name)

		// Props
		transMap(widget, inst, filter.Input.Props, trans)
		table.Filters[new] = filter
	}

	// List
	transMap(widget, inst, table.List.Layout, trans)
	transMap(widget, inst, table.List.Option, trans)

	// Edit
	transMap(widget, inst, table.Edit.Layout, trans)
	transMap(widget, inst, table.Edit.Option, trans)
}

func transMap(widget string, inst string, values map[string]interface{}, trans func(widget string, inst string, value *string) bool) {
	for key, value := range values {

		switch value.(type) {

		case string:
			v := value.(string)
			trans(widget, inst, &v)
			values[key] = v
			break

		case []interface{}:
			v := value.([]interface{})
			transArr(widget, inst, v, trans)
			values[key] = v
			break

		case map[string]interface{}:
			v := value.(map[string]interface{})
			transMap(widget, inst, v, trans)
			values[key] = v
			break
		}

	}
}

func transArr(widget string, inst string, values []interface{}, trans func(widget string, inst string, value *string) bool) {
	for key, value := range values {
		switch value.(type) {

		case string:
			v := value.(string)
			trans(widget, inst, &v)
			values[key] = v
			break

		case []interface{}:
			v := value.([]interface{})
			transArr(widget, inst, v, trans)
			values[key] = v
			break

		case map[string]interface{}:
			v := value.(map[string]interface{})
			transMap(widget, inst, v, trans)
			values[key] = v
			break
		}

	}
}
