package xlsx

import (
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/importer/from"
)

// Xlsx xlsx file
type Xlsx struct {
	File       *excelize.File
	SheetName  string
	SheetIndex int
	ColStart   int
	RowStart   int
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

	// if rows.TotalRows() > 100000 {
	// 	exception.New("数据表 %s 超过10万行 %d", 400, sheetName, rows.TotalRows()).Throw()
	// }

	cols, err := file.Cols(sheetName)
	if err != nil {
		exception.New("读取表格列信息失败 %s %s", 400, sheetName, err.Error()).Throw()
	}

	// if cols.TotalCols() > 1000 {
	// 	exception.New("数据表 %s 超过1000列 %d", 400, sheetName, cols.TotalCols()).Throw()
	// }

	return &Xlsx{File: file, Rows: rows, Cols: cols, SheetName: sheetName, SheetIndex: sheetIndex}
}

// Close 关闭文件句柄
func (xlsx *Xlsx) Close() error {
	if err := xlsx.File.Close(); err != nil {
		log.Error("Close file error: %s", err.Error())
		return err
	}
	return nil
}

// Inspect 基本信息
func (xlsx *Xlsx) Inspect() from.Inspect {
	return from.Inspect{
		SheetName:  xlsx.SheetName,
		SheetIndex: xlsx.SheetIndex,
		RowStart:   xlsx.RowStart,
		ColStart:   xlsx.ColStart,
	}
}

// Data 读取数据
func (xlsx *Xlsx) Data(row int, size int, axises []string) [][]interface{} {
	data := [][]interface{}{}
	for line := row; line < row+size; line++ {
		row, end := xlsx.readLine(line, axises)
		if end {
			break
		}
		data = append(data, row)
	}
	return data
}

// Chunk 遍历数据
func (xlsx *Xlsx) Chunk(size int, axises []string, cb func(line int, data [][]interface{})) {
	line := 0
	data := [][]interface{}{}
	for xlsx.Rows.Next() {
		line++
		if line < xlsx.RowStart {
			continue
		}
		row, end := xlsx.readLine(line, axises)
		if end {
			cb(line, data)
			return
		}

		data = append(data, row)
		if line%size == 0 {
			cb(line, data)
			data = [][]interface{}{}
		}
	}

	// 最后一批数据
	if len(data) > 0 {
		cb(line, data)
	}
}

// readLine 读取给定行信息
func (xlsx *Xlsx) readLine(line int, axises []string) ([]interface{}, bool) {
	row := []interface{}{}
	end := true
	for _, axis := range axises {
		_, c, err := axisToPosition(axis)
		var value = ""
		if c >= 0 {
			axis := positionToAxis(line, c)
			value, err = xlsx.File.GetCellValue(xlsx.SheetName, axis)
			if err != nil {
				log.With(log.F{"SheetName": xlsx.SheetName, "axis": axis}).Error("读取数据出错 %s", err.Error())
				value = ""
			}
		}
		row = append(row, value)
		if value != "" {
			end = false
		}
	}
	return row, end
}

// Columns 读取列
func (xlsx *Xlsx) Columns() []from.Column {
	columns := []from.Column{}

	// 扫描标题位置坐标 扫描行
	// 从第一行开始扫描，识别第一个不为空的列
	line := 0
	success := false
	for xlsx.Rows.Next() {
		row, err := xlsx.Rows.Columns()
		if err != nil {
			exception.New("数据表 %s 扫描行 %d 信息失败 %s", 400, xlsx.SheetName, line, err.Error()).Throw()
		}

		// 扫描列
		// 从第一列开始扫描，识别第一个不为空的列
		for i, cell := range row {
			if cell != "" {
				success = true
				axis := positionToAxis(line, i)
				if xlsx.RowStart == 0 && xlsx.ColStart == 0 {
					xlsx.RowStart = line + 1
					xlsx.ColStart = i + 1
				}
				cellType, err := xlsx.File.GetCellType(xlsx.SheetName, axis)
				if err != nil {
					log.With(log.F{"SheetName": xlsx.SheetName, "axis": axis}).Error("读取数据类型失败 %s", err.Error())
				}
				columns = append(columns, from.Column{
					Name: cell,
					Axis: axis,
					Type: byte(cellType),
				})
			}
		}

		if success == true {
			break
		}
		line++
	}
	return columns
}

func (xlsx *Xlsx) getMergeCells() {
	cells, err := xlsx.File.GetMergeCells(xlsx.SheetName)
	if err != nil {
		exception.New("读取单元格 %s 失败 %s", 400, xlsx.SheetName, err.Error()).Throw()
		return
	}

	for _, cell := range cells {
		fmt.Println(cell.GetStartAxis())
	}
}

func positionToAxis(row, col int) string {
	if row < 0 || col < 0 {
		return ""
	}
	rowString := strconv.Itoa(row + 1)
	colString := ""
	col++
	for col > 0 {
		colString = fmt.Sprintf("%c%s", 'A'+col%26-1, colString)
		col /= 26
	}
	return colString + rowString
}

func axisToPosition(axis string) (int, int, error) {
	col := 0
	for i, char := range axis {
		if char >= 'A' && char <= 'Z' {
			col *= 26
			col += int(char - 'A' + 1)
		} else if char >= 'a' && char <= 'z' {
			col *= 26
			col += int(char - 'a' + 1)
		} else {
			row, err := strconv.Atoi(axis[i:])
			return row - 1, col - 1, err
		}
	}
	return -1, -1, fmt.Errorf("invalid axis format %s", axis)
}
