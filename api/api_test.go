package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range api.APIs {
		ids[id] = true
	}
	assert.True(t, ids["user"])
	assert.True(t, ids["user.pet"])

	// assert.True(t, ids["xiang.import"])  // will be removed in the future
	// assert.True(t, ids["xiang.storage"]) // will be removed in the future

	// wskeys := []string{}
	// for key := range websocket.Upgraders {
	// 	wskeys = append(wskeys, key)
	// }

	// assert.Equal(t, 5, len(keys))
	// assert.Equal(t, 1, len(wskeys))
}
