package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/connector"
)

func TestLoad(t *testing.T) {
	connector.Load(config.Conf)
	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range store.Pools {
		keys = append(keys, key)
	}
	assert.Equal(t, 3, len(keys))
}
