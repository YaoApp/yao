package excel

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSheetOperations(t *testing.T) {
	// Get test files and open the test file
	files := testFiles(t)
	handler, err := Open(files["test-01"], true) // Open in writable mode
	if err != nil {
		t.Fatal(err)
	}
	defer Close(handler)

	excel, err := Get(handler)
	if err != nil {
		t.Fatal(err)
	}

	// Test CreateSheet
	t.Run("CreateSheet", func(t *testing.T) {
		// Create a new sheet
		idx, err := excel.CreateSheet("TestSheet1")
		assert.NoError(t, err)
		assert.Greater(t, idx, 0)

		// Try to create a sheet with the same name (should fail)
		_, err = excel.CreateSheet("TestSheet1")
		assert.Error(t, err)
	})

	// Test ReadSheet
	t.Run("ReadSheet", func(t *testing.T) {
		// Create test data
		testData := [][]interface{}{
			{"Header1", "Header2"},
			{1, "Data1"},
			{2, "Data2"},
		}

		// Write test data
		err := excel.WriteAll("TestSheet1", "A1", testData)
		assert.NoError(t, err)

		// Read the data back
		data, err := excel.ReadSheet("TestSheet1")
		assert.NoError(t, err)
		assert.Equal(t, len(testData), len(data))

		// Try to read non-existent sheet
		_, err = excel.ReadSheet("NonExistentSheet")
		assert.Error(t, err)
	})

	// Test UpdateSheet
	t.Run("UpdateSheet", func(t *testing.T) {
		newData := [][]interface{}{
			{"NewHeader1", "NewHeader2"},
			{3, "NewData1"},
			{4, "NewData2"},
		}

		// Update existing sheet
		err := excel.UpdateSheet("TestSheet1", newData)
		assert.NoError(t, err)

		// Read back and verify
		data, err := excel.ReadSheet("TestSheet1")
		assert.NoError(t, err)
		assert.Equal(t, len(newData), len(data))

		// Update non-existent sheet (should create new)
		err = excel.UpdateSheet("NewSheet", newData)
		assert.NoError(t, err)
	})

	// Test ListSheets
	t.Run("ListSheets", func(t *testing.T) {
		sheets := excel.ListSheets()
		assert.Contains(t, sheets, "TestSheet1")
		assert.Contains(t, sheets, "NewSheet")
	})

	// Test SheetExists
	t.Run("SheetExists", func(t *testing.T) {
		// Check existing sheet
		exists := excel.SheetExists("TestSheet1")
		assert.True(t, exists)

		// Check non-existent sheet
		exists = excel.SheetExists("NonExistentSheet")
		assert.False(t, exists)
	})

	// Test CopySheet
	t.Run("CopySheet", func(t *testing.T) {
		// Copy existing sheet
		err := excel.CopySheet("TestSheet1", "CopiedSheet")
		assert.NoError(t, err)

		// Verify the copy
		originalData, err := excel.ReadSheet("TestSheet1")
		assert.NoError(t, err)
		copiedData, err := excel.ReadSheet("CopiedSheet")
		assert.NoError(t, err)
		assert.Equal(t, originalData, copiedData)

		// Try to copy to existing sheet name (should fail)
		err = excel.CopySheet("TestSheet1", "CopiedSheet")
		assert.Error(t, err)

		// Try to copy non-existent sheet (should fail)
		err = excel.CopySheet("NonExistentSheet", "NewSheet2")
		assert.Error(t, err)
	})

	// Test DeleteSheet
	t.Run("DeleteSheet", func(t *testing.T) {
		// Delete existing sheet
		err := excel.DeleteSheet("CopiedSheet")
		assert.NoError(t, err)

		// Verify sheet is deleted
		sheets := excel.ListSheets()
		assert.NotContains(t, sheets, "CopiedSheet")

		// Try to delete non-existent sheet
		err = excel.DeleteSheet("NonExistentSheet")
		assert.Error(t, err)
	})

	// Test ReadSheetRows
	t.Run("ReadSheetRows", func(t *testing.T) {
		// Create test data with 10 rows
		testData := [][]interface{}{
			{"Header1", "Header2", "Header3"},
			{1, "Row1", true},
			{2, "Row2", false},
			{3, "Row3", true},
			{4, "Row4", false},
			{5, "Row5", true},
			{6, "Row6", false},
			{7, "Row7", true},
			{8, "Row8", false},
			{9, "Row9", true},
		}

		// Expected string data
		expectedData := [][]string{
			{"Header1", "Header2", "Header3"},
			{"1", "Row1", "TRUE"},
			{"2", "Row2", "FALSE"},
			{"3", "Row3", "TRUE"},
			{"4", "Row4", "FALSE"},
			{"5", "Row5", "TRUE"},
			{"6", "Row6", "FALSE"},
			{"7", "Row7", "TRUE"},
			{"8", "Row8", "FALSE"},
			{"9", "Row9", "TRUE"},
		}

		// Create a new sheet for testing
		_, err := excel.CreateSheet("RowTestSheet")
		assert.NoError(t, err)

		// Write test data
		err = excel.WriteAll("RowTestSheet", "A1", testData)
		assert.NoError(t, err)

		// Test 1: Read from middle (start at row 2, read 4 rows)
		data, err := excel.ReadSheetRows("RowTestSheet", 2, 4)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(data))
		assert.Equal(t, expectedData[2:6], data)

		// Test 2: Read from beginning (start at row 0, read 3 rows)
		data, err = excel.ReadSheetRows("RowTestSheet", 0, 3)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(data))
		assert.Equal(t, expectedData[0:3], data)

		// Test 3: Read beyond available rows (should return remaining rows)
		data, err = excel.ReadSheetRows("RowTestSheet", 8, 5)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(data)) // Only 2 rows remain
		assert.Equal(t, expectedData[8:], data)

		// Test 4: Read from non-existent sheet
		_, err = excel.ReadSheetRows("NonExistentSheet", 0, 5)
		assert.Error(t, err)

		// Test 5: Read with size 0 (should return empty slice)
		data, err = excel.ReadSheetRows("RowTestSheet", 0, 0)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(data))

		// Test 6: Read with negative start (should return error)
		_, err = excel.ReadSheetRows("RowTestSheet", -1, 5)
		assert.Error(t, err)

		// Test 7: Read with negative size (should return error)
		_, err = excel.ReadSheetRows("RowTestSheet", 0, -1)
		assert.Error(t, err)
	})

	// Test GetSheetDimension
	t.Run("GetSheetDimension", TestGetSheetDimension)
}

// TestGetSheetDimension tests the GetSheetDimension function
func TestGetSheetDimension(t *testing.T) {
	// Get test files and open the test file
	files := testFiles(t)
	filename := filepath.Dir(files["test-01"]) + "/test-dimension.xlsx"
	handler, err := Open(filename, true) // Open in writable mode
	if err != nil {
		t.Fatal(err)
	}
	defer Close(handler)

	excel, err := Get(handler)
	if err != nil {
		t.Fatal(err)
	}

	// Clean up existing sheets
	sheets := excel.ListSheets()
	for _, sheet := range sheets {
		if sheet != "Sheet1" { // Keep the default sheet
			err = excel.DeleteSheet(sheet)
			assert.NoError(t, err)
		}
	}

	// Test 1: Create a large sheet (100x100)
	_, err = excel.CreateSheet("LargeSheet")
	assert.NoError(t, err)

	// Create test data (100x100)
	largeData := make([][]interface{}, 100)
	for i := 0; i < 100; i++ {
		largeData[i] = make([]interface{}, 100)
		for j := 0; j < 100; j++ {
			largeData[i][j] = fmt.Sprintf("Cell_%d_%d", i, j)
		}
	}
	err = excel.WriteAll("LargeSheet", "A1", largeData)
	assert.NoError(t, err)

	// Save file to ensure dimensions are updated
	err = excel.Save()
	assert.NoError(t, err)

	// Test large sheet dimensions
	rows, cols, err := excel.GetSheetDimension("LargeSheet")
	assert.NoError(t, err)
	assert.Equal(t, 100, rows)
	assert.Equal(t, 100, cols)

	// Test 2: Empty sheet
	if excel.SheetExists("EmptySheet") {
		excel.DeleteSheet("EmptySheet")
	}
	_, err = excel.CreateSheet("EmptySheet")
	assert.NoError(t, err)
	rows, cols, err = excel.GetSheetDimension("EmptySheet")
	assert.NoError(t, err)
	assert.Equal(t, 0, rows)
	assert.Equal(t, 0, cols)

	// Test 3: Regular sheet with data
	testData := [][]interface{}{
		{"A1", "B1", "C1"},
		{"A2", "B2", "C2"},
		{"A3", "B3", "C3"},
	}
	err = excel.WriteAll("RegularSheet", "A1", testData)
	assert.NoError(t, err)

	// Save file to ensure dimensions are updated
	err = excel.Save()
	assert.NoError(t, err)

	rows, cols, err = excel.GetSheetDimension("RegularSheet")
	assert.NoError(t, err)
	assert.Equal(t, 3, rows)
	assert.Equal(t, 3, cols)

	// Test 4: Non-existent sheet
	rows, cols, err = excel.GetSheetDimension("NonExistentSheet")
	assert.Error(t, err)
	assert.Equal(t, 0, rows)
	assert.Equal(t, 0, cols)
}
