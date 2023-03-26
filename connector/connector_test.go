package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/connector"
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
	for id := range connector.Connectors {
		ids[id] = true
	}
	assert.True(t, ids["mongo"])
	assert.True(t, ids["mysql"])
	assert.True(t, ids["redis"])
	assert.True(t, ids["sqlite"])
}
