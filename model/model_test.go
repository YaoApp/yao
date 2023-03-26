package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/gou/model"
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
	for id := range model.Models {
		ids[id] = true
	}

	assert.True(t, ids["user"])
	assert.True(t, ids["category"])
	assert.True(t, ids["tag"])
	assert.True(t, ids["pet"])
	assert.True(t, ids["pet.tag"])
	assert.True(t, ids["user.pet"])
	assert.True(t, ids["tests.user"])
}
