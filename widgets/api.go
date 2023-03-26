package widgets

import (
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/yao/widgets/chart"
	"github.com/yaoapp/yao/widgets/form"
	"github.com/yaoapp/yao/widgets/list"
	"github.com/yaoapp/yao/widgets/table"
)

// Apis return loaded apis
func Apis() []Item {
	apis := map[string]interface{}{}
	userApis(apis)
	tableApis(apis)
	formApis(apis)
	listApis(apis)
	chartApis(apis)

	grouping := Grouping(apis)
	items := Array(grouping, []Item{})
	Sort(items, []string{"apis", "tables", "forms", "lists", "charts"})
	return items
}

func userApis(apis map[string]interface{}) {

	// Name        string `json:"name"`
	// Version     string `json:"version"`
	// Description string `json:"description,omitempty"`
	// Group       string `json:"group,omitempty"`
	// Guard       string `json:"guard,omitempty"`
	// Paths       []Path `json:"paths,omitempty"`
	for group, api := range api.APIs {
		if strings.HasPrefix(group, "widgets") {
			continue
		}

		// Label       string   `json:"label,omitempty"`
		// Description string   `json:"description,omitempty"`
		// Path        string   `json:"path"`
		// Method      string   `json:"method"`
		// Process     string   `json:"process"`
		// Guard       string   `json:"guard,omitempty"`
		// In          []string `json:"in,omitempty"`
		// Out         Out      `json:"out,omitempty"`
		paths := []map[string]interface{}{}
		for _, path := range api.HTTP.Paths {
			guard := path.Guard
			if guard == "" {
				guard = api.HTTP.Guard
			}
			fullpath := fmt.Sprintf("/apis/%s%s", api.HTTP.Group, path.Path)
			paths = append(paths, map[string]interface{}{
				"name":        path.Label,
				"description": path.Description,
				"guard":       guard,
				"method":      path.Method,
				"path":        path.Path,
				"router":      fullpath,
				"fullpath":    fullpath,
				"in":          path.In,
				"out":         path.Out,
				"process":     path.Process,
				"params":      map[string]interface{}{},
			})
		}

		dsl := fmt.Sprintf("apis%s%s.http.json", string(os.PathSeparator), strings.ReplaceAll(group, ".", string(os.PathSeparator)))
		apis[dsl] = map[string]interface{}{
			"DSL":         dsl,
			"name":        api.HTTP.Name,
			"version":     api.HTTP.Version,
			"group":       fmt.Sprintf("/%s", api.HTTP.Group),
			"guard":       api.HTTP.Guard,
			"description": api.HTTP.Description,
			"paths":       paths,
		}
	}
}

func tableApis(apis map[string]interface{}) {

	api, has := api.APIs["widgets.table"]
	if !has {
		return
	}

	for id, widget := range table.Tables {
		dsl := fmt.Sprintf("tables%s%s.tab.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		groupGuard := "bearer-jwt"
		pathGuards := []map[string]string{
			{"name": "/:id/search", "guard": widget.Action.Search.Guard},
			{"name": "/:id/get", "guard": widget.Action.Get.Guard},
			{"name": "/:id/find/:primary", "guard": widget.Action.Find.Guard},
			{"name": "/:id/save", "guard": widget.Action.Save.Guard},
			{"name": "/:id/create", "guard": widget.Action.Create.Guard},
			{"name": "/:id/insert", "guard": widget.Action.Insert.Guard},
			{"name": "/:id/update/:primary", "guard": widget.Action.Update.Guard},
			{"name": "/:id/update/in", "guard": widget.Action.UpdateIn.Guard},
			{"name": "/:id/update/where", "guard": widget.Action.UpdateWhere.Guard},
			{"name": "/:id/delete/:primary", "guard": widget.Action.Delete.Guard},
			{"name": "/:id/delete/in", "guard": widget.Action.DeleteIn.Guard},
			{"name": "/:id/delete/where", "guard": widget.Action.DeleteWhere.Guard},
			{"name": "/:id/upload/:xpath/:method", "guard": widget.Action.Upload.Guard},
			{"name": "/:id/download/:field", "guard": widget.Action.Download.Guard},
		}
		widgetApis(apis, api, id, dsl, groupGuard, pathGuards)
	}
}

func formApis(apis map[string]interface{}) {

	api, has := api.APIs["widgets.form"]
	if !has {
		return
	}

	for id, widget := range form.Forms {
		dsl := fmt.Sprintf("forms%s%s.form.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		groupGuard := "bearer-jwt"
		pathGuards := []map[string]string{
			{"name": "/:id/find/:primary", "guard": widget.Action.Find.Guard},
			{"name": "/:id/save", "guard": widget.Action.Save.Guard},
			{"name": "/:id/create", "guard": widget.Action.Create.Guard},
			{"name": "/:id/update/:primary", "guard": widget.Action.Update.Guard},
			{"name": "/:id/delete/:primary", "guard": widget.Action.Delete.Guard},
			{"name": "/:id/upload/:xpath/:method", "guard": widget.Action.Upload.Guard},
			{"name": "/:id/download/:field", "guard": widget.Action.Download.Guard},
		}
		widgetApis(apis, api, id, dsl, groupGuard, pathGuards)
	}
}

func listApis(apis map[string]interface{}) {

	api, has := api.APIs["widgets.list"]
	if !has {
		return
	}

	for id, widget := range list.Lists {
		dsl := fmt.Sprintf("lists%s%s.list.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		groupGuard := "bearer-jwt"
		pathGuards := []map[string]string{
			{"name": "/:id/get", "guard": widget.Action.Get.Guard},
			{"name": "/:id/save", "guard": widget.Action.Save.Guard},
			{"name": "/:id/upload/:xpath/:method", "guard": widget.Action.Upload.Guard},
			{"name": "/:id/download/:field", "guard": widget.Action.Download.Guard},
		}
		widgetApis(apis, api, id, dsl, groupGuard, pathGuards)
	}
}

func chartApis(apis map[string]interface{}) {

	api, has := api.APIs["widgets.chart"]
	if !has {
		return
	}

	for id, widget := range chart.Charts {
		dsl := fmt.Sprintf("charts%s%s.chart.json", string(os.PathSeparator), strings.ReplaceAll(id, ".", string(os.PathSeparator)))
		groupGuard := "bearer-jwt"
		pathGuards := []map[string]string{
			{"name": "/:id/data", "guard": widget.Action.Data.Guard},
		}
		widgetApis(apis, api, id, dsl, groupGuard, pathGuards)
	}
}

func widgetApis(apis map[string]interface{}, apiInst *api.API, widgetID string, dsl string, groupGuard string, pathGuards []map[string]string) {

	pathMapping := map[string]api.Path{}
	for _, path := range apiInst.HTTP.Paths {
		pathMapping[path.Path] = path
	}

	// Label       string   `json:"label,omitempty"`
	// Description string   `json:"description,omitempty"`
	// Path        string   `json:"path"`
	// Method      string   `json:"method"`
	// Process     string   `json:"process"`
	// Guard       string   `json:"guard,omitempty"`
	// In          []string `json:"in,omitempty"`
	// Out         Out      `json:"out,omitempty"`
	paths := []map[string]interface{}{}
	for _, pathGuard := range pathGuards {

		name := pathGuard["name"]
		guard := pathGuard["guard"]

		path, has := pathMapping[name]
		if !has {
			continue
		}

		if guard == "" {
			guard = groupGuard
		}

		fullpath := fmt.Sprintf("/apis/%s%s", apiInst.HTTP.Group, path.Path)
		paths = append(paths, map[string]interface{}{
			"name":        path.Label,
			"description": path.Description,
			"guard":       guard,
			"method":      path.Method,
			"path":        path.Path,
			"fullpath":    fullpath,
			"router":      strings.ReplaceAll(fullpath, ":id", widgetID),
			"in":          path.In,
			"out":         path.Out,
			"process":     path.Process,
			"params":      map[string]interface{}{"id": widgetID},
		})
	}

	apis[dsl] = map[string]interface{}{
		"DSL":         dsl,
		"name":        apiInst.HTTP.Name,
		"version":     apiInst.HTTP.Version,
		"group":       fmt.Sprintf("/%s", apiInst.HTTP.Group),
		"guard":       groupGuard,
		"description": apiInst.HTTP.Description,
		"paths":       paths,
	}
}
