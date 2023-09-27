package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/sui/storages/local"
)

func TestTemplateGet(t *testing.T) {
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

func TestTemplateFind(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.find", "demo", "tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &local.Template{}, res)
	assert.Equal(t, "tech-blue", res.(*local.Template).ID)
}

func load(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
