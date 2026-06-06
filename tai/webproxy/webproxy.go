package webproxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// WebProxy manages dynamic HTTP listeners that proxy to container services.
type WebProxy struct {
	mu       sync.RWMutex
	bindings map[int]*Binding // hostPort → Binding
	pool     *PortPool
	config   Config
}

// Config holds the webproxy configuration.
type Config struct {
	PortRangeStart int
	PortRangeEnd   int
	MaxPerTarget   int
	IdleTimeout    time.Duration
	Domain         string
	Prefix         string
}

// BindOptions contains all parameters needed to create a binding.
type BindOptions struct {
	TaiID       string
	TargetID    string // sandbox ID or "__host__"
	ContainerID string // Docker container ID (empty for __host__)
	TargetPort  int
	Label       string
}

// BindingInfo is the JSON-serializable representation of a binding.
type BindingInfo struct {
	HostPort   int       `json:"host_port"`
	TargetID   string    `json:"target_id"`
	TargetPort int       `json:"target_port"`
	Label      string    `json:"label"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// New creates a new WebProxy instance with the given config.
func New(cfg Config) *WebProxy {
	if cfg.PortRangeStart == 0 {
		cfg.PortRangeStart = 15000
	}
	if cfg.PortRangeEnd == 0 {
		cfg.PortRangeEnd = 15999
	}
	if cfg.MaxPerTarget == 0 {
		cfg.MaxPerTarget = 20
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 30 * time.Minute
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "p"
	}

	return &WebProxy{
		bindings: make(map[int]*Binding),
		pool:     NewPortPool(cfg.PortRangeStart, cfg.PortRangeEnd),
		config:   cfg,
	}
}

// Bind creates a new port binding and starts a listener goroutine.
func (wp *WebProxy) Bind(opts BindOptions) (*BindingInfo, error) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	// Idempotency: check if binding already exists for this target+port
	for _, b := range wp.bindings {
		if b.TargetID == opts.TargetID && b.TargetPort == opts.TargetPort {
			return b.Info(), nil
		}
	}

	// Check max per target
	count := 0
	for _, b := range wp.bindings {
		if b.TargetID == opts.TargetID {
			count++
		}
	}
	if count >= wp.config.MaxPerTarget {
		return nil, fmt.Errorf("max bindings per target reached (%d)", wp.config.MaxPerTarget)
	}

	// Allocate a port and get an open listener (eliminates TOCTOU race)
	hostPort, ln, err := wp.allocatePort()
	if err != nil {
		return nil, fmt.Errorf("no available ports: %w", err)
	}

	// Probe connection mode
	mode, targetAddr := probe(opts)

	ctx, cancel := context.WithCancel(context.Background())
	binding := &Binding{
		HostPort:    hostPort,
		TargetID:    opts.TargetID,
		ContainerID: opts.ContainerID,
		TargetPort:  opts.TargetPort,
		Label:       opts.Label,
		TaiID:       opts.TaiID,
		Mode:        mode,
		TargetAddr:  targetAddr,
		Cancel:      cancel,
		CreatedAt:   time.Now(),
		idleTimeout: wp.config.IdleTimeout,
	}

	// Start the HTTP listener with the already-open listener
	if err := binding.Start(ctx, ln); err != nil {
		cancel()
		ln.Close()
		wp.pool.Release(hostPort)
		return nil, fmt.Errorf("failed to start listener on port %d: %w", hostPort, err)
	}

	wp.bindings[hostPort] = binding
	return binding.Info(), nil
}

// Unbind stops a binding and releases its port.
func (wp *WebProxy) Unbind(hostPort int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	b, ok := wp.bindings[hostPort]
	if !ok {
		return fmt.Errorf("binding not found for port %d", hostPort)
	}

	b.Stop()
	delete(wp.bindings, hostPort)
	wp.pool.Release(hostPort)
	return nil
}

// UnbindAll removes all bindings for a given target.
func (wp *WebProxy) UnbindAll(targetID string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	for port, b := range wp.bindings {
		if b.TargetID == targetID {
			b.Stop()
			delete(wp.bindings, port)
			wp.pool.Release(port)
		}
	}
}

// List returns all bindings, optionally filtered by target_id.
func (wp *WebProxy) List(targetID string) []BindingInfo {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	var result []BindingInfo
	for _, b := range wp.bindings {
		if targetID != "" && b.TargetID != targetID {
			continue
		}
		result = append(result, *b.Info())
	}
	return result
}

// GetConfig returns the domain and prefix configuration for frontend URL construction.
func (wp *WebProxy) GetConfig() (domain, prefix string) {
	return wp.config.Domain, wp.config.Prefix
}

// allocatePort returns a port and an already-open net.Listener.
// Ports that fail to listen are released back to the pool (no leak).
func (wp *WebProxy) allocatePort() (int, net.Listener, error) {
	for {
		port, err := wp.pool.Allocate()
		if err != nil {
			return 0, nil, err
		}

		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			// Port occupied externally — release back to pool and try next
			wp.pool.Release(port)
			continue
		}
		return port, ln, nil
	}
}

// StartIdleReaper starts a goroutine that periodically checks for idle bindings.
func (wp *WebProxy) StartIdleReaper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				wp.reapIdle()
			}
		}
	}()
}

// reapIdle collects expired bindings under read lock, then stops them outside the lock.
func (wp *WebProxy) reapIdle() {
	now := time.Now()

	// Phase 1: collect expired ports under read lock
	wp.mu.RLock()
	var expired []int
	for port, b := range wp.bindings {
		if b.idleTimeout > 0 && now.Sub(b.LastActive()) > b.idleTimeout {
			expired = append(expired, port)
		}
	}
	wp.mu.RUnlock()

	if len(expired) == 0 {
		return
	}

	// Phase 2: stop and remove under write lock
	wp.mu.Lock()
	defer wp.mu.Unlock()
	for _, port := range expired {
		b, ok := wp.bindings[port]
		if !ok {
			continue
		}
		// Re-check under write lock in case of concurrent activity
		if now.Sub(b.LastActive()) > b.idleTimeout {
			b.Stop()
			delete(wp.bindings, port)
			wp.pool.Release(port)
		}
	}
}
