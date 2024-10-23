package form

import (
	"fmt"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/action"
)

// Guard form widget guard
func Guard(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		abort(c, 400, "the form widget id does not found")
		return
	}

	form, has := Forms[id]
	if !has {
		abort(c, 404, fmt.Sprintf("the form widget %s does not exist", id))
		return
	}

	act, err := form.getAction(c.FullPath())
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

func (form *DSL) getAction(path string) (*action.Process, error) {

	switch path {
	case "/api/__yao/form/:id/setting":
		return form.Action.Setting, nil
	case "/api/__yao/form/:id/component/:xpath/:method":
		return form.Action.Component, nil
	case "/api/__yao/form/:id/upload/:xpath/:method":
		return form.Action.Upload, nil
	case "/api/__yao/form/:id/download/:field":
		return form.Action.Download, nil
	case "/api/__yao/form/:id/find/:primary":
		return form.Action.Find, nil
	case "/api/__yao/form/:id/save":
		return form.Action.Save, nil
	case "/api/__yao/form/:id/create":
		return form.Action.Create, nil
	case "/api/__yao/form/:id/insert":
		return form.Action.Update, nil
	case "/api/__yao/form/:id/delete/:primary":
		return form.Action.Delete, nil
	}

	return nil, fmt.Errorf("the form widget %s %s action does not exist", form.ID, path)
}

// export API
func exportAPI() error {

	http := api.HTTP{
		Name:        "Widget Form API",
		Description: "Widget Form API",
		Version:     share.VERSION,
		Guard:       "widget-form",
		Group:       "__yao/form",
		Paths:       []api.Path{},
	}

	//   GET  /api/__yao/form/:id/setting  					-> Default process: yao.form.Xgen
	path := api.Path{
		Label:       "Setting",
		Description: "Setting",
		Path:        "/:id/setting",
		Method:      "GET",
		Process:     "yao.form.Setting",
		In:          []interface{}{"$param.id"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/form/:id/find/:primary  				-> Default process: yao.form.Find $param.id $param.primary :query
	path = api.Path{
		Label:       "Find",
		Description: "Find",
		Path:        "/:id/find/:primary",
		Method:      "GET",
		Process:     "yao.form.Find",
		In:          []interface{}{"$param.id", "$param.primary", ":query-param"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/form/:id/component/:xpath/:method  	-> Default process: yao.form.Component $param.id $param.xpath $param.method :query
	path = api.Path{
		Label:       "Component",
		Description: "Component",
		Path:        "/:id/component/:xpath/:method",
		Method:      "GET",
		Process:     "yao.form.Component",
		In:          []interface{}{"$param.id", "$param.xpath", "$param.method", ":query"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   POST  /api/__yao/form/:id/component/:xpath/:method  	-> Default process: yao.form.Component $param.id $param.xpath $param.method :payload
	path = api.Path{
		Label:       "Component",
		Description: "Component",
		Path:        "/:id/component/:xpath/:method",
		Method:      "POST",
		Process:     "yao.form.Component",
		In:          []interface{}{"$param.id", "$param.xpath", "$param.method", ":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   POST  /api/__yao/table/:id/upload/:xpath/:method  	-> Default process: yao.form.Upload $param.id $param.xpath $param.method $file.file
	path = api.Path{
		Label:       "Upload",
		Description: "Upload",
		Path:        "/:id/upload/:xpath/:method",
		Method:      "POST",
		Process:     "yao.form.Upload",
		In:          []interface{}{"$param.id", "$param.xpath", "$param.method", "$file.file"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/form/:id/download/:field  	-> Default process: yao.form.Download $param.id $param.xpath $param.field $query.name $query.token
	path = api.Path{
		Label:       "Download",
		Description: "Download",
		Path:        "/:id/download/:field",
		Method:      "GET",
		Process:     "yao.form.Download",
		In:          []interface{}{"$param.id", "$param.field", "$query.name", "$query.token", "$query.app"},
		Out: api.Out{
			Status:  200,
			Body:    "{{content}}",
			Headers: map[string]string{"Content-Type": "{{type}}"},
		},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/form/:id/save  						-> Default process: yao.form.Save $param.id :payload
	path = api.Path{
		Label:       "Save",
		Description: "Save",
		Path:        "/:id/save",
		Method:      "POST",
		Process:     "yao.form.Save",
		In:          []interface{}{"$param.id", ":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/form/:id/create  						-> Default process: yao.form.Create $param.id :payload
	path = api.Path{
		Label:       "Create",
		Description: "Create",
		Path:        "/:id/create",
		Method:      "POST",
		Process:     "yao.form.Create",
		In:          []interface{}{"$param.id", ":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/form/:id/update/:primary  			-> Default process: yao.form.Update $param.id $param.primary :payload
	path = api.Path{
		Label:       "Update",
		Description: "Update",
		Path:        "/:id/update/:primary",
		Method:      "POST",
		Process:     "yao.form.Update",
		In:          []interface{}{"$param.id", "$param.primary", ":payload"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/form/:id/delete/:primary  			-> Default process: yao.form.Delete $param.id $param.primary
	path = api.Path{
		Label:       "Delete",
		Description: "Delete",
		Path:        "/:id/delete/:primary",
		Method:      "POST",
		Process:     "yao.form.Delete",
		In:          []interface{}{"$param.id", "$param.primary"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = api.LoadSource("<widget.form>.yao", source, "widgets.form")
	return err
}
