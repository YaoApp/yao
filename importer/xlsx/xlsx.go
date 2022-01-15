package xlsx

import (
	"fmt"

	"github.com/xuri/excelize/v2"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xiang/importer/from"
	"github.com/yaoapp/xiang/xlog"
)

// Xlsx xlsx file
type Xlsx struct {
	File       *excelize.File
	SheetName  string
	SheetIndex int
	Cols       *excelize.Cols
	Rows       *excelize.Rows
}

// Open 打开 Xlsx 文件
func Open(filename string) *Xlsx {
	file, err := excelize.OpenFile(filename)
	if err != nil {
		exception.New("打开文件错误 %s", 400, err.Error()).Throw()
	}

	sheetIndex := file.GetActiveSheetIndex()
	sheetName := file.GetSheetName(sheetIndex)

	rows, err := file.Rows(sheetName)
	if err != nil {
		exception.New("读取表格行失败 %s %s", 400, sheetName, err.Error()).Throw()
	}

	if rows.TotalRows() > 100000 {
		exception.New("数据表 %s 超过10万行 %d", 400, sheetName, rows.TotalRows()).Throw()
	}

	cols, err := file.Cols(sheetName)
	if err != nil {
		exception.New("读取表格列信息失败 %s %s", 400, sheetName, err.Error()).Throw()
	}
	if cols.TotalCols() > 1000 {
		exception.New("数据表 %s 超过1000列 %d", 400, sheetName, cols.TotalCols()).Throw()
	}

	return &Xlsx{File: file, Rows: rows, Cols: cols, SheetName: sheetName, SheetIndex: sheetIndex}
}

// Close 关闭文件句柄
func (xlsx *Xlsx) Close() error {
	if err := xlsx.File.Close(); err != nil {
		xlog.Println(err.Error())
		return err
	}
	return nil
}

// Data 读取数据
func (xlsx *Xlsx) Data(page int, size int) []map[string]interface{} {
	return nil
}

// Columns 读取列
func (xlsx *Xlsx) Columns() []from.Column {
	fmt.Println(xlsx.SheetName, xlsx.Rows.TotalRows(), xlsx.Cols.TotalCols())

	// 扫描标题位置坐标 扫描行
	pos := []int{0, 0, 0} // {行, 开始列, 结束列}
	line := 0
	success := false
	for xlsx.Rows.Next() {
		row, err := xlsx.Rows.Columns()
		if err != nil {
			exception.New("数据表 %s 扫描行 %d 信息失败 %", 400, xlsx.SheetName, line, err.Error()).Throw()
		}

		// 扫描列
		for i, cell := range row {
			if cell != "" {
				pos = []int{line, i, len(row)}
				success = true
				break
			}
		}

		if success == true {
			break
		}
		line++
	}

	fmt.Println(pos)

	return nil
}

// Bind 绑定映射表
func (xlsx *Xlsx) Bind(mapping from.Mapping) {
}
