package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range connector.Connectors {
		keys = append(keys, key)
	}
	assert.Equal(t, 4, len(keys))
}
