package sandbox_test

import (
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestWatcherName(t *testing.T) {
	name := sandbox.ExportWatcherName()
	if name != "sandbox" {
		t.Errorf("Name: got %q, want %q", name, "sandbox")
	}
}

func TestWatcherInterval(t *testing.T) {
	interval := sandbox.ExportWatcherInterval()
	want := 30 * time.Second
	if interval != want {
		t.Errorf("Interval: got %v, want %v", interval, want)
	}
}
