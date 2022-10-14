package importer

// PreviewAuto 一直显示
const PreviewAuto = "auto"

// PreviewAlways 一直显示
const PreviewAlways = "always"

// PreviewNever 从不显示
const PreviewNever = "never"

// Importer 数据导入器
type Importer struct {
	Title   string            `json:"title,omitempty"`  // 导入名称
	Process string            `json:"process"`          // 处理器名称
	Output  string            `json:"output,omitempty"` // The process import output
	Columns []Column          `json:"columns"`          // 字段列表
	Option  Option            `json:"option,omitempty"` // 导入配置项
	Rules   map[string]string `json:"rules,omitempty"`  // 许可导入规则
	Sid     string            `json:"-"`                // sid
}

// Column 导入字段定义
type Column struct {
	Label    string   `json:"label"`              // 字段标签
	Name     string   `json:"name"`               // 字段名称
	Field    string   `json:"field"`              // 字段名称(原始值)
	Match    []string `json:"match,omitempty"`    // 匹配建议
	Rules    []string `json:"rules,omitempty"`    // 清洗规则定义
	Nullable bool     `json:"nullable,omitempty"` // 是否可以为空
	Primary  bool     `json:"primary,omitempty"`  // 是否为主键

	Key      string // 字段键名 Object Only
	IsArray  bool   // 字段是否为 Array
	IsObject bool   // 字段是否为 Object
}

// Option 导入配置项定
type Option struct {
	UseTemplate    bool   `json:"useTemplate,omitempty"`    // 使用已匹配过的模板
	TemplateLink   string `json:"templateLink,omitempty"`   // 默认数据模板链接
	ChunkSize      int    `json:"chunkSize,omitempty"`      // 每次处理记录数量
	MappingPreview string `json:"mappingPreview,omitempty"` // 显示字段映射界面方式 auto 匹配模板失败显示, always 一直显示, never 不显示
	DataPreview    string `json:"dataPreview,omitempty"`    // 数据预览界面方式 auto 有异常数据时显示, always 一直显示, never 不显示
}

// Mapping 字段映射表
type Mapping struct {
	Sheet            string     `json:"sheet"`            // 数据表
	ColStart         int        `json:"colStart"`         // 第一列的位置
	RowStart         int        `json:"rowStart"`         // 第一行的位置
	Columns          []*Binding `json:"data"`             // 字段数据列表
	AutoMatching     bool       `json:"autoMatching"`     // 是否自动匹配
	TemplateMatching bool       `json:"templateMatching"` // 是否通过已传模板匹配
}

// Binding 数据绑定
type Binding struct {
	Label string   `json:"label"` // 目标字段标签
	Field string   `json:"field"` // 目标字段名称
	Name  string   `json:"name"`  // 源关联字段名称
	Axis  string   `json:"axis"`  // 源关联字段坐标
	Value string   `json:"value"` // 示例数据
	Rules []string `json:"rules"` // 清洗规则
}
