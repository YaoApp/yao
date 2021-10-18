package table

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/share"
)

// Table 数据表格配置结构
type Table struct {
	Table      string                  `json:"-"`
	Source     string                  `json:"-"`
	Name       string                  `json:"name"`
	Version    string                  `json:"version"`
	Title      string                  `json:"title,omitempty"`
	Decription string                  `json:"decription,omitempty"`
	Bind       Bind                    `json:"bind,omitempty"`
	APIs       map[string]share.API    `json:"apis,omitempty"`
	Columns    map[string]share.Column `json:"columns,omitempty"`
	Filters    map[string]share.Filter `json:"filters,omitempty"`
	List       share.Page              `json:"list,omitempty"`
	Edit       share.Page              `json:"edit,omitempty"`
	View       share.Page              `json:"view,omitempty"`
	Insert     share.Page              `json:"insert,omitempty"`
}

// Bind 绑定数据模型
type Bind struct {
	Model string              `json:"model"`
	Withs map[string]gou.With `json:"withs,omitempty"`
}
