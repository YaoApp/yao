package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {
	share.DBConnect(config.Conf.DB)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, lang.Dicts, 2)
}
