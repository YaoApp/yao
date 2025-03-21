package excel

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// Cols defines an iterator to a sheet
type Cols struct {
	id string
	*excelize.Cols
	create int64
}

// Rows defines an iterator to a sheet
type Rows struct {
	id string
	*excelize.Rows
	create int64
}

var openCols = sync.Map{}
var openRows = sync.Map{}

// OpenRow each row of the sheet
func (excel *Excel) OpenRow(sheet string) (string, error) {
	id := uuid.NewString()
	rows, err := excel.Rows(sheet)
	if err != nil {
		return "", err
	}
	openRows.Store(id, &Rows{id: id, Rows: rows, create: time.Now().Unix()})
	return id, nil
}

// NextRow next row of the sheet
func NextRow(id string) ([]string, error) {
	value, ok := openRows.Load(id)
	if !ok {
		return nil, fmt.Errorf("rows %s not found", id)
	}

	if value.(*Rows).Next() {
		row, err := value.(*Rows).Columns()
		// fmt.Printf("DEBUG: %#v %v %v\n", row, err, row == nil)
		if err != nil {
			return nil, err
		}

		if row == nil {
			return []string{}, nil
		}
		return row, nil
	}
	return nil, nil
}

// CloseRow done the sheet
func CloseRow(id string) {
	openRows.Delete(id)
}

// OpenColumn each cols of the sheet
func (excel *Excel) OpenColumn(sheet string) (string, error) {
	id := uuid.NewString()
	cols, err := excel.Cols(sheet)
	if err != nil {
		return "", err
	}
	openCols.Store(id, &Cols{id: id, Cols: cols, create: time.Now().Unix()})
	return id, nil
}

// NextColumn next col of the sheet
func NextColumn(id string) ([]string, error) {
	value, ok := openCols.Load(id)
	if !ok {
		return nil, fmt.Errorf("cols %s not found", id)
	}

	if value.(*Cols).Next() {
		col, err := value.(*Cols).Rows()
		if err != nil {
			return nil, err
		}

		if col == nil {
			return []string{}, nil
		}

		return col, nil
	}

	return nil, nil
}

// CloseColumn done the sheet
func CloseColumn(id string) {
	openCols.Delete(id)
}
