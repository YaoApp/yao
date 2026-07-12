//go:build unit

package webproxy_test

import (
	"testing"

	"github.com/yaoapp/yao/tai/webproxy"
)

func TestProbe_HostID_ReturnsDirect(t *testing.T) {
	mode, addr := webproxy.ExportProbe(webproxy.BindOptions{
		TargetID:   webproxy.HostID,
		TargetPort: 3000,
	})
	if mode != webproxy.ModeDirect {
		t.Fatalf("expected ModeDirect, got %d", mode)
	}
	if addr != "127.0.0.1:3000" {
		t.Fatalf("expected 127.0.0.1:3000, got %s", addr)
	}
}

func TestProbe_EmptyContainer_ReturnsDirect(t *testing.T) {
	mode, addr := webproxy.ExportProbe(webproxy.BindOptions{
		TargetID:   "some-box",
		TargetPort: 8080,
	})
	if mode != webproxy.ModeDirect {
		t.Fatalf("expected ModeDirect for empty ContainerID, got %d", mode)
	}
	if addr != "127.0.0.1:8080" {
		t.Fatalf("expected 127.0.0.1:8080, got %s", addr)
	}
}

func TestProbe_UseTunnel_ReturnsTaiProxy(t *testing.T) {
	mode, addr := webproxy.ExportProbe(webproxy.BindOptions{
		TargetID:    "box-123",
		ContainerID: "ctr-abc",
		TargetPort:  3000,
		UseTunnel:   true,
	})
	if mode != webproxy.ModeTaiProxy {
		t.Fatalf("expected ModeTaiProxy, got %d", mode)
	}
	if addr != "" {
		t.Fatalf("expected empty addr, got %s", addr)
	}
}
