package computer

import (
	"context"
	"errors"
	"testing"

	"github.com/yaoapp/yao/config"
)

func TestGetByNodeIDWithoutModel(t *testing.T) {
	if _, err := (&Service{}).GetByNodeID(context.Background(), "node"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err=%v", err)
	}
}

func TestTransportPortDefault(t *testing.T) {
	old := config.Conf.Tai.Server.TransportPort
	config.Conf.Tai.Server.TransportPort = 0
	t.Cleanup(func() { config.Conf.Tai.Server.TransportPort = old })
	if got := transportPort(); got != 6099 {
		t.Fatalf("port=%d", got)
	}
}
