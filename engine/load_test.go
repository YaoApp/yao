package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	defer Unload()
	_, err := Load(config.Conf, LoadOption{})
	assert.Nil(t, err)
	assert.Greater(t, len(api.APIs), 0)
}

func TestReload(t *testing.T) {
	defer Unload()
	_, err := Load(config.Conf, LoadOption{})
	assert.Nil(t, err)

	Reload(config.Conf, LoadOption{})
	assert.Nil(t, err)
	assert.Greater(t, len(api.APIs), 0)
}
