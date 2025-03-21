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

		"read.cell":  processReadCell,
		"write.cell": processWriteCell,

		"set.style":       processSetStyle,
		"set.formula":     processSetFormula,
		"set.link":        processSetLink,
		"set.mergecell":   processMergeCell,
		"set.unmergecell": processUnmergeCell,

		"convert.columnnametonumber":    processColumnNameToNumber,
		"convert.columnnumbertoname":    processColumnNumberToName,
		"convert.cellnametocoordinates": processCellNameToCoordinates,
		"convert.coordinatestocellname": processCoordinatesToCellName,
	})
}

// processOpen process the excel.open <file> <readonly>
func processOpen(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	file := process.ArgsString(0)
	readonly := false
	if len(process.Args) > 1 {
		readonly = process.ArgsBool(1)
	}

	handle, err := Open(file, readonly)
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
	err = xls.Save()
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
