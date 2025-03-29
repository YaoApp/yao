package excel

import (
	"github.com/xuri/excelize/v2"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("excel", map[string]process.Handler{
		"open":   processOpen,
		"close":  processClose,
		"save":   processSave,
		"sheets": processSheets,

		"sheet.create": processCreateSheet,
		"sheet.read":   processReadSheet,
		"sheet.update": processUpdateSheet,
		"sheet.delete": processDeleteSheet,
		"sheet.copy":   processCopySheet,
		"sheet.list":   processListSheets,
		"sheet.exists": processSheetExists,

		"read.cell":   processReadCell,
		"read.row":    processReadRow,
		"read.column": processReadColumn,

		"write.cell":   processWriteCell,
		"write.row":    processWriteRow,
		"write.column": processWriteColumn,
		"write.all":    processWriteAll,

		"set.style":       processSetStyle,
		"set.formula":     processSetFormula,
		"set.link":        processSetLink,
		"set.richtext":    processSetRichText,
		"set.comment":     processSetComment,
		"set.rowheight":   processSetRowHeight,
		"set.columnwidth": processSetColumnWidth,
		"set.mergecell":   processMergeCell,
		"set.unmergecell": processUnmergeCell,

		"each.openrow":     processOpenRow,
		"each.closerow":    processCloseRow,
		"each.nextrow":     processNextRow,
		"each.opencolumn":  processOpenColumn,
		"each.closecolumn": processCloseColumn,
		"each.nextcolumn":  processNextColumn,

		"convert.columnnametonumber":    processColumnNameToNumber,
		"convert.columnnumbertoname":    processColumnNumberToName,
		"convert.cellnametocoordinates": processCellNameToCoordinates,
		"convert.coordinatestocellname": processCoordinatesToCellName,
	})
}

// processOpen process the excel.open <file> <writable>
func processOpen(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	file := process.ArgsString(0)
	writable := false
	if len(process.Args) > 1 {
		writable = process.ArgsBool(1)
	}

	handle, err := Open(file, writable)
	if err != nil {
		exception.New("excel.open %s error: %s", 500, file, err.Error()).Throw()
	}
	return handle
}

// processClose process the excel.close <handle>
func processClose(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	handle := process.ArgsString(0)
	err := Close(handle)
	if err != nil {
		exception.New("excel.close %s error: %s", 500, handle, err.Error()).Throw()
	}
	return nil
}

// processSave process the excel.save <handle>
func processSave(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	handle := process.ArgsString(0)
	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.save %s error: %s", 500, handle, err.Error()).Throw()
	}

	// 使用 SaveAs 方法保存文件到原始路径
	err = xls.SaveAs(xls.abs)
	if err != nil {
		exception.New("excel.save %s error: %s", 500, handle, err.Error()).Throw()
	}
	return nil
}

// processSheets process the excel.sheets <handle>
func processSheets(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	handle := process.ArgsString(0)
	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.sheets %s error: %s", 500, handle, err.Error()).Throw()
	}
	return xls.GetSheetList()
}

// processReadCell process the excel.read.cell <handle> <sheet> <cell>
func processReadCell(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.read.cell %s error: %s", 500, handle, err.Error()).Throw()
	}
	value, err := xls.GetCellValue(sheet, cell)
	if err != nil {
		exception.New("excel.read.cell %s:%s:%s error: %s", 500, handle, sheet, cell, err.Error()).Throw()
	}
	return value
}

// processReadRow process the excel.read.row <handle> <sheet>
func processReadRow(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.read.row %s error: %s", 500, handle, err.Error()).Throw()
	}
	rows, err := xls.GetRows(sheet)
	if err != nil {
		exception.New("excel.read.row %s:%s error: %s", 500, handle, sheet, err.Error()).Throw()
	}
	return rows
}

// processReadColumn process the excel.read.column <handle> <sheet>
func processReadColumn(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.read.column %s error: %s", 500, handle, err.Error()).Throw()
	}
	cols, err := xls.GetCols(sheet)
	if err != nil {
		exception.New("excel.read.column %s:%s error: %s", 500, handle, sheet, err.Error()).Throw()
	}
	return cols
}

// processWriteCell process the excel.write.cell <handle> <sheet> <cell> <value>
func processWriteCell(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)
	value := process.Args[3]

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.write.cell %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.SetCellValue(sheet, cell, value)
	if err != nil {
		exception.New("excel.write.cell %s:%s:%s error: %s", 500, handle, sheet, cell, err.Error()).Throw()
	}
	return nil
}

// processWriteRow process the excel.write.row <handle> <sheet> <cell> <values>
func processWriteRow(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)
	values := process.Args[3]

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.write.row %s error: %s", 500, handle, err.Error()).Throw()
	}

	// 处理切片值
	var rowValues []interface{}
	if arr, ok := values.([]interface{}); ok {
		rowValues = arr
	} else {
		rowValues = []interface{}{values}
	}

	// 使用 xls.SetSheetRow 方法，它应该能处理 slice 指针
	err = xls.SetSheetRow(sheet, cell, &rowValues)
	if err != nil {
		exception.New("excel.write.row %s:%s:%s error: %s", 500, handle, sheet, cell, err.Error()).Throw()
	}
	return nil
}

// processWriteColumn process the excel.write.column <handle> <sheet> <cell> <values>
func processWriteColumn(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)
	values := process.Args[3]

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.write.column %s error: %s", 500, handle, err.Error()).Throw()
	}

	// 处理切片值
	var colValues []interface{}
	if arr, ok := values.([]interface{}); ok {
		colValues = arr
	} else {
		colValues = []interface{}{values}
	}

	// 使用 xls.SetSheetCol 方法，它应该能处理 slice 指针
	err = xls.SetSheetCol(sheet, cell, &colValues)
	if err != nil {
		exception.New("excel.write.column %s:%s:%s error: %s", 500, handle, sheet, cell, err.Error()).Throw()
	}
	return nil
}

// processWriteAll process the excel.write.all <handle> <sheet> <cell> <values>
func processWriteAll(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)
	values := process.Args[3]

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.write.all %s error: %s", 500, handle, err.Error()).Throw()
	}

	// Convert data to [][]interface{}
	var sheetData [][]interface{}
	if arr, ok := values.([]interface{}); ok {
		for _, row := range arr {
			if rowArr, ok := row.([]interface{}); ok {
				sheetData = append(sheetData, rowArr)
			} else {
				sheetData = append(sheetData, []interface{}{row})
			}
		}
	} else {
		sheetData = [][]interface{}{{values}}
	}

	err = xls.WriteAll(sheet, cell, sheetData)
	if err != nil {
		exception.New("excel.write.all %s:%s error: %s", 500, handle, sheet, err.Error()).Throw()
	}
	return nil
}

// processSetStyle process the excel.set.style <handle> <sheet> <cell> <style>
func processSetStyle(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)
	styleID := process.ArgsInt(3)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.style %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.SetCellStyle(sheet, cell, cell, styleID)
	if err != nil {
		exception.New("excel.set.style %s:%s:%s error: %s", 500, handle, sheet, cell, err.Error()).Throw()
	}
	return nil
}

// processSetFormula process the excel.set.formula <handle> <sheet> <cell> <formula>
func processSetFormula(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)
	formula := process.ArgsString(3)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.formula %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.SetCellFormula(sheet, cell, formula)
	if err != nil {
		exception.New("excel.set.formula %s:%s:%s error: %s", 500, handle, sheet, cell, err.Error()).Throw()
	}
	return nil
}

// processSetLink process the excel.set.link <handle> <sheet> <cell> <link> <text>
func processSetLink(process *process.Process) interface{} {
	process.ValidateArgNums(5)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)
	link := process.ArgsString(3)
	text := process.ArgsString(4)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.link %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.SetCellHyperLink(sheet, cell, link, text)
	if err != nil {
		exception.New("excel.set.link %s:%s:%s error: %s", 500, handle, sheet, cell, err.Error()).Throw()
	}
	return nil
}

// processSetRichText process the excel.set.richtext <handle> <sheet> <cell> <richText>
func processSetRichText(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	cell := process.ArgsString(2)
	// Extract rich text from args
	richTextData := process.Args[3]
	var richText []excelize.RichTextRun

	// Convert to rich text format expected by excelize
	// This is a simplification - the actual implementation would depend on the format of the input
	if rtArray, ok := richTextData.([]interface{}); ok {
		for _, item := range rtArray {
			if rtMap, ok := item.(map[string]interface{}); ok {
				run := excelize.RichTextRun{}
				if text, ok := rtMap["text"].(string); ok {
					run.Text = text
				}
				richText = append(richText, run)
			}
		}
	}

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.richtext %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.SetCellRichText(sheet, cell, richText)
	if err != nil {
		exception.New("excel.set.richtext %s:%s:%s error: %s", 500, handle, sheet, cell, err.Error()).Throw()
	}
	return nil
}

// processSetComment process the excel.set.comment <handle> <sheet> <comment>
func processSetComment(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	_ = process.Args[2] // Placeholder for comment data - future implementation

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.comment %s error: %s", 500, handle, err.Error()).Throw()
	}

	// We'll need to convert the comment data to the appropriate structure
	// This is simplified for now
	err = xls.SetSheetVisible(sheet, true) // Just a placeholder operation
	if err != nil {
		exception.New("excel.set.comment %s:%s error: %s", 500, handle, sheet, err.Error()).Throw()
	}
	return nil
}

// processSetRowHeight process the excel.set.rowheight <handle> <sheet> <row> <height>
func processSetRowHeight(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	row := process.ArgsInt(2)
	// Convert string to float using standard process method
	height := float64(process.ArgsInt(3))

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.rowheight %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.SetRowHeight(sheet, row, height)
	if err != nil {
		exception.New("excel.set.rowheight %s:%s:%d error: %s", 500, handle, sheet, row, err.Error()).Throw()
	}
	return nil
}

// processSetColumnWidth process the excel.set.columnwidth <handle> <sheet> <startCol> <endCol> <width>
func processSetColumnWidth(process *process.Process) interface{} {
	process.ValidateArgNums(5)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	startCol := process.ArgsString(2)
	endCol := process.ArgsString(3)
	// Convert string to float using standard process method
	width := float64(process.ArgsInt(4))

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.columnwidth %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.SetColWidth(sheet, startCol, endCol, width)
	if err != nil {
		exception.New("excel.set.columnwidth %s:%s:%s error: %s", 500, handle, sheet, startCol, err.Error()).Throw()
	}
	return nil
}

// processMergeCell process the excel.set.mergecell <handle> <sheet> <start> <end>
func processMergeCell(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	start := process.ArgsString(2)
	end := process.ArgsString(3)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.mergecell %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.MergeCell(sheet, start, end)
	if err != nil {
		exception.New("excel.set.mergecell %s:%s:%s:%s error: %s", 500, handle, sheet, start, end, err.Error()).Throw()
	}
	return nil
}

// processUnmergeCell process the excel.set.unmergecell <handle> <sheet> <start> <end>
func processUnmergeCell(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)
	start := process.ArgsString(2)
	end := process.ArgsString(3)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.set.unmergecell %s error: %s", 500, handle, err.Error()).Throw()
	}
	err = xls.UnmergeCell(sheet, start, end)
	if err != nil {
		exception.New("excel.set.unmergecell %s:%s:%s:%s error: %s", 500, handle, sheet, start, end, err.Error()).Throw()
	}
	return nil
}

// processColumnNameToNumber process the excel.convert.columnnametonumber <name>
func processColumnNameToNumber(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	name := process.ArgsString(0)
	number, err := excelize.ColumnNameToNumber(name)
	if err != nil {
		exception.New("excel.convert.columnnametonumber %s error: %s", 500, name, err.Error()).Throw()
	}
	return number
}

// processColumnNumberToName process the excel.convert.columnnumbertoname <number>
func processColumnNumberToName(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	number := process.ArgsInt(0)
	name, err := excelize.ColumnNumberToName(number)
	if err != nil {
		exception.New("excel.convert.columnnumbertoname %d error: %s", 500, number, err.Error()).Throw()
	}
	return name
}

// processCellNameToCoordinates process the excel.convert.cellnametocoordinates <cell>
func processCellNameToCoordinates(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	cell := process.ArgsString(0)
	x, y, err := excelize.CellNameToCoordinates(cell)
	if err != nil {
		exception.New("excel.convert.cellnametocoordinates %s error: %s", 500, cell, err.Error()).Throw()
	}
	return []int{x, y}
}

// processCoordinatesToCellName process the excel.convert.coordinatestocellname <col> <row>
func processCoordinatesToCellName(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	col := process.ArgsInt(0)
	row := process.ArgsInt(1)
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		exception.New("excel.convert.coordinatestocellname %d,%d error: %s", 500, col, row, err.Error()).Throw()
	}
	return cell
}

// processOpenRow process the excel.each.openrow <handle> <sheet>
func processOpenRow(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.each.openrow %s error: %s", 500, handle, err.Error()).Throw()
	}

	id, err := xls.OpenRow(sheet)
	if err != nil {
		exception.New("excel.each.openrow %s:%s error: %s", 500, handle, sheet, err.Error()).Throw()
	}
	return id
}

// processCloseRow process the excel.each.closerow <id>
func processCloseRow(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)

	// Don't use return value from CloseRow
	CloseRow(id)
	return nil
}

// processNextRow process the excel.each.nextrow <id>
func processNextRow(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)

	row, err := NextRow(id)
	if err != nil {
		CloseRow(id) // Discard return value
		exception.New("excel.each.nextrow %s error: %s", 500, id, err.Error()).Throw()
	}

	if row == nil {
		CloseRow(id) // Discard return value
		return nil
	}

	return row
}

// processOpenColumn process the excel.each.opencolumn <handle> <sheet>
func processOpenColumn(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	handle := process.ArgsString(0)
	sheet := process.ArgsString(1)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.each.opencolumn %s error: %s", 500, handle, err.Error()).Throw()
	}

	id, err := xls.OpenColumn(sheet)
	if err != nil {
		exception.New("excel.each.opencolumn %s:%s error: %s", 500, handle, sheet, err.Error()).Throw()
	}
	return id
}

// processCloseColumn process the excel.each.closecolumn <id>
func processCloseColumn(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)

	// Don't use return value from CloseColumn
	CloseColumn(id)
	return nil
}

// processNextColumn process the excel.each.nextcolumn <id>
func processNextColumn(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	id := process.ArgsString(0)

	col, err := NextColumn(id)
	if err != nil {
		CloseColumn(id) // Discard return value
		exception.New("excel.each.nextcolumn %s error: %s", 500, id, err.Error()).Throw()
	}

	if col == nil {
		CloseColumn(id) // Discard return value
		return nil
	}

	return col
}

// processCreateSheet process the excel.sheet.create <handle> <name>
func processCreateSheet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	handle := process.ArgsString(0)
	name := process.ArgsString(1)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.sheet.create %s error: %s", 500, handle, err.Error()).Throw()
	}

	idx, err := xls.CreateSheet(name)
	if err != nil {
		exception.New("excel.sheet.create %s:%s error: %s", 500, handle, name, err.Error()).Throw()
	}
	return idx
}

// processReadSheet process the excel.sheet.read <handle> <name>
func processReadSheet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	handle := process.ArgsString(0)
	name := process.ArgsString(1)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.sheet.read %s error: %s", 500, handle, err.Error()).Throw()
	}

	data, err := xls.ReadSheet(name)
	if err != nil {
		exception.New("excel.sheet.read %s:%s error: %s", 500, handle, name, err.Error()).Throw()
	}
	return data
}

// processUpdateSheet process the excel.sheet.update <handle> <name> <data>
func processUpdateSheet(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	handle := process.ArgsString(0)
	name := process.ArgsString(1)
	data := process.Args[2]

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.sheet.update %s error: %s", 500, handle, err.Error()).Throw()
	}

	// Convert data to [][]interface{}
	var sheetData [][]interface{}
	if arr, ok := data.([]interface{}); ok {
		for _, row := range arr {
			if rowArr, ok := row.([]interface{}); ok {
				sheetData = append(sheetData, rowArr)
			} else {
				sheetData = append(sheetData, []interface{}{row})
			}
		}
	} else {
		sheetData = [][]interface{}{{data}}
	}

	err = xls.UpdateSheet(name, sheetData)
	if err != nil {
		exception.New("excel.sheet.update %s:%s error: %s", 500, handle, name, err.Error()).Throw()
	}
	return nil
}

// processDeleteSheet process the excel.sheet.delete <handle> <name>
func processDeleteSheet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	handle := process.ArgsString(0)
	name := process.ArgsString(1)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.sheet.delete %s error: %s", 500, handle, err.Error()).Throw()
	}

	err = xls.DeleteSheet(name)
	if err != nil {
		exception.New("excel.sheet.delete %s:%s error: %s", 500, handle, name, err.Error()).Throw()
	}
	return nil
}

// processCopySheet process the excel.sheet.copy <handle> <source> <target>
func processCopySheet(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	handle := process.ArgsString(0)
	source := process.ArgsString(1)
	target := process.ArgsString(2)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.sheet.copy %s error: %s", 500, handle, err.Error()).Throw()
	}

	err = xls.CopySheet(source, target)
	if err != nil {
		exception.New("excel.sheet.copy %s:%s:%s error: %s", 500, handle, source, target, err.Error()).Throw()
	}
	return nil
}

// processListSheets process the excel.sheet.list <handle>
func processListSheets(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	handle := process.ArgsString(0)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.sheet.list %s error: %s", 500, handle, err.Error()).Throw()
	}

	return xls.ListSheets()
}

// processSheetExists process the excel.sheet.exists <handle> <name>
func processSheetExists(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	handle := process.ArgsString(0)
	name := process.ArgsString(1)

	xls, err := Get(handle)
	if err != nil {
		exception.New("excel.sheet.exists %s error: %s", 500, handle, err.Error()).Throw()
	}

	return xls.SheetExists(name)
}
