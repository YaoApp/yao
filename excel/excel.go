package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"github.com/yaoapp/yao/config"
)

// Excel the excel file
type Excel struct {
	id     string
	path   string
	create int64
	abs    string
	*excelize.File
}

// openFiles the open files
var openFiles = sync.Map{}

// Open open the excel file
func Open(path string, writable bool) (string, error) {

	excel := &Excel{path: path}
	// GET DATA ROOT
	root := config.Conf.DataRoot
	absPath, err := filepath.Abs(filepath.Join(root, path))
	if err != nil {
		return "", err
	}

	if writable {

		// if the file not exists, create it
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			create := excelize.NewFile()
			err := create.SaveAs(absPath)
			if err != nil {
				return "", err
			}
			create.Close()
		}

		excelFile, err := excelize.OpenFile(absPath)
		if err != nil {
			return "", err
		}
		id := uuid.NewString()
		excel.File = excelFile
		excel.id = id
		excel.abs = absPath
		excel.create = time.Now().Unix()
		openFiles.Store(id, excel)
		return id, nil
	}

	file, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("open file %s failed: %w", absPath, err)
	}

	excelFile, err := excelize.OpenReader(file)
	if err != nil {
		return "", err
	}

	id := uuid.NewString()
	excel.File = excelFile
	excel.id = id
	excel.abs = absPath
	excel.create = time.Now().Unix()
	openFiles.Store(id, excel)
	return id, nil
}

// Close close the excel file
func Close(handler string) error {
	excel, ok := openFiles.Load(handler)
	if !ok {
		return fmt.Errorf("file not found")
	}

	err := excel.(*Excel).Close()
	if err != nil {
		return err
	}

	openFiles.Delete(handler)
	return nil
}

// Get get the excel file
func Get(handler string) (*Excel, error) {
	excel, ok := openFiles.Load(handler)
	if !ok {
		return nil, fmt.Errorf("%s not found", handler)
	}
	return excel.(*Excel), nil
}
