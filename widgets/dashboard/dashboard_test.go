package dashboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/flow"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/test"
	"github.com/yaoapp/yao/widgets/component"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	prepare(t)

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(Dashboards))
}

func prepare(t *testing.T, language ...string) {
	i18n.Load(config.Conf)

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	err = flow.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// export
	err = Export()
	if err != nil {
		t.Fatal(err)
	}

	component.Export()
}
