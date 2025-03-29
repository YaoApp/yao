package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// New creates a new Excel workbook
func New() (*Excel, error) {
	f := excelize.NewFile()
	return &Excel{
		File:   f,
		id:     "",
		path:   "",
		create: 0,
		abs:    "",
	}, nil
}

// CreateSheet creates a new sheet with the given name
// Returns the index of the new sheet and any error encountered
func (excel *Excel) CreateSheet(name string) (int, error) {
	// Check if sheet already exists
	if idx, _ := excel.GetSheetIndex(name); idx != -1 {
		return 0, fmt.Errorf("sheet %s already exists", name)
	}

	return excel.NewSheet(name)
}

// ReadSheet reads all data from a sheet
// Returns the data as a 2D array of interfaces and any error encountered
func (excel *Excel) ReadSheet(name string) ([][]interface{}, error) {
	// Check if sheet exists
	if idx, _ := excel.GetSheetIndex(name); idx == -1 {
		return nil, fmt.Errorf("sheet %s does not exist", name)
	}

	rows, err := excel.GetRows(name)
	if err != nil {
		return nil, err
	}

	// Convert [][]string to [][]interface{}
	result := make([][]interface{}, len(rows))
	for i, row := range rows {
		result[i] = make([]interface{}, len(row))
		for j, cell := range row {
			result[i][j] = cell
		}
	}
	return result, nil
}

// UpdateSheet updates an existing sheet with new data
// If the sheet doesn't exist, it will be created
func (excel *Excel) UpdateSheet(name string, data [][]interface{}) error {
	// Ensure sheet exists
	_, err := excel.SetSheet(name)
	if err != nil {
		return err
	}

	// Clear existing content by deleting the sheet
	err = excel.DeleteSheet(name)
	if err != nil {
		return err
	}

	// Create new sheet with same name
	_, err = excel.NewSheet(name)
	if err != nil {
		return err
	}

	// Write new data
	return excel.WriteAll(name, "A1", data)
}

// DeleteSheet removes a sheet by name
func (excel *Excel) DeleteSheet(name string) error {
	// Check if sheet exists
	if idx, _ := excel.GetSheetIndex(name); idx == -1 {
		return fmt.Errorf("sheet %s does not exist", name)
	}

	return excel.File.DeleteSheet(name)
}

// ListSheets returns a list of all sheet names in the workbook
func (excel *Excel) ListSheets() []string {
	return excel.GetSheetList()
}

// CopySheet copies a sheet to a new name
func (excel *Excel) CopySheet(source, destination string) error {
	// Check if source exists
	if idx, _ := excel.GetSheetIndex(source); idx == -1 {
		return fmt.Errorf("source sheet %s does not exist", source)
	}

	// Check if destination already exists
	if idx, _ := excel.GetSheetIndex(destination); idx != -1 {
		return fmt.Errorf("destination sheet %s already exists", destination)
	}

	// Create new sheet
	_, err := excel.NewSheet(destination)
	if err != nil {
		return err
	}

	// Copy content
	rows, err := excel.GetRows(source)
	if err != nil {
		return err
	}

	// Convert [][]string to [][]interface{}
	data := make([][]interface{}, len(rows))
	for i, row := range rows {
		data[i] = make([]interface{}, len(row))
		for j, cell := range row {
			data[i][j] = cell
		}
	}

	return excel.WriteAll(destination, "A1", data)
}
