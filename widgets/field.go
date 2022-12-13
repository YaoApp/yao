package widgets

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yaoapp/yao/widgets/chart"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/form"
	"github.com/yaoapp/yao/widgets/list"
	"github.com/yaoapp/yao/widgets/table"
)

// Fields return loaded widgets fields
func Fields() []Item {

	fields := map[string]interface{}{}
	tableFields(fields)
	formFields(fields)
	listFields(fields)
	chartFields(fields)

	grouping := Grouping(fields)
	items := Array(grouping, []Item{})
	Sort(items, []string{"tables", "forms", "lists", "charts"})
	return items
}

// Filters return loaded widgets filters
func Filters() []Item {
	filters := map[string]interface{}{}
	tableFilters(filters)
	chartFilters(filters)

	grouping := Grouping(filters)
	items := Array(grouping, []Item{})
	Sort(items, []string{"tables", "forms", "lists", "charts"})
	return items
}

func tableFields(fields map[string]interface{}) {
	for id, widget := range table.Tables {
		dsl := fmt.Sprintf("tables%s%s.tab.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetFields(fields, widget.Fields.Table, id, dsl, widget.Name)
	}
}

func formFields(fields map[string]interface{}) {
	for id, widget := range form.Forms {
		dsl := fmt.Sprintf("forms%s%s.form.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetFields(fields, widget.Fields.Form, id, dsl, widget.Name)
	}
}

func chartFields(fields map[string]interface{}) {
	for id, widget := range chart.Charts {
		dsl := fmt.Sprintf("charts%s%s.chart.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetFields(fields, widget.Fields.Chart, id, dsl, widget.Name)
	}
}

func listFields(fields map[string]interface{}) {
	for id, widget := range list.Lists {
		dsl := fmt.Sprintf("lists%s%s.list.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetFields(fields, widget.Fields.List, id, dsl, widget.Name)
	}
}

func tableFilters(filters map[string]interface{}) {
	for id, widget := range table.Tables {
		dsl := fmt.Sprintf("tables%s%s.tab.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetFilters(filters, widget.Fields.Filter, id, dsl, widget.Name)
	}
}

func chartFilters(filters map[string]interface{}) {
	for id, widget := range chart.Charts {
		dsl := fmt.Sprintf("charts%s%s.chart.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetFilters(filters, widget.Fields.Filter, id, dsl, widget.Name)
	}
}

func widgetFields(items map[string]interface{}, fields map[string]field.ColumnDSL, widgetID string, dsl string, name string) map[string]interface{} {

	fieldlist := []map[string]string{}
	if fields != nil {
		names := []string{}
		mapping := map[string]string{}
		for name, field := range fields {
			if field.ID != "" {
				mapping[name] = field.ID
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			fieldlist = append(fieldlist, map[string]string{
				"name": name,
				"id":   mapping[name],
			})
		}
	}

	items[dsl] = map[string]interface{}{
		"items": fieldlist,
		"DSL":   dsl,
		"ID":    widgetID,
		"name":  name,
	}

	return nil
}

func widgetFilters(items map[string]interface{}, fields map[string]field.FilterDSL, widgetID string, dsl string, name string) map[string]interface{} {

	fieldlist := []map[string]string{}
	if fields != nil {
		names := []string{}
		mapping := map[string]string{}
		for name, field := range fields {
			if field.ID != "" {
				mapping[name] = field.ID
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			fieldlist = append(fieldlist, map[string]string{
				"name": name,
				"id":   mapping[name],
			})
		}
	}

	items[dsl] = map[string]interface{}{
		"items": fieldlist,
		"DSL":   dsl,
		"ID":    widgetID,
		"name":  name,
	}

	return nil
}
