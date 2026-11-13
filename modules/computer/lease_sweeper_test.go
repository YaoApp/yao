package computer

import (
	"context"
	"testing"
	"time"
)

func TestRunLeaseSweeperStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		New().RunLeaseSweeper(ctx, time.Millisecond)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sweeper did not stop")
	}
}
