package form

import (
	"fmt"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
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

	http := gou.HTTP{
		Name:        "Widget Form API",
		Description: "Widget Form API",
		Version:     share.VERSION,
		Guard:       "widget-form",
		Group:       "__yao/form",
		Paths:       []gou.Path{},
	}

	//   GET  /api/__yao/form/:id/setting  					-> Default process: yao.form.Xgen
	path := gou.Path{
		Label:       "Setting",
		Description: "Setting",
		Path:        "/:id/setting",
		Method:      "GET",
		Process:     "yao.form.Setting",
		In:          []string{"$param.id"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/form/:id/find/:primary  				-> Default process: yao.form.Find $param.id $param.primary :query
	path = gou.Path{
		Label:       "Find",
		Description: "Find",
		Path:        "/:id/find/:primary",
		Method:      "GET",
		Process:     "yao.form.Find",
		In:          []string{"$param.id", "$param.primary", ":query-param"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/form/:id/component/:xpath/:method  	-> Default process: yao.form.Component $param.id $param.xpath $param.method :query
	path = gou.Path{
		Label:       "Component",
		Description: "Component",
		Path:        "/:id/component/:xpath/:method",
		Method:      "GET",
		Process:     "yao.form.Component",
		In:          []string{"$param.id", "$param.xpath", "$param.method", ":query"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/form/:id/save  						-> Default process: yao.form.Save $param.id :payload
	path = gou.Path{
		Label:       "Save",
		Description: "Save",
		Path:        "/:id/save",
		Method:      "POST",
		Process:     "yao.form.Save",
		In:          []string{"$param.id", ":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/form/:id/create  						-> Default process: yao.form.Create $param.id :payload
	path = gou.Path{
		Label:       "Create",
		Description: "Create",
		Path:        "/:id/create",
		Method:      "POST",
		Process:     "yao.form.Create",
		In:          []string{"$param.id", ":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/form/:id/update/:primary  			-> Default process: yao.form.Update $param.id $param.primary :payload
	path = gou.Path{
		Label:       "Update",
		Description: "Update",
		Path:        "/:id/update/:primary",
		Method:      "POST",
		Process:     "yao.form.Update",
		In:          []string{"$param.id", "$param.primary", ":payload"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//  POST  /api/__yao/form/:id/delete/:primary  			-> Default process: yao.form.Delete $param.id $param.primary
	path = gou.Path{
		Label:       "Delete",
		Description: "Delete",
		Path:        "/:id/delete/:primary",
		Method:      "POST",
		Process:     "yao.form.Delete",
		In:          []string{"$param.id", "$param.primary"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = gou.LoadAPIReturn(string(source), "widgets.form")
	return err
}
