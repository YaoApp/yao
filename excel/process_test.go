package excel

import (
	"fmt"
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

	// Create test data
	testData := make([]interface{}, 0)
	testData = append(testData, []interface{}{"Header1", "Header2", "Header3"})
	testData = append(testData, []interface{}{1, "Row1", true})
	testData = append(testData, []interface{}{2, "Row2", false})
	testData = append(testData, []interface{}{3, "Row3", true})
	testData = append(testData, []interface{}{4, "Row4", false})
	testData = append(testData, []interface{}{5, "Row5", true})
	testData = append(testData, []interface{}{6, "Row6", false})
	testData = append(testData, []interface{}{7, "Row7", true})
	testData = append(testData, []interface{}{8, "Row8", false})
	testData = append(testData, []interface{}{9, "Row9", true})

	// Create a new sheet and write test data
	p, err = process.Of("excel.sheet.create", handle, "RowTestSheet")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	p, err = process.Of("excel.write.all", handle, "RowTestSheet", "A1", testData)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

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

func TestProcessSheetOperations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Create a new test file path
	dataRoot := config.Conf.DataRoot
	newFile := filepath.Join(filepath.Dir(files["test-01"]), "test-sheet-ops.xlsx")

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

	// Test sheet.create
	t.Run("CreateSheet", func(t *testing.T) {
		p, err := process.Of("excel.sheet.create", handle, "TestSheet1")
		assert.NoError(t, err)

		idx, err := p.Exec()
		assert.NoError(t, err)
		assert.Greater(t, idx.(int), 0)

		// Try to create a sheet with the same name (should fail)
		p, err = process.Of("excel.sheet.create", handle, "TestSheet1")
		assert.NoError(t, err)

		_, err = p.Exec()
		assert.Error(t, err)
	})

	// Test sheet.list
	t.Run("ListSheets", func(t *testing.T) {
		p, err := process.Of("excel.sheet.list", handle)
		assert.NoError(t, err)

		sheets, err := p.Exec()
		assert.NoError(t, err)
		sheetList := sheets.([]string)
		assert.Contains(t, sheetList, "TestSheet1")
	})

	// Test sheet.update and sheet.read
	t.Run("UpdateAndReadSheet", func(t *testing.T) {
		testData := [][]interface{}{
			{"Header1", "Header2"},
			{1, "Data1"},
			{2, "Data2"},
		}

		p, err := process.Of("excel.sheet.update", handle, "TestSheet1", testData)
		assert.NoError(t, err)

		_, err = p.Exec()
		assert.NoError(t, err)

		// Read and verify
		p, err = process.Of("excel.sheet.read", handle, "TestSheet1")
		assert.NoError(t, err)

		data, err := p.Exec()
		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Try to read non-existent sheet
		p, err = process.Of("excel.sheet.read", handle, "NonExistentSheet")
		assert.NoError(t, err)

		_, err = p.Exec()
		assert.Error(t, err)
	})

	// Test sheet.copy
	t.Run("CopySheet", func(t *testing.T) {
		p, err := process.Of("excel.sheet.copy", handle, "TestSheet1", "CopiedSheet")
		assert.NoError(t, err)

		_, err = p.Exec()
		assert.NoError(t, err)

		// Verify the copy exists
		p, err = process.Of("excel.sheet.list", handle)
		assert.NoError(t, err)

		sheets, err := p.Exec()
		assert.NoError(t, err)
		sheetList := sheets.([]string)
		assert.Contains(t, sheetList, "CopiedSheet")

		// Try to copy to existing sheet name (should fail)
		p, err = process.Of("excel.sheet.copy", handle, "TestSheet1", "CopiedSheet")
		assert.NoError(t, err)

		_, err = p.Exec()
		assert.Error(t, err)
	})

	// Test sheet.delete
	t.Run("DeleteSheet", func(t *testing.T) {
		p, err := process.Of("excel.sheet.delete", handle, "CopiedSheet")
		assert.NoError(t, err)

		_, err = p.Exec()
		assert.NoError(t, err)

		// Verify the sheet is deleted
		p, err = process.Of("excel.sheet.list", handle)
		assert.NoError(t, err)

		sheets, err := p.Exec()
		assert.NoError(t, err)
		sheetList := sheets.([]string)
		assert.NotContains(t, sheetList, "CopiedSheet")

		// Try to delete non-existent sheet
		p, err = process.Of("excel.sheet.delete", handle, "NonExistentSheet")
		assert.NoError(t, err)

		_, err = p.Exec()
		assert.Error(t, err)
	})

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

func TestProcessReadSheetRows(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Create a new test file path
	dataRoot := config.Conf.DataRoot
	newFile := filepath.Join(filepath.Dir(files["test-01"]), "test-read-rows.xlsx")

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

	// Create test data
	testData := make([]interface{}, 0)
	testData = append(testData, []interface{}{"Header1", "Header2", "Header3"})
	testData = append(testData, []interface{}{1, "Row1", true})
	testData = append(testData, []interface{}{2, "Row2", false})
	testData = append(testData, []interface{}{3, "Row3", true})
	testData = append(testData, []interface{}{4, "Row4", false})
	testData = append(testData, []interface{}{5, "Row5", true})
	testData = append(testData, []interface{}{6, "Row6", false})
	testData = append(testData, []interface{}{7, "Row7", true})
	testData = append(testData, []interface{}{8, "Row8", false})
	testData = append(testData, []interface{}{9, "Row9", true})

	// Create a new sheet and write test data
	p, err = process.Of("excel.sheet.create", handle, "RowTestSheet")
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	p, err = process.Of("excel.write.all", handle, "RowTestSheet", "A1", testData)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Save and close the file
	p, err = process.Of("excel.save", handle)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	p, err = process.Of("excel.close", handle)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Reopen the file for reading
	p, err = process.Of("excel.open", newFile, true)
	if err != nil {
		t.Fatal(err)
	}
	handle, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Test cases for reading rows
	t.Run("ReadFromMiddle", func(t *testing.T) {
		p, err := process.Of("excel.sheet.rows", handle, "RowTestSheet", 2, 4)
		assert.NoError(t, err)
		data, err := p.Exec()
		assert.NoError(t, err)
		rows := data.([][]string)
		assert.Equal(t, 4, len(rows))
		assert.Equal(t, "2", rows[0][0]) // First row should be row 2
	})

	t.Run("ReadFromBeginning", func(t *testing.T) {
		p, err := process.Of("excel.sheet.rows", handle, "RowTestSheet", 0, 3)
		assert.NoError(t, err)
		data, err := p.Exec()
		assert.NoError(t, err)
		rows := data.([][]string)
		assert.Equal(t, 3, len(rows))
		assert.Equal(t, "Header1", rows[0][0]) // First row should be header
	})

	t.Run("ReadBeyondAvailable", func(t *testing.T) {
		p, err := process.Of("excel.sheet.rows", handle, "RowTestSheet", 8, 5)
		assert.NoError(t, err)
		data, err := p.Exec()
		assert.NoError(t, err)
		rows := data.([][]string)
		assert.Equal(t, 2, len(rows)) // Only 2 rows remain
	})

	t.Run("ReadNonExistentSheet", func(t *testing.T) {
		p, err := process.Of("excel.sheet.rows", handle, "NonExistentSheet", 0, 5)
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.Error(t, err)
	})

	t.Run("ReadWithSizeZero", func(t *testing.T) {
		p, err := process.Of("excel.sheet.rows", handle, "RowTestSheet", 0, 0)
		assert.NoError(t, err)
		data, err := p.Exec()
		assert.NoError(t, err)
		rows := data.([][]string)
		assert.Equal(t, 0, len(rows))
	})

	t.Run("ReadWithNegativeStart", func(t *testing.T) {
		p, err := process.Of("excel.sheet.rows", handle, "RowTestSheet", -1, 5)
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.Error(t, err)
	})

	t.Run("ReadWithNegativeSize", func(t *testing.T) {
		p, err := process.Of("excel.sheet.rows", handle, "RowTestSheet", 0, -1)
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.Error(t, err)
	})

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

func TestProcessGetSheetDimension(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	files := testFiles(t)

	// Create a new test file path
	dataRoot := config.Conf.DataRoot
	newFile := filepath.Join(filepath.Dir(files["test-01"]), "test-dimension.xlsx")

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

	// Test 1: Empty sheet
	t.Run("EmptySheet", func(t *testing.T) {
		// Create empty sheet
		p, err := process.Of("excel.sheet.create", handle, "EmptySheet")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.NoError(t, err)

		// Get dimensions
		p, err = process.Of("excel.sheet.dimension", handle, "EmptySheet")
		assert.NoError(t, err)
		dim, err := p.Exec()
		assert.NoError(t, err)

		// Verify dimensions
		dimMap := dim.(map[string]int)
		assert.Equal(t, 0, dimMap["rows"])
		assert.Equal(t, 0, dimMap["cols"])
	})

	// Test 2: Sheet with data
	t.Run("SheetWithData", func(t *testing.T) {

		// Create test data
		testData := make([]interface{}, 0)
		testData = append(testData, []interface{}{"A1", "B1", "C1"})
		testData = append(testData, []interface{}{"A2", "B2", "C2"})
		testData = append(testData, []interface{}{"A3", "B3", "C3"})

		// Create and write to sheet
		p, err := process.Of("excel.sheet.create", handle, "DataSheet")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.NoError(t, err)

		p, err = process.Of("excel.write.all", handle, "DataSheet", "A1", testData)
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.NoError(t, err)

		// Save to ensure dimensions are updated
		p, err = process.Of("excel.save", handle)
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.NoError(t, err)

		// Get dimensions
		p, err = process.Of("excel.sheet.dimension", handle, "DataSheet")
		assert.NoError(t, err)
		dim, err := p.Exec()
		assert.NoError(t, err)

		// Verify dimensions
		dimMap := dim.(map[string]int)
		assert.Equal(t, 3, dimMap["rows"])
		assert.Equal(t, 3, dimMap["cols"])
	})

	// Test 3: Non-existent sheet
	t.Run("NonExistentSheet", func(t *testing.T) {
		p, err := process.Of("excel.sheet.dimension", handle, "NonExistentSheet")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.Error(t, err)
	})

	// Test 4: Large sheet
	t.Run("LargeSheet", func(t *testing.T) {
		// Create large test data (100x50)
		largeData := make([]interface{}, 0)
		for i := 0; i < 100; i++ {
			row := make([]interface{}, 50)
			for j := 0; j < 50; j++ {
				row[j] = fmt.Sprintf("Cell_%d_%d", i, j)
			}
			largeData = append(largeData, row)
		}

		// Create and write to sheet
		p, err := process.Of("excel.sheet.create", handle, "LargeSheet")
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.NoError(t, err)

		p, err = process.Of("excel.write.all", handle, "LargeSheet", "A1", largeData)
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.NoError(t, err)

		// Save to ensure dimensions are updated
		p, err = process.Of("excel.save", handle)
		assert.NoError(t, err)
		_, err = p.Exec()
		assert.NoError(t, err)

		// Get dimensions
		p, err = process.Of("excel.sheet.dimension", handle, "LargeSheet")
		assert.NoError(t, err)
		dim, err := p.Exec()
		assert.NoError(t, err)

		// Verify dimensions
		dimMap := dim.(map[string]int)
		assert.Equal(t, 100, dimMap["rows"])
		assert.Equal(t, 50, dimMap["cols"])
	})

	// Clean up
	p, err = process.Of("excel.close", handle)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec()
	assert.NoError(t, err)
}
