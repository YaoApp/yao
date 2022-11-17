package importer

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestProcessMapping(t *testing.T) {
	simple := filepath.Join("assets", "simple.xlsx")
	args := []interface{}{"order", simple}
	response := gou.NewProcess("xiang.import.Mapping", args...).Run()
	_, ok := response.(*Mapping)
	assert.True(t, ok)
}

func TestProcessMappingSetting(t *testing.T) {
	simple := filepath.Join("assets", "simple.xlsx")
	args := []interface{}{"order", simple}
	response := gou.NewProcess("xiang.import.MappingSetting", args...).Run()
	_, ok := response.(map[string]interface{})
	assert.True(t, ok)
}

func TestProcessData(t *testing.T) {
	simple := filepath.Join("assets", "simple.xlsx")
	mapping := gou.NewProcess("xiang.import.Mapping", "order", simple).Run()
	args := []interface{}{"order", simple, 1, 2, mapping}
	response := gou.NewProcess("xiang.import.Data", args...).Run()
	_, ok := response.(map[string]interface{})
	assert.True(t, ok)
}

func TestProcessDataSetting(t *testing.T) {
	args := []interface{}{"order"}
	response := gou.NewProcess("xiang.import.DataSetting", args...).Run()
	_, ok := response.(map[string]interface{})
	assert.True(t, ok)
}

func TestProcessSetting(t *testing.T) {
	args := []interface{}{"order"}
	response := gou.NewProcess("xiang.import.Setting", args...).Run()
	_, ok := response.(map[string]interface{})
	assert.True(t, ok)
}

func TestProcessRun(t *testing.T) {
	simple := filepath.Join("assets", "simple.xlsx")
	mapping := gou.NewProcess("xiang.import.Mapping", "order", simple).Run()
	args := []interface{}{"order", simple, mapping}
	response := gou.NewProcess("xiang.import.Run", args...).Run()
	_, ok := response.(map[string]int)
	assert.True(t, ok)
}
