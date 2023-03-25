package compute

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/widgets/field"
)

var views = map[string]bool{"find": true, "get": true, "search": true, "data": true}

// ComputeEdit edit compute edit
func (c *Computable) ComputeEdit(name string, process *process.Process, args []interface{}, getField func(string) (*field.ColumnDSL, string, string, error)) error {
	namer := strings.Split(strings.ToLower(name), ".")
	name = namer[len(namer)-1]

	switch name {
	case "save", "create":
		if len(args) == 0 {
			return nil
		}
		switch args[0].(type) {
		case maps.MapStr:
			return c.editRow(process, args[0].(maps.MapStr), getField)

		case map[string]interface{}:
			return c.editRow(process, args[0].(map[string]interface{}), getField)
		}
		return nil

	case "update", "updatewhere", "updatein":
		if len(args) < 2 {
			return nil
		}

		switch args[1].(type) {
		case maps.MapStr:
			return c.editRow(process, args[1].(maps.MapStr), getField)

		case map[string]interface{}:
			return c.editRow(process, args[1].(map[string]interface{}), getField)
		}
		return nil

	case "insert":
		if len(args) < 2 {
			return nil
		}

		if columns, ok := args[0].([]interface{}); ok {
			new := []string{}
			for _, col := range columns {
				new = append(new, fmt.Sprintf("%v", col))
			}
			args[0] = new
		}

		if values, ok := args[1].([]interface{}); ok {
			new := [][]interface{}{}
			for _, value := range values {
				arr, ok := value.([]interface{})
				if !ok {
					return fmt.Errorf("args[1] is not a [][] %s", reflect.ValueOf(args[1]).Type().Name())
				}
				new = append(new, arr)
			}
			args[1] = new
		}

		if _, ok := args[0].([]string); !ok {
			return fmt.Errorf("args[0] is not a []string %s", reflect.ValueOf(args[0]).Type().Name())
		}

		if _, ok := args[1].([][]interface{}); !ok {
			return fmt.Errorf("args[1] is not a [][] %s", reflect.ValueOf(args[1]).Type().Name())
		}

		return c.editRows(process, args[0].([]string), args[1].([][]interface{}), getField)
	}

	return nil
}

// EditRow edit row
func (c *Computable) editRow(process *process.Process, res map[string]interface{}, getField func(string) (*field.ColumnDSL, string, string, error)) error {

	messages := []string{}
	row := maps.MapOf(res).Dot()
	data := maps.StrAny{"row": row}.Dot()
	for key := range res {
		if computes, has := c.Computes.Edit[key]; has {
			unit := computes[0]
			field, path, id, err := getField(unit.Name)
			if err != nil {
				messages = append(messages, err.Error())
				continue
			}

			data.Set("id", id)
			data.Set("value", res[key])
			data.Set("path", fmt.Sprintf("%s.%s", path, unit.Name))
			data.Merge(any.MapOf(field.Edit.Map()).MapStrAny.Dot())
			new, err := field.Edit.Compute.Value(data, process.Sid, process.Global)
			if err != nil {
				messages = append(messages, fmt.Sprintf("%s.%s bind: %s, value: %v error: %s", path, unit.Name, key, res[key], err.Error()))
				continue
			}
			res[key] = new
		}
	}

	if len(messages) > 0 {
		return fmt.Errorf("\n%s", strings.Join(messages, ";\n"))
	}

	return nil
}

// EditRows edit row
func (c *Computable) editRows(process *process.Process, columns []string, res [][]interface{}, getField func(string) (*field.ColumnDSL, string, string, error)) error {

	messages := []string{}
	keys := map[string]int{}
	for i, name := range columns {
		keys[name] = i
	}

	for i := range res {
		if len(keys) != len(res[i]) {
			continue
		}

		row := map[string]interface{}{}
		for i, v := range res[i] {
			row[columns[i]] = v
		}

		err := c.editRow(process, row, getField)
		if err != nil {
			messages = append(messages, err.Error())
		}

		for k, v := range row {
			res[i][keys[k]] = v
		}
	}

	if len(messages) > 0 {
		return fmt.Errorf("\n%s", strings.Join(messages, ";\n"))
	}

	return nil
}

// ComputeView view view
func (c *Computable) ComputeView(name string, process *process.Process, res interface{}, getField func(string) (*field.ColumnDSL, string, string, error)) error {

	namer := strings.Split(strings.ToLower(name), ".")
	name = namer[len(namer)-1]
	if _, has := views[name]; !has {
		return nil
	}

	switch res.(type) {
	case []maps.MapStrAny, []interface{}:
		return c.viewRows(name, process, res, getField)

	case map[string]interface{}:
		return c.viewRow(name, process, res.(map[string]interface{}), getField)

	case maps.MapStrAny:
		return c.viewRow(name, process, res.(maps.MapStrAny), getField)
	}

	return fmt.Errorf("res should be a map or array, but got a %s", reflect.ValueOf(res).Kind().String())
}

// ViewRows viewrows
func (c *Computable) viewRows(name string, process *process.Process, res interface{}, getField func(string) (*field.ColumnDSL, string, string, error)) error {
	switch res.(type) {

	case []interface{}:
		messages := []string{}
		for i := range res.([]interface{}) {
			err := c.ComputeView(name, process, res.([]interface{})[i], getField)
			if err != nil {
				messages = append(messages, err.Error())
			}
		}
		if len(messages) > 0 {
			return fmt.Errorf("\n%s", strings.Join(messages, ";\n"))
		}
		return nil

	case []maps.MapStrAny:
		messages := []string{}
		for i := range res.([]maps.MapStrAny) {
			err := c.ComputeView(name, process, res.([]maps.MapStrAny)[i], getField)
			if err != nil {
				messages = append(messages, fmt.Sprintf("%d %s", i, err.Error()))
			}
		}
		if len(messages) > 0 {
			return fmt.Errorf("\n%s", strings.Join(messages, ";\n"))
		}
		return nil
	}

	return nil
}

// ViewRow row
func (c *Computable) viewRow(name string, process *process.Process, res map[string]interface{}, getField func(string) (*field.ColumnDSL, string, string, error)) error {

	if c.Computes == nil {
		return nil
	}

	messages := []string{}
	row := maps.MapOf(res).Dot()
	data := maps.StrAny{"row": row}.Dot()

	//  page
	if row.Has("data") && row.Has("total") &&
		row.Has("pagesize") && row.Has("pagecnt") &&
		row.Has("prev") && row.Has("next") {
		switch res["data"].(type) {

		case []maps.MapStrAny:
			return c.viewRows(name, process, res["data"].([]maps.MapStrAny), getField)

		case []interface{}:
			return c.viewRows(name, process, res["data"].([]interface{}), getField)
		}
	}

	for key, computes := range c.Computes.View {
		unit := computes[0]
		field, path, id, err := getField(unit.Name)
		if err != nil {
			messages = append(messages, err.Error())
			continue
		}

		data.Set("value", res[key])
		data.Set("id", id)
		data.Set("path", fmt.Sprintf("%s.%s", path, unit.Name))
		data.Merge(any.MapOf(field.View.Map()).MapStrAny.Dot())
		new, err := field.View.Compute.Value(data, process.Sid, process.Global)
		if err != nil {
			res[key] = nil
			messages = append(messages, fmt.Sprintf("%s.%s bind: %s, value: %v error: %s", path, unit.Name, key, res[key], err.Error()))
			continue
		}
		res[key] = new
	}

	if len(messages) > 0 {
		return fmt.Errorf("\n%s", strings.Join(messages, ";\n"))
	}

	return nil
}

// ComputeFilter filter
func (c *Computable) ComputeFilter(name string, process *process.Process, args []interface{}, getFilter func(string) (*field.FilterDSL, string, string, error)) error {
	return nil
}
