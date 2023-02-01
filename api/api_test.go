package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range api.APIs {
		keys = append(keys, key)
	}

	wskeys := []string{}
	for key := range websocket.Upgraders {
		wskeys = append(wskeys, key)
	}

	assert.Equal(t, 5, len(keys))
	assert.Equal(t, 1, len(wskeys))
}
