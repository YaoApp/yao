package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/fs"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/test"
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
	assert.Equal(t, 3, len(Lists))
}

func prepare(t *testing.T, language ...string) {

	i18n.Load(config.Conf)

	// load fs
	err := fs.Load(config.Conf)
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

	// export
	err = Export()
	if err != nil {
		t.Fatal(err)
	}
}
