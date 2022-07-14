package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
)

func TestLoad(t *testing.T) {
	Load(config.Conf)
	LoadFrom("not a path", "404.")
	check(t)
}

func TestProcessStart(t *testing.T) {
	Load(config.Conf)
	// assert.NotPanics(t, func() {
	// 	gou.NewProcess("xiang.server.Start", "rfid").Run()
	// })
}

// func TestProcessConnect(t *testing.T) {
// 	Load(config.Conf)
// 	// assert.NotPanics(t, func() {
// 	// 	gou.NewProcess("xiang.server.Connect", "rfid_client").Run()
// 	// })
// }

func check(t *testing.T) {
	keys := []string{}
	for key := range gou.Sockets {
		keys = append(keys, key)
	}
	assert.Equal(t, 2, len(keys))
}
