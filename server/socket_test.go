package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range gou.Servers {
		keys = append(keys, key)
	}
	assert.Equal(t, 1, len(keys))
}
