package table

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

// Export Export query result to Excel
func (dsl *DSL) Export(filename string, data interface{}, page int, chunkSize int) error {

	log.Trace("[Export] %s %d %d Before: %#v", filename, page, chunkSize, data)

	rows := []maps.MapStr{}
	if values, ok := data.([]maps.MapStrAny); ok {
		for _, row := range values {
			rows = append(rows, row.Dot())
		}
	} else if values, ok := data.([]map[string]interface{}); ok {
		for _, row := range values {
			rows = append(rows, maps.Of(row).Dot())
		}
	} else if values, ok := data.([]interface{}); ok {
		for _, row := range values {
			rows = append(rows, any.Of(row).MapStr().Dot())
		}
	}

	log.Trace("[Export] %s %d %d After: %#v", filename, page, chunkSize, data)
	columns, err := dsl.exportSetting()
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return fmt.Errorf("the table does not support export")
	}

	// filename = filepath.Join(xfs.Stor.Root, filename)
	fs, err := fs.Get("system")
	if err != nil {
		return err
	}

	filename = filepath.Join(fs.Root(), filename)
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		f := excelize.NewFile()
		index := f.GetActiveSheetIndex()
		name := f.GetSheetName(index)
		f.SetSheetName(name, dsl.Name)
		for i, column := range columns {
			axis, err := excelize.CoordinatesToCellName(i+1, 1)
			if err != nil {
				return err
			}
			f.SetCellValue(dsl.Name, axis, column["name"])
		}
		if err := f.SaveAs(filename); err != nil {
			fmt.Println(err)
			return err
		}
	}

	f, err := excelize.OpenFile(filename)
	if err != nil {
		return err
	}

	defer f.Close()
	offset := (page-1)*chunkSize + 2
	for line, row := range rows {
		for i, column := range columns {
			v := row.Get(column["field"])
			if v != nil {
				axis, err := excelize.CoordinatesToCellName(i+1, line+offset)
				if err != nil {
					return err
				}
				f.SetCellValue(dsl.Name, axis, v)
			}
		}
		// fmt.Println("--", line, page, offset, filename, chunkSize, "--")
	}

	return f.Save()
}

func (dsl *DSL) exportSetting() ([]map[string]string, error) {
	// Validate params
	if dsl.Layout == nil {
		return nil, fmt.Errorf("the table layout does not found")
	}

	if dsl.Fields == nil || dsl.Fields.Table == nil {
		return nil, fmt.Errorf("the table fields does not found")
	}

	if dsl.Layout.Table == nil || dsl.Layout.Table.Columns == nil {
		return nil, fmt.Errorf("the columns table layout does not found")
	}

	setting := []map[string]string{}
	for _, column := range dsl.Layout.Table.Columns {

		if field, has := dsl.Fields.Table[column.Name]; has {
			bind := field.Bind
			if field.View != nil && field.View.Bind != "" {
				bind = field.View.Bind
			}
			setting = append(setting, map[string]string{"name": column.Name, "field": bind})
		}
	}

	return setting, nil
}
