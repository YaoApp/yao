package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	assert.Equal(t, "Yao", share.App.L["Yao"])
	assert.Equal(t, "Xiang", share.App.L["象传"])
}
