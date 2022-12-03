package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/api"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widgets/test"
)

func TestLoad(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	assert.Nil(t, err)
}

func prepare(t *testing.T, language ...string) {

	err := test.LoadEngine(language...)
	if err != nil {
		t.Fatal(err)
	}

	runtime.Load(config.Conf)
	i18n.Load(config.Conf)
	share.DBConnect(config.Conf.DB) // removed later

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

	// load apis
	err = api.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

}
