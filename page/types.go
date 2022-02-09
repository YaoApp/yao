package page

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/share"
)

// Page 页面格式
type Page struct {
	gou.Flow
	APIs    map[string]share.API    `json:"apis,omitempty"`
	Filters map[string]share.Filter `json:"filters,omitempty"`
	Page    share.Page              `json:"page,omitempty"`
}
