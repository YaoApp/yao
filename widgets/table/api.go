package table

import (
	"fmt"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/action"
)

// Guard table widget guard
func Guard(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		abort(c, 400, "the table widget id does not found")
		return
	}

	tab, has := Tables[id]
	if !has {
		abort(c, 404, fmt.Sprintf("the table widget %s does not exist", id))
		return
	}

	act, err := tab.getAction(c.FullPath())
	if err != nil {
		abort(c, 404, err.Error())
		return
	}

	err = act.UseGuard(c, id)
	if err != nil {
		abort(c, 400, err.Error())
		return
	}

}

func abort(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"code": code, "message": message})
	c.Abort()
}

func (table *DSL) getAction(path string) (*action.Process, error) {

	switch path {
	case "/api/__yao/table/:id/setting":
		return table.Action.Setting, nil
	case "/api/__yao/table/:id/component/:xpath/:method":
		return table.Action.Component, nil
	case "/api/__yao/table/:id/upload/:xpath/:method":
		return table.Action.Upload, nil
	case "/api/__yao/table/:id/download/:field":
		return table.Action.Download, nil
	case "/api/__yao/table/:id/search":
		return table.Action.Search, nil
	case "/api/__yao/table/:id/get":
		return table.Action.Get, nil
	case "/api/__yao/table/:id/find/:primary":
		return table.Action.Find, nil
	case "/api/__yao/table/:id/save":
		return table.Action.Save, nil
	case "/api/__yao/table/:id/create":
		return table.Action.Create, nil
	case "/api/__yao/table/:id/insert":
		return table.Action.Insert, nil
	case "/api/__yao/table/:id/update/:primary":
		return table.Action.Update, nil
	case "/api/__yao/table/:id/update/in":
		return table.Action.UpdateIn, nil
	case "/api/__yao/table/:id/update/where":
		return table.Action.UpdateWhere, nil
	case "/api/__yao/table/:id/delete/:primary":
		return table.Action.Delete, nil
	case "/api/__yao/table/:id/delete/in":
		return table.Action.DeleteIn, nil
	case "/api/__yao/table/:id/delete/where":
		return table.Action.DeleteWhere, nil
	}

	return nil, fmt.Errorf("the table widget %s %s action does not exist", table.ID, path)
}

// export API
func exportAPI() error {

	http := gou.HTTP{
		Name:        "Widget Table API",
		Description: "Widget Table API",
		Version:     share.VERSION,
		Guard:       "widget-table",
		Group:       "__yao/table",
		Paths:       []gou.Path{},
	}

	//   GET  /api/__yao/table/:id/setting  					-> Default process: yao.table.Xgen
	path := gou.Path{
		Label:       "Setting",
		Description: "Setting",
		Path:        "/:id/setting",
		Method:      "GET",
		Process:     "yao.table.Setting",
		In:          []string{"$param.id"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/table/:id/search  						-> Default process: yao.table.Search $param.id :query $query.page  $query.pagesize
	path = gou.Path{
		Label:       "Search",
		Description: "Search",
		Path:        "/:id/search",
		Method:      "GET",
		Process:     "yao.table.Search",
		In:          []string{"$param.id", ":query-param", "$query.page", "$query.pagesize"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/table/:id/get  						-> Default process: yao.table.Get $param.id :query
	path = gou.Path{
		Label:       "Get",
		Description: "Get",
		Path:        "/:id/get",
		Method:      "GET",
		Process:     "yao.table.Get",
		In:          []string{"$param.id", ":query-param"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/table/:id/find/:primary  				-> Default process: yao.table.Find $param.id $param.primary :query
	path = gou.Path{
		Label:       "Find",
		Description: "Find",
		Path:        "/:id/find/:primary",
		Method:      "GET",
		Process:     "yao.table.Find",
		In:          []string{"$param.id", "$param.primary", ":query-param"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/table/:id/component/:xpath/:method  	-> Default process: yao.table.Component $param.id $param.xpath $param.method :query
	path = gou.Path{
		Label:       "Component",
		Description: "Component",
		Path:        "/:id/component/:xpath/:method",
		Method:      "GET",
		Process:     "yao.table.Component",
		In:          []string{"$param.id", "$param.xpath", "$param.method", ":query"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   POST  /api/__yao/table/:id/upload/:xpath/:method  	-> Default process: yao.table.Upload $param.id $param.xpath $param.method $file.file
	path = gou.Path{
		Label:       "Upload",
		Description: "Upload",
		Path:        "/:id/upload/:xpath/:method",
		Method:      "POST",
		Process:     "yao.table.Upload",
		In:          []string{"$param.id", "$param.xpath", "$param.method", "$file.file"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/table/:id/download/:field  	-> Default process: yao.table.Download $param.id $param.xpath $param.field $query.name $query.token
	path = gou.Path{
		Label:       "Download",
		Description: "Download",
		Path:        "/:id/download/:field",
		Method:      "GET",
		Process:     "yao.table.Download",
		In:          []string{"$param.id", "$param.field", "$query.name", "$query.token"},
		Out: gou.Out{
			Status:  200,
			Body:    "{{content}}",
			Headers: map[string]string{"Content-Type": "{{type}}"},
		},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/save  						-> Default process: yao.table.Save $param.id :payload
	path = gou.Path{
		Label:       "Save",
		Description: "Save",
		Path:        "/:id/save",
		Method:      "POST",
		Process:     "yao.table.Save",
		In:          []string{"$param.id", ":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/create  						-> Default process: yao.table.Create $param.id :payload
	path = gou.Path{
		Label:       "Create",
		Description: "Create",
		Path:        "/:id/create",
		Method:      "POST",
		Process:     "yao.table.Create",
		In:          []string{"$param.id", ":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/insert  						-> Default process: yao.table.Insert :payload
	path = gou.Path{
		Label:       "Insert",
		Description: "Insert",
		Path:        "/:id/insert",
		Method:      "POST",
		Process:     "yao.table.Insert",
		In:          []string{"$param.id", "$payload.columns", "$payload.values"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/update/:primary  			-> Default process: yao.table.Update $param.id $param.primary :payload
	path = gou.Path{
		Label:       "Update",
		Description: "Update",
		Path:        "/:id/update/:primary",
		Method:      "POST",
		Process:     "yao.table.Update",
		In:          []string{"$param.id", "$param.primary", ":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/update/where  				-> Default process: yao.table.UpdateWhere $param.id :query :payload
	path = gou.Path{
		Label:       "Update Where",
		Description: "Update Where",
		Path:        "/:id/update/where",
		Method:      "POST",
		Process:     "yao.table.UpdateWhere",
		In:          []string{"$param.id", ":query-param", ":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/update/in  					-> Default process: yao.table.UpdateIn $param.id $query.ids :payload
	path = gou.Path{
		Label:       "Update In",
		Description: "Update In",
		Path:        "/:id/update/in",
		Method:      "POST",
		Process:     "yao.table.UpdateIn",
		In:          []string{"$param.id", "$query.ids", ":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/delete/:primary  			-> Default process: yao.table.Delete $param.id $param.primary
	path = gou.Path{
		Label:       "Delete",
		Description: "Delete",
		Path:        "/:id/delete/:primary",
		Method:      "POST",
		Process:     "yao.table.Delete",
		In:          []string{"$param.id", "$param.primary"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/delete/where  				-> Default process: yao.table.DeleteWhere $param.id :query
	path = gou.Path{
		Label:       "Delete Where",
		Description: "Delete Where",
		Path:        "/:id/delete/where",
		Method:      "POST",
		Process:     "yao.table.DeleteWhere",
		In:          []string{"$param.id", ":query-param"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/table/:id/delete/in  					-> Default process: yao.table.DeleteIn $param.id $query.ids
	path = gou.Path{
		Label:       "Delete In",
		Description: "Delete In",
		Path:        "/:id/delete/in",
		Method:      "POST",
		Process:     "yao.table.DeleteIn",
		In:          []string{"$param.id", "$query.ids"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = gou.LoadAPIReturn(string(source), "widgets.table")
	return err
}
