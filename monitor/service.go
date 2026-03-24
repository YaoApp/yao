package monitor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yaoapp/yao/config"
)

var svc = &monitorService{
	watchers: make(map[string]*watcherEntry),
	subs:     make(map[string]chan<- *Alert),
}

type watcherEntry struct {
	watcher    Watcher
	cancel     context.CancelFunc
	lastTick   atomic.Int64 // unix timestamp of last tick completion
	lastAlerts atomic.Int64 // alert count from last tick
	totalTicks atomic.Int64 // total ticks since start
	panics     atomic.Int64 // total panics caught
}

type monitorService struct {
	mu       sync.Mutex
	watchers map[string]*watcherEntry
	subs     map[string]chan<- *Alert
	subSeq   int
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	started  bool
}

// GetWatcher returns a registered watcher by name, or nil if not found.
func GetWatcher(name string) Watcher {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if entry, ok := svc.watchers[name]; ok {
		return entry.watcher
	}
	return nil
}

// Register adds a watcher. Call before Start (typically in init).
// Registering a watcher with the same name replaces the previous one.
func Register(w Watcher) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	name := w.Name()
	if old, ok := svc.watchers[name]; ok && old.cancel != nil {
		old.cancel()
	}
	svc.watchers[name] = &watcherEntry{watcher: w}

	if svc.started {
		svc.startWatcher(svc.watchers[name])
	}
}

// Start initializes the logger and launches a goroutine per registered watcher.
func Start(ctx context.Context) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.started {
		return fmt.Errorf("monitor: already started")
	}

	initLogger(config.Conf.Root, config.Conf.LogMode, config.Conf.Mode)

	svc.ctx, svc.cancel = context.WithCancel(ctx)
	for _, entry := range svc.watchers {
		svc.startWatcher(entry)
	}
	svc.started = true

	if logger != nil {
		logger.Info("monitor started", "watchers", len(svc.watchers))
	}
	return nil
}

// Stop cancels all watcher goroutines and waits for them to finish.
func Stop() error {
	svc.mu.Lock()
	if !svc.started {
		svc.mu.Unlock()
		return nil
	}
	svc.cancel()
	svc.started = false
	svc.mu.Unlock()

	svc.wg.Wait()

	if logger != nil {
		logger.Info("monitor stopped")
	}
	return nil
}

// Subscribe registers a channel to receive alert notifications.
// Returns a subscription ID for unsubscribing.
// Non-blocking: if the channel is full, alerts are dropped for that subscriber.
func Subscribe(ch chan<- *Alert) string {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.subSeq++
	id := fmt.Sprintf("sub-%d", svc.subSeq)
	svc.subs[id] = ch
	return id
}

// Unsubscribe removes a subscription by ID.
func Unsubscribe(id string) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	delete(svc.subs, id)
}

// WatcherHealth describes the runtime status of a single watcher.
type WatcherHealth struct {
	Name       string        `json:"name"`
	Interval   time.Duration `json:"interval"`
	LastTick   time.Time     `json:"last_tick"`   // zero if never ticked
	LastAlerts int64         `json:"last_alerts"` // alert count from most recent tick
	TotalTicks int64         `json:"total_ticks"`
	Panics     int64         `json:"panics"`
}

// HealthStatus describes the overall monitor health.
type HealthStatus struct {
	Running  bool            `json:"running"`
	Watchers []WatcherHealth `json:"watchers"`
}

// Health returns the current health status of the monitor service.
func Health() HealthStatus {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	status := HealthStatus{Running: svc.started}
	for _, entry := range svc.watchers {
		wh := WatcherHealth{
			Name:       entry.watcher.Name(),
			Interval:   entry.watcher.Interval(),
			LastAlerts: entry.lastAlerts.Load(),
			TotalTicks: entry.totalTicks.Load(),
			Panics:     entry.panics.Load(),
		}
		if ts := entry.lastTick.Load(); ts > 0 {
			wh.LastTick = time.Unix(ts, 0)
		}
		status.Watchers = append(status.Watchers, wh)
	}
	return status
}

func (s *monitorService) startWatcher(entry *watcherEntry) {
	ctx, cancel := context.WithCancel(s.ctx)
	entry.cancel = cancel
	s.wg.Add(1)
	go s.runLoop(ctx, entry)
}

func (s *monitorService) runLoop(ctx context.Context, entry *watcherEntry) {
	defer s.wg.Done()

	w := entry.watcher
	name := w.Name()
	interval := w.Interval()

	if logger != nil {
		logger.Info("watcher started", "watcher", name, "interval", interval)
	}

	// Run first check immediately, then on ticker.
	s.tick(ctx, entry)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick(ctx, entry)
		case <-ctx.Done():
			if logger != nil {
				logger.Info("watcher stopped", "watcher", name)
			}
			return
		}
	}
}

func (s *monitorService) tick(ctx context.Context, entry *watcherEntry) {
	name := entry.watcher.Name()

	defer func() {
		if r := recover(); r != nil {
			entry.panics.Add(1)
			if logger != nil {
				logger.Error("watcher panic", "watcher", name, "recover", fmt.Sprintf("%v", r))
			}
		}
		entry.totalTicks.Add(1)
		entry.lastTick.Store(time.Now().Unix())
	}()

	alerts := entry.watcher.Check(ctx)
	entry.lastAlerts.Store(int64(len(alerts)))

	for i := range alerts {
		a := &alerts[i]
		a.Watcher = name

		// Log level filtering is handled by slog handler:
		//   production  → Info and above (Trace skipped)
		//   development → Trace and above (everything)
		if logger != nil {
			logger.Log(ctx, levelToSlog(a.Level), a.Message,
				"watcher", name, "target", a.Target)
		}

		if a.Action != nil {
			s.execAction(ctx, name, a)
		}

		s.notify(a)
	}
}

func (s *monitorService) execAction(ctx context.Context, watcherName string, a *Alert) {
	defer func() {
		if r := recover(); r != nil {
			if logger != nil {
				logger.Error("action panic", "watcher", watcherName, "target", a.Target, "recover", fmt.Sprintf("%v", r))
			}
		}
	}()
	a.Action(ctx)
}

func (s *monitorService) notify(a *Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ch := range s.subs {
		select {
		case ch <- a:
		default:
		}
	}
}
