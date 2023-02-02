package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/connector"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	loadConnectors(t)

	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range query.Engines {
		ids[id] = true
	}
	assert.True(t, ids["default"])
	assert.True(t, ids["mysql"])
	assert.True(t, ids["sqlite"])
}

func loadConnectors(t *testing.T) {
	err := connector.Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
