package form

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/test"
	"github.com/yaoapp/yao/widgets/app"
	"github.com/yaoapp/yao/widgets/expression"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/table"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(Forms))
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
		"name": "Pet Admin Form Bind Model",
		"action": {
		  "bind": { "model": "pet" }
		}
	  }
	`)

	form, err := LoadSourceSync(source, `dynamic.pet`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Pet Admin Form Bind Model", form.Name)
	assert.Equal(t, "pet", form.Action.Bind.Model)
	assert.True(t, Exists("dynamic.pet"))

	// Reload
	form, err = form.Reload()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Pet Admin Form Bind Model", form.Name)
	assert.Equal(t, "pet", form.Action.Bind.Model)
	assert.True(t, Exists("dynamic.pet"))

	// Unload
	Unload("dynamic.pet")
	assert.False(t, Exists("dynamic.pet"))
}

func TestRead(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)
	form := MustGet("pet")
	assert.NotNil(t, form)

	// Read
	source := form.Read()
	if source == nil {
		t.Fatal("Read Error")
	}

	form, err := LoadSourceSync(source, `dynamic.pet`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "::Pet Admin", form.Name)
}

func prepare(t *testing.T) {

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

	// load tables
	err = table.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

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
