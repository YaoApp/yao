package table

import (
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
	err := LoadID("pet")
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadSourceSync(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)
	source := []byte(`{
		"name": "Pet Admin Bind Model And Form",
		"action": {
		  "bind": { "model": "pet", "option": { "form": "pet" } },
		  "search": {
			"guard": "-",
			"process": "scripts.pet.Search",
			"default": [null, 1, 5]
		  }
		}
	  }
	`)

	tab, err := LoadSourceSync(source, `dynamic.pet`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Pet Admin Bind Model And Form", tab.Name)
	assert.Equal(t, "pet", tab.Action.Bind.Model)
	assert.True(t, Exists("dynamic.pet"))

	// Reload
	tab, err = tab.Reload()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Pet Admin Bind Model And Form", tab.Name)
	assert.Equal(t, "pet", tab.Action.Bind.Model)
	assert.True(t, Exists("dynamic.pet"))

	// Unload
	Unload("dynamic.pet")
	assert.False(t, Exists("dynamic.pet"))
}

func TestRead(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)
	tab := MustGet("pet")
	assert.NotNil(t, tab)

	// Read
	source := tab.Read()
	if source == nil {
		t.Fatal("Read Error")
	}

	tab, err := LoadSourceSync(source, `dynamic.pet`)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "::Pet Admin", tab.Name)
}

func prepare(t *testing.T) {

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
