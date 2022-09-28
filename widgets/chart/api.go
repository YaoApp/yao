package chart

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
		Group:       "__yao/chart",
		Paths:       []gou.Path{},
	}

	//   GET  /api/__yao/chart/:id/setting  					-> Default process: yao.chart.Xgen
	path := gou.Path{
		Label:       "Setting",
		Description: "Setting",
		Path:        "/:id/setting",
		Method:      "GET",
		Process:     "yao.chart.Setting",
		In:          []string{"$param.id"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/chart/:id/data 					-> Default process: yao.chart.Data $param.id :query
	path = gou.Path{
		Label:       "Data",
		Description: "Data",
		Path:        "/:id/data",
		Method:      "GET",
		Process:     "yao.chart.Data",
		In:          []string{"$param.id", ":query"},
		Out:         gou.Out{Status: 200, Type: "application/json"},
	}
	http.Paths = append(http.Paths, path)

	//   GET  /api/__yao/chart/:id/component/:xpath/:method  	-> Default process: yao.chart.Component $param.id $param.xpath $param.method :query
	path = gou.Path{
		Label:       "Component",
		Description: "Component",
		Path:        "/:id/component/:xpath/:method",
		Method:      "GET",
		Process:     "yao.chart.Component",
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
	_, err = gou.LoadAPIReturn(string(source), "widgets.chart")
	return err
}
