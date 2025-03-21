package excel

import (
	"github.com/xuri/excelize/v2"
)

// WriteCell write the cell
func (excel *Excel) WriteCell(sheet string, cell string, value interface{}) error {

	_, err := excel.SetSheet(sheet)
	if err != nil {
		return err
	}
	return excel.SetCellValue(sheet, cell, value)
}

// WriteRow write the row
func (excel *Excel) WriteRow(sheet string, cell string, value []interface{}) error {

	_, err := excel.SetSheet(sheet)
	if err != nil {
		return err
	}

	return excel.SetSheetRow(sheet, cell, &value)
}

// WriteColumn write the column
func (excel *Excel) WriteColumn(sheet string, cell string, value []interface{}) error {

	_, err := excel.SetSheet(sheet)
	if err != nil {
		return err
	}
	return excel.SetSheetCol(sheet, cell, &value)
}

// WriteAll write all the sheet
func (excel *Excel) WriteAll(sheet string, cell string, rows [][]interface{}) error {

	_, err := excel.SetSheet(sheet)
	if err != nil {
		return err
	}

	for _, row := range rows {
		if err := excel.SetSheetRow(sheet, cell, &row); err != nil {
			return err
		}

		colIndex, rowIndex, err := excelize.CellNameToCoordinates(cell)
		cell, err = excelize.CoordinatesToCellName(colIndex, rowIndex+1)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetSheet set the sheet
func (excel *Excel) SetSheet(name string) (int, error) {

	idx, err := excel.GetSheetIndex(name)
	if err != nil {
		return 0, err
	}

	if idx == -1 {
		idx, err = excel.NewSheet(name)
		if err != nil {
			return 0, err
		}
	}
	return idx, nil
}
