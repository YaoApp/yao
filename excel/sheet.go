package excel

import (
	"fmt"
	"strings"

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

// validateSheetName checks if the sheet name contains invalid characters
func (excel *Excel) validateSheetName(name string) error {
	invalidChars := []string{":", "\\", "/", "?", "*", "[", "]"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("sheet name cannot contain any of these characters: :/?*[\\]")
		}
	}
	if len(name) == 0 {
		return fmt.Errorf("sheet name cannot be empty")
	}
	if len(name) > 31 {
		return fmt.Errorf("sheet name cannot be longer than 31 characters")
	}
	return nil
}

// CreateSheet creates a new sheet with the given name
// Returns the index of the new sheet and any error encountered
func (excel *Excel) CreateSheet(name string) (int, error) {
	// Validate sheet name
	if err := excel.validateSheetName(name); err != nil {
		return 0, err
	}

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

// GetSheetDimension returns the number of rows and columns in a sheet
func (excel *Excel) GetSheetDimension(name string) (rows int, cols int, err error) {
	// Check if sheet exists
	if idx, _ := excel.GetSheetIndex(name); idx == -1 {
		return 0, 0, fmt.Errorf("sheet %s does not exist", name)
	}
	rows = 0
	cols = 0
	ri, err := excel.File.Rows(name)
	if err != nil {
		return 0, 0, err
	}
	defer ri.Close()
	for ri.Next() {
		rows++
	}

	// Get column count
	ci, err := excel.File.Cols(name)
	if err != nil {
		return 0, 0, err
	}
	for ci.Next() {
		cols++
	}
	return rows, cols, nil

}

// ReadSheetRows reads all data from a sheet by rows
func (excel *Excel) ReadSheetRows(name string, start int, size int) ([][]string, error) {
	// Validate parameters
	if start < 0 {
		return nil, fmt.Errorf("start position cannot be negative")
	}
	if size < 0 {
		return nil, fmt.Errorf("size cannot be negative")
	}

	// Check if sheet exists
	if idx, _ := excel.GetSheetIndex(name); idx == -1 {
		return nil, fmt.Errorf("sheet %s does not exist", name)
	}

	// If size is 0, return empty slice
	if size == 0 {
		return [][]string{}, nil
	}

	// Get rows iterator
	rows, err := excel.File.Rows(name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Skip to start position
	currentRow := -1
	for rows.Next() {
		currentRow++
		if currentRow >= start {
			break
		}
	}

	// Read requested number of rows
	result := make([][]string, 0, size)
	if currentRow == start {
		row, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	for i := 1; i < size && rows.Next(); i++ {
		row, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, nil
}

// UpdateSheet updates an existing sheet with new data
// If the sheet doesn't exist, it will be created
func (excel *Excel) UpdateSheet(name string, data [][]interface{}) error {
	// Validate sheet name
	if err := excel.validateSheetName(name); err != nil {
		return err
	}

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

// SheetExists checks if a sheet exists in the workbook
func (excel *Excel) SheetExists(name string) bool {
	idx, _ := excel.GetSheetIndex(name)
	return idx != -1
}

// CopySheet copies a sheet to a new name
func (excel *Excel) CopySheet(source, destination string) error {
	// Validate destination sheet name
	if err := excel.validateSheetName(destination); err != nil {
		return err
	}

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
