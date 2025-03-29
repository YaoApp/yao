package excel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestWriteAll(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create a new Excel file
	xls := excelize.NewFile()
	defer func() {
		if err := xls.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Create Excel instance
	excel := &Excel{
		File: xls,
		abs:  "test.xlsx",
	}

	t.Run("Write to default sheet", func(t *testing.T) {
		data := [][]interface{}{
			{"Header1", "Header2", "Header3"},
			{1, "Data1", true},
			{2, "Data2", false},
		}

		err := excel.WriteAll("Sheet1", "A1", data)
		assert.NoError(t, err)

		// Verify data was written
		rows, err := excel.GetRows("Sheet1")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(rows), 3)
		assert.Equal(t, "Header1", rows[0][0])
		assert.Equal(t, "Header2", rows[0][1])
		assert.Equal(t, "Header3", rows[0][2])
	})

	t.Run("Write to new sheet", func(t *testing.T) {
		data := [][]interface{}{
			{"Name", "Age", "Active"},
			{"John", 30, true},
			{"Jane", 25, false},
		}

		// Verify sheet doesn't exist before writing
		sheets := excel.ListSheets()
		assert.NotContains(t, sheets, "NewSheet")

		err := excel.WriteAll("NewSheet", "B2", data)
		assert.NoError(t, err)

		// Verify sheet was created
		sheets = excel.ListSheets()
		assert.Contains(t, sheets, "NewSheet")

		// Verify data was written
		rows, err := excel.GetRows("NewSheet")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(rows), 4) // Account for B2 start position
		assert.Equal(t, "Name", rows[1][1])    // B2 position
		assert.Equal(t, "Age", rows[1][2])
		assert.Equal(t, "Active", rows[1][3])
	})

	t.Run("Write empty data to new sheet", func(t *testing.T) {
		var data [][]interface{}

		// Verify sheet doesn't exist before writing
		sheets := excel.ListSheets()
		assert.NotContains(t, sheets, "EmptySheet")

		err := excel.WriteAll("EmptySheet", "A1", data)
		assert.NoError(t, err)

		// Verify sheet was created but is empty
		sheets = excel.ListSheets()
		assert.Contains(t, sheets, "EmptySheet")

		rows, err := excel.GetRows("EmptySheet")
		assert.NoError(t, err)
		assert.Empty(t, rows)
	})

	t.Run("Write empty data to existing sheet", func(t *testing.T) {
		// First write some data
		data := [][]interface{}{
			{"Test"},
		}
		err := excel.WriteAll("ExistingSheet", "A1", data)
		assert.NoError(t, err)

		// Then write empty data
		var emptyData [][]interface{}
		err = excel.WriteAll("ExistingSheet", "A1", emptyData)
		assert.NoError(t, err)

		// Verify original data remains
		rows, err := excel.GetRows("ExistingSheet")
		assert.NoError(t, err)
		assert.NotEmpty(t, rows)
		assert.Equal(t, "Test", rows[0][0])
	})

	t.Run("Write with invalid cell reference", func(t *testing.T) {
		data := [][]interface{}{
			{"Test"},
		}
		err := excel.WriteAll("InvalidCell", "INVALID", data)
		assert.Error(t, err)

		// Verify sheet was still created despite error
		sheets := excel.ListSheets()
		assert.Contains(t, sheets, "InvalidCell")
	})

	t.Run("Write to sheet with special characters", func(t *testing.T) {
		// Valid sheet name with allowed special characters
		data := [][]interface{}{
			{"Special"},
		}
		err := excel.WriteAll("Sheet-123_中文", "A1", data)
		assert.NoError(t, err)

		// Verify sheet was created and data written
		sheets := excel.ListSheets()
		assert.Contains(t, sheets, "Sheet-123_中文")

		rows, err := excel.GetRows("Sheet-123_中文")
		assert.NoError(t, err)
		assert.Equal(t, "Special", rows[0][0])

		// Invalid sheet names
		invalidNames := []string{
			"Sheet:1",
			"Sheet/2",
			"Sheet\\3",
			"Sheet?4",
			"Sheet*5",
			"Sheet[6]",
			"", // Empty name
			"ThisSheetNameIsWayTooLongAndShouldFailBecauseExcelHasALimitOf31Characters", // Too long
		}

		for _, name := range invalidNames {
			err := excel.WriteAll(name, "A1", data)
			assert.Error(t, err, "Should fail for invalid sheet name: %s", name)
		}
	})

	// Optional: Save the file for manual inspection
	// err := excel.SaveAs("test_output.xlsx")
	// assert.NoError(t, err)
}
