package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestProcessMapping(t *testing.T) {
	args := []interface{}{"order"}
	response := gou.NewProcess("xiang.import.Mapping", args...).Run()
	assert.Nil(t, response)
}

func TestProcessMappingSetting(t *testing.T) {
	args := []interface{}{"order"}
	response := gou.NewProcess("xiang.import.MappingSetting", args...).Run()
	assert.Nil(t, response)
}

func TestProcessData(t *testing.T) {
	args := []interface{}{"order"}
	response := gou.NewProcess("xiang.import.Data", args...).Run()
	assert.Nil(t, response)
}

func TestProcessDataSetting(t *testing.T) {
	args := []interface{}{"order"}
	response := gou.NewProcess("xiang.import.DataSetting", args...).Run()
	assert.Nil(t, response)
}

func TestProcessRules(t *testing.T) {
	args := []interface{}{"order"}
	response := gou.NewProcess("xiang.import.Rules", args...).Run()
	assert.Nil(t, response)
}
