package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	err := Load(config.Conf)
	assert.Nil(t, err)
}
