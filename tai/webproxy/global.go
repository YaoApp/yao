package webproxy

import (
	"context"
	"sync"
	"time"
)

var (
	instance *WebProxy
	mu       sync.Mutex
	cancel   context.CancelFunc
)

// Init initializes the global WebProxy singleton with the given config.
// Can be called multiple times (e.g., on config reload); previous instance
// is shut down and replaced.
func Init(cfg Config) {
	mu.Lock()
	defer mu.Unlock()

	if cancel != nil {
		cancel()
	}
	instance = New(cfg)
	ctx, c := context.WithCancel(context.Background())
	cancel = c
	instance.StartIdleReaper(ctx)
}

// WP returns the global WebProxy instance, or nil if not initialized.
func WP() *WebProxy {
	mu.Lock()
	defer mu.Unlock()
	return instance
}

// ParseIdleTimeout parses a duration string (e.g., "30m") into time.Duration.
func ParseIdleTimeout(s string) time.Duration {
	if s == "" {
		return 30 * time.Minute
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}
