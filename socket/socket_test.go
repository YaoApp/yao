package socket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/socket"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range socket.Sockets {
		keys = append(keys, key)
	}
	assert.Equal(t, 0, len(keys))
}
