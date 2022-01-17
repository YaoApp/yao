package from

const (
	// TUnknown 未知
	TUnknown byte = iota
	// TBool bool
	TBool
	// TDatetime 日期时间
	TDatetime
	// TError 错误
	TError
	// TNumber 数字
	TNumber
	// TString 字符串
	TString
)

// Source 导入文件接口
type Source interface {
	Data(page int, size int) []map[string]interface{}
	Columns() []Column
	Inspect() Inspect
	Bind()
	Close() error
}

// Column 源数据列
type Column struct {
	Name string
	Type byte
	Col  int
	Row  int
	Axis string
}

// Inspect 基础信息
type Inspect struct {
	SheetName  string
	SheetIndex int
	ColStart   int
	RowStart   int
}
