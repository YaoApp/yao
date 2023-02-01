package widgets

import (
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/gou/model"
)

// Models return loaded models
func Models() []Item {

	models := map[string]interface{}{}
	for id, widget := range model.Models {

		if strings.HasPrefix(id, "xiang.") {
			continue
		}

		name := fmt.Sprintf("%s.mod.json", strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		dsl := fmt.Sprintf("models%s%s.mod.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		models[name] = map[string]interface{}{
			"DSL":       dsl,
			"ID":        id,
			"connector": widget.MetaData.Connector,
			"table":     widget.MetaData.Table,
			"columns":   widget.MetaData.Columns,
			"indexes":   widget.MetaData.Indexes,
			"values":    widget.MetaData.Values,
			"option":    widget.MetaData.Option,
			"relations": widget.MetaData.Relations,
		}
	}

	grouping := Grouping(models)
	items := Array(grouping, []Item{})
	Sort(items, []string{})
	return items
}
