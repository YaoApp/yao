package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
)

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
	}
	process := gou.NewProcess("xiang.table.Search", args...)
	response := processSearch(process)
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
