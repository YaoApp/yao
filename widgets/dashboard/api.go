package dashboard

import (
	"fmt"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/action"
)

// Guard form widget dashboard
func Guard(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		abort(c, 400, "the dashboard widget id does not found")
		return
	}

	dashboard, has := Dashboards[id]
	if !has {
		abort(c, 404, fmt.Sprintf("the dashboard widget %s does not exist", id))
		return
	}

	act, err := dashboard.getAction(c.FullPath())
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

func (dashboard *DSL) getAction(path string) (*action.Process, error) {

	switch path {
	case "/api/__yao/dashboard/:id/setting":
		return dashboard.Action.Setting, nil
	case "/api/__yao/dashboard/:id/component/:xpath/:method":
		return dashboard.Action.Component, nil
	case "/api/__yao/dashboard/:id/data":
		return dashboard.Action.Data, nil
	}

	return nil, fmt.Errorf("the form widget %s %s action does not exist", dashboard.ID, path)
}

// export API
func exportAPI() error {

	http := gou.HTTP{
		Name:        "Widget Dashboard API",
		Description: "Widget Dashboard API",
		Version:     share.VERSION,
		Guard:       "widget-dashboard",
		Group:       "__yao/dashboard",
		Paths:       []gou.Path{},
	}

	//   GET  /api/__yao/dashboard/:id/setting  					-> Default process: yao.dashboard.Xgen
	path := gou.Path{
		Label:       "Setting",
		Description: "Setting",
		Path:        "/:id/setting",
		Method:      "GET",
		Process:     "yao.dashboard.Setting",
		In:          []string{"$param.id"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/dashboard/:id/data 					-> Default process: yao.dashboard.Data $param.id :query
	path = gou.Path{
		Label:       "Data",
		Description: "Data",
		Path:        "/:id/data",
		Method:      "GET",
		Process:     "yao.dashboard.Data",
		In:          []string{"$param.id", ":query"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/dashboard/:id/component/:xpath/:method  	-> Default process: yao.dashboard.Component $param.id $param.xpath $param.method :query
	path = gou.Path{
		Label:       "Component",
		Description: "Component",
		Path:        "/:id/component/:xpath/:method",
		Method:      "GET",
		Process:     "yao.dashboard.Component",
		In:          []string{"$param.id", "$param.xpath", "$param.method", ":query"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = gou.LoadAPIReturn(string(source), "widgets.dashboard")
	return err
}
