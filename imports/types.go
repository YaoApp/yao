package imports

import "github.com/yaoapp/gou"

// PreviewAuto 一直显示
const PreviewAuto = "auto"

// PreviewAlways 一直显示
const PreviewAlways = "always"

// PreviewNever 从不显示
const PreviewNever = "never"

// Imports 数据导入
type Imports struct {
	Title   string       `json:"title,omitempty"`  // 导入名称
	Process *gou.Process `json:"process"`          // 处理器名称
	Columns []Column     `json:"columns"`          // 字段列表
	Option  Option       `json:"option,omitempty"` // 导入配置项
}

// Column 导入字段定义
type Column struct {
	Label    string        `json:"label"`              // 字段标签
	Name     string        `json:"name"`               // 字段名称
	Match    []string      `json:"match,omitempty"`    // 匹配建议
	Rules    []gou.Process `json:"rules,omitempty"`    // 清洗规则定义
	Nullable bool          `json:"nullable,omitempty"` // 是否可以为空

	Key      string // 字段键名 Object Only
	IsArray  bool   // 字段是否为 Array
	IsObject bool   // 字段是否为 Object
}

// Option 导入配置项定
type Option struct {
	AutoMatching   bool   `json:"autoMatching,omitempty"`   // 是否自动匹配已设定模板
	ChunkSize      int    `json:"chunkSize,omitempty"`      // 每次处理记录数量
	MappingPreview string `json:"mappingPreview,omitempty"` // 显示字段映射界面方式 auto 匹配模板失败显示, always 一直显示, never 不显示
	DataPreview    string `json:"dataPreview,omitempty"`    // 数据预览界面方式 auto 有异常数据时显示, always 一直显示, never 不显示
}
