package excel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestOpenClose(t *testing.T) {
	files := testFiles(t)

	h1, err := Open(files["test-01"], false)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := openFiles.Load(h1); !ok {
		t.Fatal("open file failed")
	}

	h2, err := Open(files["test-02"], true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := openFiles.Load(h2); !ok {
		t.Fatal("open file failed")
	}

	h3, err := Open(files["test-03"], false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := openFiles.Load(h3); !ok {
		t.Fatal("open file failed")
	}

	_, err = Open(files["test-04"], false)
	assert.Error(t, err)

	err = Close(h1)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := openFiles.Load(h1); ok {
		t.Fatal("close file failed")
	}
}

func TestGetSheetList(t *testing.T) {

	files := testFiles(t)
	h1, err := Open(files["test-01"], false)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(h1)

	xls, err := Get(h1)
	if err != nil {
		t.Fatal(err)
	}

	sheets := xls.GetSheetList()
	assert.Equal(t, []string{"供销存管理表格", "使用说明"}, sheets)

	_, err = Get("not found")
	assert.Error(t, err)
}

func TestOpenInvalidFile(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create an invalid excel file in the data root
	root := "excel"
	invalidFile := filepath.Join(root, "invalid.xlsx")

	// Ensure cleanup after test
	defer func() {
		if err := os.Remove(filepath.Join(config.Conf.DataRoot, invalidFile)); err != nil {
			t.Logf("Failed to cleanup test file: %v", err)
		}
	}()

	err := os.WriteFile(filepath.Join(config.Conf.DataRoot, invalidFile), []byte("invalid content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Open(invalidFile, false)
	assert.Error(t, err, "should fail to open invalid excel file")
}

func TestCloseErrors(t *testing.T) {
	// Test closing non-existent handler
	err := Close("non-existent-handler")
	assert.Error(t, err, "should fail to close non-existent file")

	// Test double close
	files := testFiles(t)
	h1, err := Open(files["test-01"], false)
	if err != nil {
		t.Fatal(err)
	}

	// First close
	err = Close(h1)
	assert.NoError(t, err)

	// Second close should fail
	err = Close(h1)
	assert.Error(t, err, "should fail on second close")
}

func TestOpenWithInvalidPath(t *testing.T) {
	// Test with invalid path
	_, err := Open("../invalid/path/file.xlsx", false)
	assert.Error(t, err, "should fail with invalid path")

	// Test with path trying to escape data root
	_, err = Open("../../../../etc/file.xlsx", false)
	assert.Error(t, err, "should fail with path trying to escape data root")
}

func TestWrite(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create a new writable file for testing
	root := "excel"
	testFile := filepath.Join(root, "write-test.xlsx")

	// Ensure cleanup after test
	defer func() {
		if err := os.Remove(filepath.Join(config.Conf.DataRoot, testFile)); err != nil {
			t.Logf("Failed to cleanup test file: %v", err)
		}
	}()

	h1, err := Open(testFile, true)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(h1)

	xls, err := Get(h1)
	if err != nil {
		t.Fatal(err)
	}

	// Test WriteCell
	err = xls.WriteCell("Sheet1", "A1", "Hello")
	assert.NoError(t, err)
	err = xls.WriteCell("Sheet1", "B1", 123)
	assert.NoError(t, err)
	err = xls.WriteCell("Sheet1", "C1", true)
	assert.NoError(t, err)

	// Test WriteRow
	row := []interface{}{"Row1", 456, false}
	err = xls.WriteRow("Sheet1", "A2", row)
	assert.NoError(t, err)

	// Test WriteColumn
	col := []interface{}{"Col1", 789, true}
	err = xls.WriteColumn("Sheet1", "D1", col)
	assert.NoError(t, err)

	// Test WriteAll
	data := [][]interface{}{
		{"Name", "Age", "City"},
		{"John", 30, "New York"},
		{"Alice", 25, "London"},
	}
	err = xls.WriteAll("Sheet2", "A1", data)
	assert.NoError(t, err)

	// Test error cases
	// Invalid cell reference
	err = xls.WriteCell("Sheet1", "invalid", "test")
	assert.Error(t, err)

	// Save the file to verify changes
	err = xls.SaveAs(filepath.Join(config.Conf.DataRoot, testFile))
	assert.NoError(t, err)

	// Verify written data
	val, err := xls.GetCellValue("Sheet1", "A1")
	assert.NoError(t, err)
	assert.Equal(t, "Hello", val)

	val, err = xls.GetCellValue("Sheet2", "A1")
	assert.NoError(t, err)
	assert.Equal(t, "Name", val)
}

func TestSetSheet(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	root := "excel"
	testFile := filepath.Join(root, "sheet-test.xlsx")

	// Ensure cleanup after test
	defer func() {
		if err := os.Remove(filepath.Join(config.Conf.DataRoot, testFile)); err != nil {
			t.Logf("Failed to cleanup test file: %v", err)
		}
	}()

	h1, err := Open(testFile, true)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(h1)

	xls, err := Get(h1)
	if err != nil {
		t.Fatal(err)
	}

	// Test creating new sheet
	idx, err := xls.SetSheet("NewSheet")
	assert.NoError(t, err)
	assert.Greater(t, idx, 0)

	// Test getting existing sheet
	idx2, err := xls.SetSheet("NewSheet")
	assert.NoError(t, err)
	assert.Equal(t, idx, idx2)

	// Verify sheet exists
	sheets := xls.GetSheetList()
	assert.Contains(t, sheets, "NewSheet")
}

func testFiles(t *testing.T) map[string]string {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// test data root path
	root := "excel"
	return map[string]string{
		"test-01": filepath.Join(root, "test-01.xlsx"),
		"test-02": filepath.Join(root, "test-02.xlsx"),
		"test-03": filepath.Join(root, "test-03.xlsx"),
	}
}
