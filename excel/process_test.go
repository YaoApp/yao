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
	p, err := process.Of("excel.open", newFile, true)
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
