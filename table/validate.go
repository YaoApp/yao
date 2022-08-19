package table

import "fmt"

// Validate 校验表格格式
func (table Table) Validate() error {
	err := table.validateList()
	if err != nil {
		return err
	}

	err = table.validateEdit()
	if err != nil {
		return err
	}

	return nil
}

func (table Table) validateEdit() error {
	if table.Edit.Layout == nil {
		return nil
	}

	fieldset, ok := table.Edit.Layout["fieldset"].([]interface{})
	if !ok {
		return fmt.Errorf("edit.layout.columns is required")
	}

	for idx, set := range fieldset {
		set, ok := set.(map[string]interface{})
		if !ok {
			return fmt.Errorf("edit.layout.fieldset.%d format error", idx)
		}

		columns, ok := set["columns"].([]interface{})
		if !ok {
			return fmt.Errorf("edit.layout.fieldset.%d.columns format error", idx)
		}

		for cidx, column := range columns {
			col, ismap := column.(map[string]interface{})
			name, isstr := column.(string)
			if !ismap && !isstr {
				return fmt.Errorf("edit.layout.fieldset.%d.columns.%d format error", idx, cidx)
			}

			if ismap {
				v, has := col["name"]
				if !has {
					return fmt.Errorf("edit.layout.fieldset.%d.columns.%d.name is required", idx, cidx)
				}
				namestr, ok := v.(string)
				if !ok {
					return fmt.Errorf("edit.layout.fieldset.%d.columns.%d.name format error", idx, cidx)
				}
				name = namestr
			}

			if _, has := table.Columns[name]; !has {
				return fmt.Errorf("edit.layout.fieldset.%d.columns.%d.name %s is not found in columns", idx, cidx, name)
			}
		}

	}

	return nil
}

func (table Table) validateList() error {

	if table.List.Layout == nil {
		return nil
	}

	// table.List.Layout
	columns, ok := table.List.Layout["columns"].([]interface{})
	if !ok {
		return fmt.Errorf("list.layout.columns is required")
	}

	for idx, column := range columns {

		col, ismap := column.(map[string]interface{})
		name, isstr := column.(string)
		if !ismap && !isstr {
			return fmt.Errorf("list.layout.columns.%d format error", idx)
		}

		if ismap {
			v, has := col["name"]
			if !has {
				return fmt.Errorf("list.layout.columns.%d.name is required", idx)
			}
			namestr, ok := v.(string)
			if !ok {
				return fmt.Errorf("list.layout.columns.%d.name format error", idx)
			}
			name = namestr
		}

		if _, has := table.Columns[name]; !has {
			return fmt.Errorf("list.layout.columns.%d.name %s is not found in columns", idx, name)
		}
	}

	filters, ok := table.List.Layout["filters"].([]interface{})
	if ok {
		for idx, filter := range filters {
			fli, ok := filter.(map[string]interface{})
			if !ok {
				return fmt.Errorf("list.layout.filters.%d format error", idx)
			}

			name, has := fli["name"]
			if !has {
				return fmt.Errorf("list.layout.filters.%d.name is required", idx)
			}

			namestr, ok := name.(string)
			if !ok {
				return fmt.Errorf("list.layout.filters.%d.name format error", idx)
			}

			if _, has := table.Filters[namestr]; !has {
				return fmt.Errorf("list.layout.filters.%d.name %s is not found in filters", idx, namestr)
			}
		}
	}

	return nil
}
