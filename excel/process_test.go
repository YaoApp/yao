package excel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessOpen(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Test opening file in read-only mode
	p, err := process.Of("excel.open", files["test-01"], true)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, handle)

	// Test opening file in write mode
	p, err = process.Of("excel.open", files["test-01"], false)
	if err != nil {
		t.Fatal(err)
	}

	handle2, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, handle2)
}

func TestProcessSheets(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Open file first
	p, err := process.Of("excel.open", files["test-01"], true)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Get sheets
	p, err = process.Of("excel.sheets", handle)
	if err != nil {
		t.Fatal(err)
	}

	sheets, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	sheetList := sheets.([]string)
	assert.Equal(t, []string{"供销存管理表格", "使用说明"}, sheetList)
}

func TestProcessReadCell(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Open file first
	p, err := process.Of("excel.open", files["test-01"], true)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Get sheets first to verify we have the right sheet
	p, err = process.Of("excel.sheets", handle)
	if err != nil {
		t.Fatal(err)
	}

	sheets, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	sheetList := sheets.([]string)
	if len(sheetList) == 0 {
		t.Fatal("no sheets found")
	}

	// Read cell from the first sheet
	p, err = process.Of("excel.read.cell", handle, sheetList[0], "B2")
	if err != nil {
		t.Fatal(err)
	}

	value, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Print the value for debugging
	assert.NotEmpty(t, value)

	// Try reading from a non-existent cell
	p, err = process.Of("excel.read.cell", handle, sheetList[0], "ZZ999")
	if err != nil {
		t.Fatal(err)
	}

	value, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, value) // Non-existent cell should return empty string
}

func TestProcessClose(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Open file first
	p, err := process.Of("excel.open", files["test-01"], true)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Close file
	p, err = process.Of("excel.close", handle)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Try to use closed handle
	p, err = process.Of("excel.sheets", handle)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.Error(t, err)
}

func TestProcessConvert(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Test ColumnNameToNumber
	p, err := process.Of("excel.convert.columnnametonumber", "AK")
	if err != nil {
		t.Fatal(err)
	}

	number, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 37, number)

	// Test ColumnNumberToName
	p, err = process.Of("excel.convert.columnnumbertoname", 37)
	if err != nil {
		t.Fatal(err)
	}

	name, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "AK", name)

	// Test CellNameToCoordinates
	p, err = process.Of("excel.convert.cellnametocoordinates", "A1")
	if err != nil {
		t.Fatal(err)
	}

	coords, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	coordsArr := coords.([]int)
	assert.Equal(t, []int{1, 1}, coordsArr)

	// Test CoordinatesToCellName
	p, err = process.Of("excel.convert.coordinatestocellname", 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	cell, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "A1", cell)
}

func TestProcessSave(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Create a new test file path
	dataRoot := config.Conf.DataRoot
	newFile := filepath.Join(filepath.Dir(files["test-01"]), "test-save.xlsx")

	// Copy test-01.xlsx to new file
	content, err := os.ReadFile(filepath.Join(dataRoot, files["test-01"]))
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dataRoot, newFile), content, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filepath.Join(dataRoot, newFile)) // Clean up after test

	// Open new file in write mode
	p, err := process.Of("excel.open", newFile, false)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Write something to the file
	p, err = process.Of("excel.write.cell", handle, "供销存管理表格", "A1", "Test Save")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Save the file
	p, err = process.Of("excel.save", handle)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Close the file before reading
	p, err = process.Of("excel.close", handle)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Verify the file exists and has been modified
	savedContent, err := os.ReadFile(filepath.Join(dataRoot, newFile))
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEqual(t, content, savedContent, "New file should be modified")

	// Verify original file is unchanged
	originalContent, err := os.ReadFile(filepath.Join(dataRoot, files["test-01"]))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, content, originalContent, "Original file should not be modified")

	// Try to save with invalid handle
	p, err = process.Of("excel.save", "invalid-handle")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.Error(t, err)
}

func TestProcessWriteCell(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Open file in write mode
	p, err := process.Of("excel.open", files["test-01"], false)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Write string value
	p, err = process.Of("excel.write.cell", handle, "供销存管理表格", "A1", "Test Write")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Write number value
	p, err = process.Of("excel.write.cell", handle, "供销存管理表格", "B1", 123.45)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Verify written values
	p, err = process.Of("excel.read.cell", handle, "供销存管理表格", "A1")
	if err != nil {
		t.Fatal(err)
	}

	value, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Test Write", value)

	// Test with invalid handle
	p, err = process.Of("excel.write.cell", "invalid-handle", "供销存管理表格", "A1", "Test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.Error(t, err)

	// Test with invalid sheet name
	p, err = process.Of("excel.write.cell", handle, "InvalidSheet", "A1", "Test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.Error(t, err)
}

func TestProcessSetStyle(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Open file in write mode
	p, err := process.Of("excel.open", files["test-01"], false)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Set style
	p, err = process.Of("excel.set.style", handle, "供销存管理表格", "A1", 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Test with invalid handle
	p, err = process.Of("excel.set.style", "invalid-handle", "供销存管理表格", "A1", 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.Error(t, err)

	// Test with invalid sheet name
	p, err = process.Of("excel.set.style", handle, "InvalidSheet", "A1", 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.Error(t, err)

	// Test with invalid style ID
	p, err = process.Of("excel.set.style", handle, "供销存管理表格", "A1", -1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.Error(t, err)
}

func TestProcessReadRow(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Open file first
	p, err := process.Of("excel.open", files["test-01"], true)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Read rows from the first sheet
	p, err = process.Of("excel.read.row", handle, "供销存管理表格")
	if err != nil {
		t.Fatal(err)
	}

	rows, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Verify we got some rows
	assert.NotNil(t, rows)
	rowsData := rows.([][]string)
	assert.True(t, len(rowsData) > 0, "Should have at least one row")
}

func TestProcessReadColumn(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Open file first
	p, err := process.Of("excel.open", files["test-01"], true)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Read columns from the first sheet
	p, err = process.Of("excel.read.column", handle, "供销存管理表格")
	if err != nil {
		t.Fatal(err)
	}

	cols, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Verify we got some columns
	assert.NotNil(t, cols)
	colsData := cols.([][]string)
	assert.True(t, len(colsData) > 0, "Should have at least one column")
}

func TestProcessWriteOperations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Create a new test file path
	dataRoot := config.Conf.DataRoot
	newFile := filepath.Join(filepath.Dir(files["test-01"]), "test-write-ops.xlsx")

	// Copy test-01.xlsx to new file
	content, err := os.ReadFile(filepath.Join(dataRoot, files["test-01"]))
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dataRoot, newFile), content, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filepath.Join(dataRoot, newFile)) // Clean up after test

	// Open file in write mode
	p, err := process.Of("excel.open", newFile, false)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Test write row
	p, err = process.Of("excel.write.row", handle, "供销存管理表格", "A1", []interface{}{"Test1", "Test2", "Test3"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Test write column
	p, err = process.Of("excel.write.column", handle, "供销存管理表格", "B1", []interface{}{"Col1", "Col2", "Col3"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Test write all
	p, err = process.Of("excel.write.all", handle, "供销存管理表格", "C1", [][]interface{}{
		{"All1", "All2", "All3"},
		{"All4", "All5", "All6"},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Save and close
	p, err = process.Of("excel.save", handle)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	p, err = process.Of("excel.close", handle)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)
}

func TestProcessSetOptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Create a new test file path
	dataRoot := config.Conf.DataRoot
	newFile := filepath.Join(filepath.Dir(files["test-01"]), "test-set-ops.xlsx")

	// Copy test-01.xlsx to new file
	content, err := os.ReadFile(filepath.Join(dataRoot, files["test-01"]))
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dataRoot, newFile), content, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filepath.Join(dataRoot, newFile)) // Clean up after test

	// Open file in write mode
	p, err := process.Of("excel.open", newFile, false)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Test row height
	p, err = process.Of("excel.set.rowheight", handle, "供销存管理表格", 1, 30)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Test column width
	p, err = process.Of("excel.set.columnwidth", handle, "供销存管理表格", "A", "B", 20)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Test merge cells
	p, err = process.Of("excel.set.mergecell", handle, "供销存管理表格", "C3", "D4")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Test formula
	p, err = process.Of("excel.set.formula", handle, "供销存管理表格", "E5", "SUM(A1:A4)")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Save and close
	p, err = process.Of("excel.save", handle)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	p, err = process.Of("excel.close", handle)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)
}

func TestProcessIterators(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Open file first
	p, err := process.Of("excel.open", files["test-01"], true)
	if err != nil {
		t.Fatal(err)
	}

	handle, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Test row iterator
	p, err = process.Of("excel.each.openrow", handle, "供销存管理表格")
	if err != nil {
		t.Fatal(err)
	}

	rowID, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, rowID)

	// Get first row
	p, err = process.Of("excel.each.nextrow", rowID)
	if err != nil {
		t.Fatal(err)
	}

	row, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// May be nil if empty sheet, but shouldn't error
	if row != nil {
		assert.IsType(t, []string{}, row)
	}

	// Close row iterator
	p, err = process.Of("excel.each.closerow", rowID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)

	// Test column iterator
	p, err = process.Of("excel.each.opencolumn", handle, "供销存管理表格")
	if err != nil {
		t.Fatal(err)
	}

	colID, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, colID)

	// Get first column
	p, err = process.Of("excel.each.nextcolumn", colID)
	if err != nil {
		t.Fatal(err)
	}

	col, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// May be nil if empty sheet, but shouldn't error
	if col != nil {
		assert.IsType(t, []string{}, col)
	}

	// Close column iterator
	p, err = process.Of("excel.each.closecolumn", colID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NoError(t, err)
}
