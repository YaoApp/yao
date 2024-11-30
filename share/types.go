package share

import "github.com/yaoapp/kun/maps"

// Importable 可导入JSON
type Importable struct {
	Import string        `json:"@,omitempty"`  // 从 Global 或 Vendor 载入
	In     []interface{} `json:"in,omitempty"` // 从 Global 或 Vendor 载入, 解析参数
}

// APIImp 导入配置数据结构
type APIImp API

// API API 配置数据结构
type API struct {
	Name    string        `json:"-"`
	Source  string        `json:"-"`
	Disable bool          `json:"disable,omitempty"`
	Process string        `json:"process,omitempty"`
	Guard   string        `json:"guard,omitempty"`
	Default []interface{} `json:"default,omitempty"`
	Importable
}

// ColumnImp 导入模式查询过滤器
type ColumnImp Column

// Column 字段呈现方式
type Column struct {
	Label  string `json:"label"`
	Export string `json:"export,omitempty"`
	View   Render `json:"view,omitempty"`
	Edit   Render `json:"edit,omitempty"`
	Form   Render `json:"form,omitempty"`
	Importable
}

// FilterImp 导入模式查询过滤器
type FilterImp Filter

// Filter 查询过滤器
type Filter struct {
	Label string `json:"label"`
	Bind  string `json:"bind,omitempty"`
	Input Render `json:"input,omitempty"`
	Importable
}

// RenderImp 导入模式组件渲染方式
type RenderImp Render

// Render 组件渲染方式
type Render struct {
	Type       string                 `json:"type,omitempty"`
	Props      map[string]interface{} `json:"props,omitempty"`
	Components map[string]interface{} `json:"components,omitempty"`
	Importable
}

// PageImp 导入模式页面
type PageImp Page

// Page 页面
type Page struct {
	Primary string                 `json:"primary"`
	Layout  map[string]interface{} `json:"layout"`
	Actions map[string]Render      `json:"actions,omitempty"`
	Option  map[string]interface{} `json:"option,omitempty"`
	Importable
}

// AppInfo 应用信息
type AppInfo struct {
	Name         string                 `json:"name,omitempty"`
	L            map[string]string      `json:"-"`
	Short        string                 `json:"short,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Icons        maps.MapStrSync        `json:"icons,omitempty"`
	Storage      AppStorage             `json:"storage,omitempty"`
	Option       map[string]interface{} `json:"option,omitempty"`
	XGen         string                 `json:"xgen,omitempty"`
	AdminRoot    string                 `json:"adminRoot,omitempty"`
	Static       Static                 `json:"public,omitempty"`
	Optional     map[string]interface{} `json:"optional,omitempty"`
	Moapi        Moapi                  `json:"moapi,omitempty"`
	AfterLoad    string                 `json:"afterLoad,omitempty"`    // Process executed after the app is loaded
	AfterMigrate string                 `json:"afterMigrate,omitempty"` // Process executed after the app is migrated
}

// Moapi AIGC App Store API
type Moapi struct {
	Channel      string   `json:"channel,omitempty"`
	Mirrors      []string `json:"mirrors,omitempty"`
	Secret       string   `json:"secret,omitempty"`
	Organization string   `json:"organization,omitempty"`
}

// Static setting
type Static struct {
	DisableGzip bool                `json:"disableGzip,omitempty"`
	Rewrite     []map[string]string `json:"rewrite,omitempty"`
	SourceRoots map[string]string   `json:"sourceRoots,omitempty"`
}

// AppStorage 应用存储
type AppStorage struct {
	Default string                 `json:"default"`
	Buckets map[string]string      `json:"buckets,omitempty"`
	S3      map[string]interface{} `json:"s3,omitempty"`
	OSS     *AppStorageOSS         `json:"oss,omitempty"`
	COS     map[string]interface{} `json:"cos,omitempty"`
}

// AppStorageOSS 阿里云存储
type AppStorageOSS struct {
	Endpoint    string `json:"endpoint,omitempty"`
	ID          string `json:"id,omitempty"`
	Secret      string `json:"secret,omitempty"`
	RoleArn     string `json:"roleArn,omitempty"`
	SessionName string `json:"sessionName,omitempty"`
}

// Script 脚本文件类型
type Script struct {
	Name    string
	Type    string
	Content []byte
	File    string
}

// AppRoot 应用目录
type AppRoot struct {
	APIs    string
	Flows   string
	Models  string
	Plugins string
	Tables  string
	Charts  string
	Screens string
	Data    string
}
