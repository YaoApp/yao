package table

import "github.com/yaoapp/gou"

// Table 数据表格配置结构
type Table struct {
	Table   string            `json:"-"`
	Source  string            `json:"-"`
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Bind    Bind              `json:"bind,omitempty"`
	APIs    map[string]API    `json:"apis,omitempty"`
	Columns map[string]Column `json:"columns,omitempty"`
	Filters map[string]Filter `json:"filters,omitempty"`
	List    Page              `json:"list,omitempty"`
	Edit    Page              `json:"edit,omitempty"`
	View    Page              `json:"view,omitempty"`
	Insert  Page              `json:"insert,omitempty"`
}

// Bind 绑定数据模型
type Bind struct {
	Model string              `json:"model"`
	Withs map[string]gou.With `json:"withs,omitempty"`
}

// API API 配置数据结构
type API struct {
	Name    string        `json:"-"`
	Source  string        `json:"-"`
	Process string        `json:"process,omitempty"`
	Guard   string        `json:"guard,omitempty"`
	Default []interface{} `json:"default,omitempty"`
}

// Column 字段呈现方式
type Column struct {
	Label string `json:"label"`
	View  Render `json:"view,omitempty"`
	Edit  Render `json:"edit,omitempty"`
	Form  Render `json:"form,omitempty"`
}

// Filter 查询过滤器
type Filter struct {
	Label string `json:"label"`
	Bind  string `json:"bind,omitempty"`
	Input Render `json:"input,omitempty"`
}

// Page 页面
type Page struct {
	Primary string                 `json:"primary"`
	Layout  map[string]interface{} `json:"layout"`
	Actions map[string]Render      `json:"actions,omitempty"`
}

// Render 组件渲染方式
type Render struct {
	Type  string                 `json:"type,omitempty"`
	Props map[string]interface{} `json:"props,omitempty"`
}
