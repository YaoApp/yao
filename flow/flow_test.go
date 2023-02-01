package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/flow"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range flow.Flows {
		keys = append(keys, key)
	}
	assert.Equal(t, 26, len(keys))
}
