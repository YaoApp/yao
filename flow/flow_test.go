package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/runtime"
)

func TestLoad(t *testing.T) {
	runtime.Load(config.Conf)
	gou.Flows = make(map[string]*gou.Flow)
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func check(t *testing.T) {
	keys := []string{}
	for key := range gou.Flows {
		keys = append(keys, key)
	}
	assert.Equal(t, 26, len(keys))
}
