package widgets

import (
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/yao/widgets/chart"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/form"
	"github.com/yaoapp/yao/widgets/table"
)

// WidgetAction the widget actionlist
type WidgetAction interface {
	Actions() []component.ActionsExport
}

// Actions return loaded widgets actions
func Actions() []Item {

	actions := map[string]interface{}{}
	tableActions(actions)
	formActions(actions)
	chartActions(actions)

	grouping := Grouping(actions)
	items := Array(grouping, []Item{})
	Sort(items, []string{"tables", "forms", "lists", "charts"})
	return items
}

func tableActions(actions map[string]interface{}) {
	for id, widget := range table.Tables {
		dsl := fmt.Sprintf("tables%s%s.tab.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetActions(actions, widget, id, dsl, widget.Name)
	}
}

func formActions(actions map[string]interface{}) {
	for id, widget := range form.Forms {
		dsl := fmt.Sprintf("forms%s%s.form.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetActions(actions, widget, id, dsl, widget.Name)
	}
}

func chartActions(actions map[string]interface{}) {
	for id, widget := range chart.Charts {
		dsl := fmt.Sprintf("charts%s%s.chart.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		widgetActions(actions, widget, id, dsl, widget.Name)
	}
}

func widgetActions(actions map[string]interface{}, widget WidgetAction, widgetID string, dsl string, name string) map[string]interface{} {
	items := widget.Actions()
	if len(items) > 0 {
		actions[dsl] = map[string]interface{}{
			"items": items,
			"DSL":   dsl,
			"ID":    widgetID,
			"name":  name,
		}
	}
	return nil
}
