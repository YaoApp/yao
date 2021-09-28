package global

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/xiang/table"
)

func TestProcessPing(t *testing.T) {
	process := gou.NewProcess("xiang.global.ping")
	res, ok := processPing(process).(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, res["version"], VERSION)
}

func TestProcessSearch(t *testing.T) {
	args := []interface{}{
		"service",
		gou.QueryParam{
			Wheres: []gou.QueryWhere{
				{Column: "status", Value: "enabled"},
			},
		},
		1,
		2,
		&gin.Context{},
	}
	process := gou.NewProcess("xiang.table.Search", args...)
	response := table.ProcessSearch(process)
	assert.NotNil(t, response)
	res := any.Of(response).Map()
	assert.True(t, res.Has("data"))
	assert.True(t, res.Has("next"))
	assert.True(t, res.Has("page"))
	assert.True(t, res.Has("pagecnt"))
	assert.True(t, res.Has("pagesize"))
	assert.True(t, res.Has("prev"))
	assert.True(t, res.Has("total"))
	assert.Equal(t, 1, res.Get("page"))
	assert.Equal(t, 2, res.Get("pagesize"))
}
