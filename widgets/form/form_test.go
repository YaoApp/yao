package form

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/lang"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/script"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(Forms))
}

func prepare(t *testing.T, language ...string) {

	// langs
	if len(language) < 1 {
		os.Unsetenv("YAO_LANG")
	} else {
		os.Setenv("YAO_LANG", language[0])
	}
	lang.Load(config.Conf)

	share.DBConnect(config.Conf.DB) // removed later

	// load scripts
	err := script.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// load models
	err = model.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// load scripts

	// export
	err = Export()
	if err != nil {
		t.Fatal(err)
	}
}
