package table

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/test"
	_ "github.com/yaoapp/yao/utils"
	"github.com/yaoapp/yao/widgets/app"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/expression"
	"github.com/yaoapp/yao/widgets/field"
)

func TestLoad(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 13, len(Tables))
}

func TestLoadID(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	err := LoadID("pet", filepath.Join(config.Conf.Root))
	if err != nil {
		t.Fatal(err)
	}
}

func prepare(t *testing.T, language ...string) {

	// test.Prepare(t, config.Conf)
	// defer test.Clean()

	// // runtime.Load(config.Conf)
	// err := test.LoadEngine(language...)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // load fs
	// err = fs.Load(config.Conf)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // load scripts
	// err = script.Load(config.Conf)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // load models
	// err = model.Load(config.Conf)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// load flows
	err := flow.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	//  load app
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

	// Load table
	err = Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// export
	err = Export()
	if err != nil {
		t.Fatal(err)
	}
}
