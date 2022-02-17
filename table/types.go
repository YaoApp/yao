package table

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/share"
)

// Table 数据表格配置结构
type Table struct {
	Table      string                  `json:"-"`
	Source     string                  `json:"-"`
	Guard      string                  `json:"guard,omitempty"`
	Name       string                  `json:"name"`
	Version    string                  `json:"version"`
	Title      string                  `json:"title,omitempty"`
	Decription string                  `json:"decription,omitempty"`
	Bind       Bind                    `json:"bind,omitempty"`
	Hooks      Hooks                   `json:"hooks,omitempty"`
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

// Hooks 表格数据模型
type Hooks struct {
	BeforeFind        string `json:"before:find,omitempty"`
	AfterFind         string `json:"after:find,omitempty"`
	BeforeSearch      string `json:"before:search,omitempty"`
	AfterSearch       string `json:"after:search,omitempty"`
	BeforeSave        string `json:"before:save,omitempty"`
	AfterSave         string `json:"after:save,omitempty"`
	BeforeDelete      string `json:"before:delete,omitempty"`
	AfterDelete       string `json:"after:delete,omitempty"`
	BeforeInsert      string `json:"before:insert,omitempty"`
	AfterInsert       string `json:"after:insert,omitempty"`
	BeforeDeleteIn    string `json:"before:delete-in,omitempty"`
	AfterDeleteIn     string `json:"after:delete-in,omitempty"`
	BeforeDeleteWhere string `json:"before:delete-where,omitempty"`
	AfterDeleteWhere  string `json:"after:delete-where,omitempty"`
	BeforeUpdateIn    string `json:"before:update-in,omitempty"`
	AfterUpdateIn     string `json:"after:update-in,omitempty"`
	BeforeUpdateWhere string `json:"before:update-where,omitempty"`
	AfterUpdateWhere  string `json:"after:update-where,omitempty"`
	BeforeQuicksave   string `json:"before:quicksave,omitempty"`
	AfterQuicksave    string `json:"after:quicksave,omitempty"`
	BeforeSelect      string `json:"before:select,omitempty"`
	AfterSelect       string `json:"after:select,omitempty"`
}
