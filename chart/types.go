package chart

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/share"
)

// Chart 图表格式
type Chart struct {
	gou.Flow
	APIs    map[string]API          `json:"apis,omitempty"`
	Filters map[string]share.Filter `json:"filters,omitempty"`
	Page    share.Page              `json:"page,omitempty"`
}

// API 图表 API
type API struct {
	Disable bool        `json:"disable,omitempty"`
	Guard   string      `json:"guard,omitempty"`
	Default interface{} `json:"default,omitempty"`
}
