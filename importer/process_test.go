package importer

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessMapping(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t, config.Conf)

	simple := filepath.Join("assets", "simple.xlsx")
	args := []interface{}{"order", simple}
	response := process.New("yao.import.Mapping", args...).Run()
	_, ok := response.(*Mapping)
	assert.True(t, ok)
}

func TestProcessMappingSetting(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t, config.Conf)

	simple := filepath.Join("assets", "simple.xlsx")
	args := []interface{}{"order", simple}
	response := process.New("yao.import.MappingSetting", args...).Run()
	_, ok := response.(map[string]interface{})
	assert.True(t, ok)
}

func TestProcessData(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t, config.Conf)

	simple := filepath.Join("assets", "simple.xlsx")
	mapping := process.New("yao.import.Mapping", "order", simple).Run()
	args := []interface{}{"order", simple, 1, 2, mapping}
	response := process.New("yao.import.Data", args...).Run()
	_, ok := response.(map[string]interface{})
	assert.True(t, ok)
}

func TestProcessDataSetting(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t, config.Conf)

	args := []interface{}{"order"}
	response := process.New("yao.import.DataSetting", args...).Run()
	_, ok := response.(map[string]interface{})
	assert.True(t, ok)
}

func TestProcessSetting(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t, config.Conf)

	args := []interface{}{"order"}
	response := process.New("yao.import.Setting", args...).Run()
	_, ok := response.(map[string]interface{})
	assert.True(t, ok)
}

func TestProcessRun(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t, config.Conf)

	simple := filepath.Join("assets", "simple.xlsx")
	mapping := process.New("yao.import.Mapping", "order", simple).Run()
	args := []interface{}{"order", simple, mapping}
	response := process.New("yao.import.Run", args...).Run()
	_, ok := response.(map[string]int)
	assert.True(t, ok)
}
