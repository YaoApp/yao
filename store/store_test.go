package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/connector"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	loadConnectors(t)

	// Remove the data store (For cleaning the stores whitch created by the test)
	var path = filepath.Join(config.Conf.DataRoot, "stores")
	os.RemoveAll(path)

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range store.Pools {
		ids[id] = true
	}
	assert.True(t, ids["cache"])
	assert.True(t, ids["data"])
	assert.True(t, ids["share"])

	// System stores
	assert.True(t, ids["__yao.store"])
	assert.True(t, ids["__yao.cache"])
	assert.True(t, ids["__yao.oauth.store"])
	assert.True(t, ids["__yao.oauth.client"])
	assert.True(t, ids["__yao.oauth.cache"])
	assert.True(t, ids["__yao.agent.memory.user"])
	assert.True(t, ids["__yao.agent.memory.team"])
	assert.True(t, ids["__yao.agent.memory.chat"])
	assert.True(t, ids["__yao.agent.memory.context"])
	assert.True(t, ids["__yao.agent.cache"])
}

func loadConnectors(t *testing.T) {
	err := connector.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
