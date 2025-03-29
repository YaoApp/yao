package excel

import (
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
}
