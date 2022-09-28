package form

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/share"
)

// export API
func exportAPI() error {

	http := gou.HTTP{
		Name:        "Widget Form API",
		Description: "Widget Form API",
		Version:     share.VERSION,
		Guard:       "-",
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
