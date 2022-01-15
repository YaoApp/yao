package from

// Source 导入文件接口
type Source interface {
	Data(page int, size int) []map[string]interface{}
	Columns() []Column
	Bind(mapping Mapping)
	Close() error
}

// Column 源数据列
type Column struct {
	Name string
	Type string
	Col  int
	Row  int
	Pos  string
}

// Mapping 数值映射表
type Mapping struct{}
