package form

import (
	"path/filepath"
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

	err := LoadID("pet", filepath.Join(config.Conf.Root))
	if err != nil {
		t.Fatal(err)
	}
}

func prepare(t *testing.T, language ...string) {

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
