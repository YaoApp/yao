package table

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/widgets/app"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/expression"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/test"
)

func TestLoad(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(Tables))
}

func TestLoadID(t *testing.T) {
	prepare(t)
	err := LoadID("pet", filepath.Join(config.Conf.Root, "tables"))
	if err != nil {
		t.Fatal(err)
	}
}

func prepare(t *testing.T, language ...string) {

	err := test.LoadEngine(language...)
	if err != nil {
		t.Fatal(err)
	}

	// load scripts
	err = script.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// load models
	err = model.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// load flows
	err = flow.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// load app widget
	err = app.LoadAndExport(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// load field transform
	err = field.LoadAndExport(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// load expression
	err = expression.Export()
	if err != nil {
		t.Fatal(err)
	}

	// load component
	err = component.Export()
	if err != nil {
		t.Fatal(err)
	}

	// export
	err = Export()
	if err != nil {
		t.Fatal(err)
	}

}
