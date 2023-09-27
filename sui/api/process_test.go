package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/sui/core"
)

func TestTemplates(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.get", "demo")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.ITemplate{}, res)
	assert.Equal(t, 3, len(res.([]core.ITemplate)))
}

func load(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
