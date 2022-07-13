package chart

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/share"
)

// Chart 图表格式
type Chart struct {
	*gou.Flow
	APIs    map[string]share.API    `json:"apis,omitempty"`
	Filters map[string]share.Filter `json:"filters,omitempty"`
	Page    share.Page              `json:"page,omitempty"`
}
