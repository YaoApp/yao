package chart

import (
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xiang/config"
	_ "github.com/yaoapp/xiang/helper"
	"github.com/yaoapp/xiang/model"
	"github.com/yaoapp/xiang/query"
	"github.com/yaoapp/xiang/share"
)

func init() {
	share.DBConnect(config.Conf.Database)
	model.Load(config.Conf)
	query.Load(config.Conf)
	Load(config.Conf)
}
func TestProcessSetting(t *testing.T) {

	args := []interface{}{
		"service.compare",
		nil,
		&gin.Context{},
	}
	process := gou.NewProcess("xiang.chart.Setting", args...)
	response := ProcessSetting(process)
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.True(t, res.Has("name"))
	assert.True(t, res.Has("label"))
	assert.True(t, res.Has("description"))
	assert.True(t, res.Has("page"))
	assert.True(t, res.Has("version"))

	args = []interface{}{
		"service.compare",
		"page,name",
		&gin.Context{},
	}
	process = gou.NewProcess("xiang.chart.Setting", args...)
	response = ProcessSetting(process)
	assert.NotNil(t, response)

	res = any.Of(response).Map()
	assert.True(t, res.Has("name"))
	assert.True(t, res.Has("page"))
	assert.False(t, res.Has("label"))
}

func TestProcessData(t *testing.T) {

	params := url.Values{
		"from": []string{"1981-01-01", "1990-01-01"},
	}
	params.Set("to", "2049-12-31")

	args := []interface{}{
		"service.compare",
		params,
		&gin.Context{},
	}
	process := gou.NewProcess("xiang.chart.Data", args...)
	response := ProcessData(process)
	utils.Dump(response)

	assert.NotNil(t, response)

	res := any.Of(response).Map().Dot()
	assert.Equal(t, "北京", res.Get("合并.0.城市"))
}
