package cmd

import (
	"testing"

	"github.com/yaoapp/yao/config"
)

func TestStartTaiTransportDisabled(t *testing.T) {
	oldEnabled, oldHost := config.Conf.Tai.Server.Enabled, config.Conf.Host
	config.Conf.Tai.Server.Enabled = "false"
	config.Conf.Host = "127.0.0.1"
	t.Cleanup(func() {
		config.Conf.Tai.Server.Enabled = oldEnabled
		config.Conf.Host = oldHost
	})
	server, err := startTaiTransport()
	if err != nil || server != nil {
		t.Fatalf("disabled start = %v, %v", server, err)
	}
}
