package chart

import (
	"fmt"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/action"
)

// Guard form widget chart
func Guard(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		abort(c, 400, "the chart widget id does not found")
		return
	}

	chart, has := Charts[id]
	if !has {
		abort(c, 404, fmt.Sprintf("the chart widget %s does not exist", id))
		return
	}

	act, err := chart.getAction(c.FullPath())
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

func (chart *DSL) getAction(path string) (*action.Process, error) {

	switch path {
	case "/api/__yao/chart/:id/setting":
		return chart.Action.Setting, nil
	case "/api/__yao/chart/:id/component/:xpath/:method":
		return chart.Action.Component, nil
	case "/api/__yao/chart/:id/data":
		return chart.Action.Data, nil
	}

	return nil, fmt.Errorf("the form widget %s %s action does not exist", chart.ID, path)
}

// export API
func exportAPI() error {

	http := api.HTTP{
		Name:        "Widget Chart API",
		Description: "Widget Chart API",
		Version:     share.VERSION,
		Guard:       "widget-chart",
		Group:       "__yao/chart",
		Paths:       []api.Path{},
	}

	//   GET  /api/__yao/chart/:id/setting  					-> Default process: yao.chart.Xgen
	path := api.Path{
		Label:       "Setting",
		Description: "Setting",
		Path:        "/:id/setting",
		Method:      "GET",
		Process:     "yao.chart.Setting",
		In:          []interface{}{"$param.id"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/chart/:id/data 					-> Default process: yao.chart.Data $param.id :query
	path = api.Path{
		Label:       "Data",
		Description: "Data",
		Path:        "/:id/data",
		Method:      "GET",
		Process:     "yao.chart.Data",
		In:          []interface{}{"$param.id", ":query"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/chart/:id/component/:xpath/:method  	-> Default process: yao.chart.Component $param.id $param.xpath $param.method :query
	path = api.Path{
		Label:       "Component",
		Description: "Component",
		Path:        "/:id/component/:xpath/:method",
		Method:      "GET",
		Process:     "yao.chart.Component",
		In:          []interface{}{"$param.id", "$param.xpath", "$param.method", ":query"},
		Out:         api.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	// api source
	source, err := jsoniter.Marshal(http)
	if err != nil {
		return err
	}

	// load apis
	_, err = api.LoadSource("<widget.chart>.yao", source, "widgets.chart")
	return err
}
