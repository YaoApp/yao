package table

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/maps"
)

var computeViews = map[string]bool{
	"yao.table.find":   true,
	"yao.table.get":    true,
	"yao.table.search": true,
}

func (dsl *DSL) computeMapping() error {
	if dsl.computes == nil {
		dsl.computes = &computeMaps{
			filter: map[string][]compute{},
			edit:   map[string][]compute{},
			view:   map[string][]compute{},
		}
	}

	if dsl.Fields == nil {
		return nil
	}

	// Filter
	if dsl.Fields.Filter != nil && dsl.Layout.Filter != nil && dsl.Layout.Filter.Columns != nil {
		for _, inst := range dsl.Layout.Filter.Columns {
			if filter, has := dsl.Fields.Filter[inst.Name]; has {
				if filter.Edit != nil && filter.Edit.Compute != nil {
					bind := filter.FilterBind()
					if _, has := dsl.computes.filter[bind]; !has {
						dsl.computes.filter[bind] = []compute{}
					}
					dsl.computes.filter[bind] = append(dsl.computes.filter[bind], compute{name: inst.Name, kind: TypeFilter})
				}
			}
		}
	}

	if dsl.Fields.Table != nil && dsl.Layout.Table != nil && dsl.Layout.Table.Columns != nil {
		for _, inst := range dsl.Layout.Table.Columns {
			if field, has := dsl.Fields.Table[inst.Name]; has {
				// View
				if field.View != nil && field.View.Compute != nil {
					bind := field.ViewBind()
					if _, has := dsl.computes.view[bind]; !has {
						dsl.computes.view[bind] = []compute{}
					}
					dsl.computes.view[bind] = append(dsl.computes.view[bind], compute{name: inst.Name, kind: TypeView})
				}

				// Edit
				if field.Edit != nil && field.Edit.Compute != nil {
					bind := field.EditBind()
					if _, has := dsl.computes.edit[bind]; !has {
						dsl.computes.edit[bind] = []compute{}
					}
					dsl.computes.edit[bind] = append(dsl.computes.edit[bind], compute{name: inst.Name, kind: TypeEdit})
				}
			}
		}
	}

	return nil
}

func (dsl *DSL) computeView(name string, process *gou.Process, res interface{}) error {

	if _, has := computeViews[strings.ToLower(name)]; !has {
		return nil
	}

	switch res.(type) {
	case []maps.MapStrAny, []interface{}:
		return dsl.computeViewRows(name, process, res)

	case map[string]interface{}:
		return dsl.computeViewRow(name, process, res.(map[string]interface{}))

	case maps.MapStrAny:
		return dsl.computeViewRow(name, process, res.(maps.MapStrAny))
	}

	return fmt.Errorf("res should be a map or array, but got a %s", reflect.ValueOf(res).Kind().String())
}

func (dsl *DSL) computeViewRows(name string, process *gou.Process, res interface{}) error {
	switch res.(type) {

	case []interface{}:
		messages := []string{}
		for i := range res.([]interface{}) {
			err := dsl.computeView(name, process, res.([]interface{})[i])
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
			err := dsl.computeView(name, process, res.([]maps.MapStrAny)[i])
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

func (dsl *DSL) computeViewRow(name string, process *gou.Process, res map[string]interface{}) error {

	messages := []string{}
	row := maps.MapOf(res).Dot()
	data := maps.StrAny{"row": row}.Dot()

	//  page
	if row.Has("data") && row.Has("total") &&
		row.Has("pagesize") && row.Has("pagecnt") &&
		row.Has("prev") && row.Has("next") {
		switch res["data"].(type) {

		case []maps.MapStrAny:
			return dsl.computeViewRows(name, process, res["data"].([]maps.MapStrAny))

		case []interface{}:
			return dsl.computeViewRows(name, process, res["data"].([]interface{}))
		}
	}

	for key, computes := range dsl.computes.view {
		c := computes[0]
		field := dsl.Fields.Table[c.name]
		data.Set("value", res[key])
		data.Set("type", field.View.Type)
		data.Set("props", field.View.Props)
		new, err := field.View.Compute.Value(data, process.Sid, process.Global)
		if err != nil {
			res[key] = nil
			messages = append(messages, fmt.Sprintf("fields.table.%s bind: %s, value: %v error: %s", c.name, key, res[key], err.Error()))
			continue
		}
		res[key] = new
	}

	if len(messages) > 0 {
		return fmt.Errorf("\n%s", strings.Join(messages, ";\n"))
	}

	return nil
}

func (dsl *DSL) computeEdit(name string, process *gou.Process, args []interface{}) error {
	name = strings.ToLower(name)
	switch name {
	case "yao.table.save", "yao.table.create":
		if len(args) == 0 {
			return nil
		}
		switch args[0].(type) {
		case maps.MapStr:
			return dsl.computeEditRow(name, process, args[0].(maps.MapStr))

		case map[string]interface{}:
			return dsl.computeEditRow(name, process, args[0].(map[string]interface{}))
		}
		return nil

	case "yao.table.update", "yao.table.updatewhere", "yao.table.updatein":
		if len(args) < 2 {
			return nil
		}

		switch args[1].(type) {
		case maps.MapStr:
			return dsl.computeEditRow(name, process, args[1].(maps.MapStr))

		case map[string]interface{}:
			return dsl.computeEditRow(name, process, args[1].(map[string]interface{}))
		}
		return nil

	case "yao.table.insert":
		if len(args) < 2 {
			return nil
		}

		if _, ok := args[0].([]string); !ok {
			return fmt.Errorf("args[0] is not a []string %s", reflect.ValueOf(args[0]).Type().Name())
		}

		if _, ok := args[1].([][]interface{}); !ok {
			return fmt.Errorf("args[1] is not a [][] %s", reflect.ValueOf(args[1]).Type().Name())
		}

		return dsl.computeEditRows(name, process, args[0].([]string), args[1].([][]interface{}))
	}

	return nil
}

func (dsl *DSL) computeEditRow(name string, process *gou.Process, res map[string]interface{}) error {

	messages := []string{}
	row := maps.MapOf(res).Dot()
	data := maps.StrAny{"row": row}.Dot()
	for key := range res {
		if computes, has := dsl.computes.edit[key]; has {
			c := computes[0]
			field := dsl.Fields.Table[c.name]
			data.Set("value", res[key])
			data.Set("type", field.Edit.Type)
			data.Set("props", field.Edit.Props)
			new, err := field.Edit.Compute.Value(data, process.Sid, process.Global)
			if err != nil {
				messages = append(messages, fmt.Sprintf("fields.table.%s bind: %s, value: %v error: %s", c.name, key, res[key], err.Error()))
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

func (dsl *DSL) computeEditRows(name string, process *gou.Process, columns []string, res [][]interface{}) error {

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

		err := dsl.computeEditRow(name, process, row)
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

func (dsl *DSL) computeFilter(name string, process *gou.Process, args []interface{}) error {
	return nil
}
