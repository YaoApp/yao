package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestIP(t *testing.T) {
	resp := IP()
	assert.True(t, len(resp) > 0)
}

func TestProcessIP(t *testing.T) {
	res := gou.NewProcess("xiang.network.ip").Run()
	resp, ok := res.(map[string]string)
	assert.True(t, ok)
	assert.True(t, len(resp) > 0)
}

func TestFreePort(t *testing.T) {
	port := FreePort()
	assert.True(t, port > 0)
}

func TestProcessFreePort(t *testing.T) {
	res := gou.NewProcess("xiang.network.FreePort").Run()
	port, ok := res.(int)
	assert.True(t, ok)
	assert.True(t, port > 0)
}
