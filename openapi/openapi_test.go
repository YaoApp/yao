package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	_, err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, Server)
}
