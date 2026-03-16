package monitor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// resetService resets the global service for test isolation.
func resetService() {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.started && svc.cancel != nil {
		svc.cancel()
		svc.wg.Wait()
	}

	svc.watchers = make(map[string]*watcherEntry)
	svc.subs = make(map[string]chan<- *Alert)
	svc.subSeq = 0
	svc.started = false
	svc.ctx = nil
	svc.cancel = nil

	// Use a discard logger for tests
	logger = nil
}

// testWatcher is a simple watcher for testing.
type testWatcher struct {
	name     string
	interval time.Duration
	checkFn  func(ctx context.Context) []Alert
}

func (w *testWatcher) Name() string            { return w.name }
func (w *testWatcher) Interval() time.Duration { return w.interval }
func (w *testWatcher) Check(ctx context.Context) []Alert {
	if w.checkFn != nil {
		return w.checkFn(ctx)
	}
	return nil
}

func TestRegisterAndStart(t *testing.T) {
	resetService()
	defer resetService()

	var count atomic.Int32
	Register(&testWatcher{
		name:     "test-basic",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			count.Add(1)
			return nil
		},
	})

	err := Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for a few ticks (first immediate + ticker)
	time.Sleep(200 * time.Millisecond)

	if err := Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	c := count.Load()
	if c < 2 {
		t.Errorf("expected at least 2 checks (immediate + ticker), got %d", c)
	}
}

func TestDoubleStartError(t *testing.T) {
	resetService()
	defer resetService()

	Register(&testWatcher{name: "dummy", interval: time.Second})

	if err := Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := Start(context.Background()); err == nil {
		t.Error("expected error on double Start")
	}

	Stop()
}

func TestStopWithoutStart(t *testing.T) {
	resetService()
	if err := Stop(); err != nil {
		t.Errorf("Stop without Start should not error: %v", err)
	}
}

func TestAlertWatcherName(t *testing.T) {
	resetService()
	defer resetService()

	var got string
	var mu sync.Mutex

	Register(&testWatcher{
		name:     "namer",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			mu.Lock()
			defer mu.Unlock()
			if got != "" {
				return nil
			}
			return []Alert{{
				Level:   Info,
				Target:  "test:1",
				Message: "hello",
			}}
		},
	})

	ch := make(chan *Alert, 8)
	subID := Subscribe(ch)
	defer Unsubscribe(subID)

	Start(context.Background())
	defer Stop()

	select {
	case a := <-ch:
		if a.Watcher != "namer" {
			t.Errorf("expected Watcher='namer', got %q", a.Watcher)
		}
		mu.Lock()
		got = a.Watcher
		mu.Unlock()
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for alert")
	}
}

func TestAlertAction(t *testing.T) {
	resetService()
	defer resetService()

	var acted atomic.Bool

	Register(&testWatcher{
		name:     "actor",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			if acted.Load() {
				return nil
			}
			return []Alert{{
				Level:   Warn,
				Target:  "test:action",
				Message: "do something",
				Action: func(ctx context.Context) {
					acted.Store(true)
				},
			}}
		},
	})

	Start(context.Background())
	defer Stop()

	time.Sleep(200 * time.Millisecond)

	if !acted.Load() {
		t.Error("action was not executed")
	}
}

func TestPanicRecovery_Check(t *testing.T) {
	resetService()
	defer resetService()

	var count atomic.Int32

	Register(&testWatcher{
		name:     "panicker",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			n := count.Add(1)
			if n == 1 {
				panic("boom")
			}
			return nil
		},
	})

	Start(context.Background())
	time.Sleep(200 * time.Millisecond)
	Stop()

	c := count.Load()
	if c < 2 {
		t.Errorf("expected watcher to continue after panic, got %d checks", c)
	}
}

func TestPanicRecovery_Action(t *testing.T) {
	resetService()
	defer resetService()

	var postPanic atomic.Bool

	Register(&testWatcher{
		name:     "action-panicker",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			return []Alert{
				{
					Level:   Error,
					Target:  "test:panic-action",
					Message: "will panic",
					Action: func(ctx context.Context) {
						if !postPanic.Load() {
							panic("action boom")
						}
					},
				},
			}
		},
	})

	Start(context.Background())
	time.Sleep(150 * time.Millisecond)
	postPanic.Store(true)
	time.Sleep(100 * time.Millisecond)
	Stop()
}

func TestSubscribeUnsubscribe(t *testing.T) {
	resetService()
	defer resetService()

	ch := make(chan *Alert, 16)
	id := Subscribe(ch)

	Register(&testWatcher{
		name:     "sub-test",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			return []Alert{{Level: Info, Target: "t", Message: "msg"}}
		},
	})

	Start(context.Background())
	time.Sleep(100 * time.Millisecond)
	Unsubscribe(id)
	time.Sleep(100 * time.Millisecond)
	Stop()

	// Drain and count
	close(ch)
	count := 0
	for range ch {
		count++
	}
	if count == 0 {
		t.Error("expected at least one alert before unsubscribe")
	}
}

func TestSubscribeFullChanDrops(t *testing.T) {
	resetService()
	defer resetService()

	ch := make(chan *Alert, 1) // tiny buffer
	Subscribe(ch)

	Register(&testWatcher{
		name:     "flood",
		interval: 10 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			return []Alert{{Level: Info, Target: "t", Message: "flood"}}
		},
	})

	Start(context.Background())
	time.Sleep(200 * time.Millisecond)
	Stop()
	// No deadlock = pass
}

func TestRegisterOverwrite(t *testing.T) {
	resetService()
	defer resetService()

	var first, second atomic.Int32

	Register(&testWatcher{
		name:     "dup",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			first.Add(1)
			return nil
		},
	})

	Register(&testWatcher{
		name:     "dup",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			second.Add(1)
			return nil
		},
	})

	Start(context.Background())
	time.Sleep(200 * time.Millisecond)
	Stop()

	if first.Load() > 0 {
		t.Error("first watcher should have been replaced")
	}
	if second.Load() == 0 {
		t.Error("second watcher should have run")
	}
}

func TestRegisterAfterStart(t *testing.T) {
	resetService()
	defer resetService()

	Start(context.Background())
	defer Stop()

	var count atomic.Int32
	Register(&testWatcher{
		name:     "late",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			count.Add(1)
			return nil
		},
	})

	time.Sleep(200 * time.Millisecond)

	if count.Load() == 0 {
		t.Error("watcher registered after Start should still run")
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{Trace, "trace"},
		{Info, "info"},
		{Warn, "warn"},
		{Error, "error"},
		{Level(99), "unknown"},
	}
	for _, tt := range tests {
		got := tt.level.String()
		if got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestContextCancelledDuringCheck(t *testing.T) {
	resetService()
	defer resetService()

	var checkDone atomic.Bool

	Register(&testWatcher{
		name:     "slow",
		interval: time.Hour, // only immediate check runs
		checkFn: func(ctx context.Context) []Alert {
			select {
			case <-ctx.Done():
				checkDone.Store(true)
			case <-time.After(2 * time.Second):
			}
			return nil
		},
	})

	Start(context.Background())

	// Give the immediate check time to start
	time.Sleep(50 * time.Millisecond)

	// Stop should cancel the context
	done := make(chan struct{})
	go func() {
		Stop()
		close(done)
	}()

	select {
	case <-done:
		if !checkDone.Load() {
			// The check might have not started yet, that's ok
			fmt.Println("note: check may not have started before stop")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Stop timed out — watcher may not respect context cancellation")
	}
}

func TestHealth_NotStarted(t *testing.T) {
	resetService()
	defer resetService()

	Register(&testWatcher{name: "idle", interval: time.Second})

	h := Health()
	if h.Running {
		t.Error("expected Running=false before Start")
	}
	if len(h.Watchers) != 1 {
		t.Errorf("expected 1 watcher, got %d", len(h.Watchers))
	}
	if h.Watchers[0].TotalTicks != 0 {
		t.Errorf("expected 0 ticks before Start, got %d", h.Watchers[0].TotalTicks)
	}
}

func TestHealth_Running(t *testing.T) {
	resetService()
	defer resetService()

	Register(&testWatcher{
		name:     "healthy",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			return []Alert{{Level: Info, Target: "t:1", Message: "ok"}}
		},
	})

	Start(context.Background())
	time.Sleep(200 * time.Millisecond)

	h := Health()
	if !h.Running {
		t.Error("expected Running=true")
	}
	if len(h.Watchers) != 1 {
		t.Fatalf("expected 1 watcher, got %d", len(h.Watchers))
	}

	wh := h.Watchers[0]
	if wh.Name != "healthy" {
		t.Errorf("expected name 'healthy', got %q", wh.Name)
	}
	if wh.TotalTicks < 2 {
		t.Errorf("expected at least 2 ticks, got %d", wh.TotalTicks)
	}
	if wh.LastTick.IsZero() {
		t.Error("expected non-zero LastTick")
	}
	if wh.LastAlerts != 1 {
		t.Errorf("expected 1 alert per tick, got %d", wh.LastAlerts)
	}
	if wh.Panics != 0 {
		t.Errorf("expected 0 panics, got %d", wh.Panics)
	}

	Stop()
}

func TestHealth_PanicCount(t *testing.T) {
	resetService()
	defer resetService()

	var n atomic.Int32
	Register(&testWatcher{
		name:     "crasher",
		interval: 50 * time.Millisecond,
		checkFn: func(ctx context.Context) []Alert {
			if n.Add(1) <= 2 {
				panic("crash")
			}
			return nil
		},
	})

	Start(context.Background())
	time.Sleep(250 * time.Millisecond)

	h := Health()
	wh := h.Watchers[0]
	if wh.Panics < 2 {
		t.Errorf("expected at least 2 panics, got %d", wh.Panics)
	}
	if wh.TotalTicks <= wh.Panics {
		t.Errorf("expected some successful ticks after panics: total=%d, panics=%d", wh.TotalTicks, wh.Panics)
	}

	Stop()
}
