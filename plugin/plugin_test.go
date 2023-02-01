package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range plugin.Plugins {
		keys = append(keys, key)
	}
	assert.Equal(t, 1, len(keys))
}
