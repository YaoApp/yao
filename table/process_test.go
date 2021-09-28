package table

import (
	"testing"

	"github.com/yaoapp/gou"
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
	processSearch(process)

}
