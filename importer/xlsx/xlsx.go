package xlsx

import "github.com/yaoapp/xiang/importer/from"

// Xlsx xlsx file
type Xlsx struct{}

// Open 打开 Xlsx 文件
func Open() *Xlsx {
	return &Xlsx{}
}

// Data 读取数据
func (xlsx *Xlsx) Data(page int, size int) []map[string]interface{} {
	return nil
}

// Columns 读取列
func (xlsx *Xlsx) Columns() []from.Column { return nil }

// Bind 绑定映射表
func (xlsx *Xlsx) Bind(mapping from.Mapping) {
}
