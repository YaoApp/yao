package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestProcessApis(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	testData(t)
	args := []interface{}{}
	res, err := process.New("widget.apis", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(res.([]Item)), 0)
}

func TestProcessActions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	testData(t)
	args := []interface{}{}
	res, err := process.New("widget.actions", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(res.([]Item)), 0)
}

func TestProcessModels(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	testData(t)
	args := []interface{}{}
	res, err := process.New("widget.models", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(res.([]Item)), 0)
}

func TestProcessFields(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	testData(t)
	args := []interface{}{}
	res, err := process.New("widget.fields", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(res.([]Item)), 0)
}

func TestProcessFilters(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	testData(t)
	args := []interface{}{}
	res, err := process.New("widget.filters", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(res.([]Item)), 0)
}

func testData(t *testing.T) {

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
