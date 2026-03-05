package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/yao/tai"
	taisandbox "github.com/yaoapp/yao/tai/sandbox"
)

// Manager manages a pool of tai.Client connections and sandbox lifecycle.
type Manager struct {
	pool        map[string]*tai.Client
	poolDefs    []Pool
	defaultPool string
	config      Config
	boxes       sync.Map
	mu          sync.Mutex
	cancel      context.CancelFunc
	grpcPort    int
}

func newManager(cfg Config) (*Manager, error) {
	m := &Manager{
		pool:     make(map[string]*tai.Client),
		poolDefs: cfg.Pool,
		config:   cfg,
		grpcPort: 9099,
	}
	if len(cfg.Pool) > 0 {
		m.defaultPool = cfg.Pool[0].Name
	}
	return m, nil
}

// Start discovers existing containers from all pools, rebuilds the boxes map,
// and starts the cleanup loop.
func (m *Manager) Start(ctx context.Context) error {
	if len(m.poolDefs) == 0 {
		return nil
	}

	for _, pd := range m.poolDefs {
		client, err := m.getPool(pd.Name)
		if err != nil {
			continue
		}
		m.recoverBoxes(ctx, &pd, client)
	}

	loopCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	go m.cleanupLoop(loopCtx)
	return nil
}

// AddPool registers a new pool at runtime.
func (m *Manager) AddPool(_ context.Context, p Pool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, pd := range m.poolDefs {
		if pd.Name == p.Name {
			return fmt.Errorf("sandbox: pool %q already exists", p.Name)
		}
	}
	m.poolDefs = append(m.poolDefs, p)
	if m.defaultPool == "" {
		m.defaultPool = p.Name
	}
	return nil
}

// RemovePool removes a pool by name.
func (m *Manager) RemovePool(ctx context.Context, name string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, pd := range m.poolDefs {
		if pd.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrPoolNotFound
	}

	count := 0
	m.boxes.Range(func(_, value any) bool {
		if value.(*Box).pool == name {
			count++
		}
		return true
	})

	if count > 0 && !force {
		return ErrPoolInUse
	}

	if count > 0 {
		m.boxes.Range(func(key, value any) bool {
			b := value.(*Box)
			if b.pool == name {
				b.Remove(ctx)
			}
			return true
		})
	}

	m.poolDefs = append(m.poolDefs[:idx], m.poolDefs[idx+1:]...)
	if client, ok := m.pool[name]; ok {
		client.Close()
		delete(m.pool, name)
	}
	return nil
}

// Pools returns all registered pool names and their status.
func (m *Manager) Pools() []PoolInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]PoolInfo, 0, len(m.poolDefs))
	for _, pd := range m.poolDefs {
		_, connected := m.pool[pd.Name]
		count := 0
		m.boxes.Range(func(_, value any) bool {
			if value.(*Box).pool == pd.Name {
				count++
			}
			return true
		})
		result = append(result, PoolInfo{
			Name:        pd.Name,
			Addr:        pd.Addr,
			Connected:   connected,
			Boxes:       count,
			MaxPerUser:  pd.MaxPerUser,
			MaxTotal:    pd.MaxTotal,
			IdleTimeout: pd.IdleTimeout,
			MaxLifetime: pd.MaxLifetime,
		})
	}
	return result
}

// Heartbeat updates the box's last heartbeat timestamp.
func (m *Manager) Heartbeat(sandboxID string, active bool, processCount int) error {
	v, ok := m.boxes.Load(sandboxID)
	if !ok {
		return ErrNotFound
	}
	b := v.(*Box)
	if active {
		b.lastHeartbeat.Store(time.Now().UnixMilli())
	}
	b.processCount.Store(int32(processCount))
	return nil
}

// Create creates and starts a new sandbox.
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Box, error) {
	if len(m.poolDefs) == 0 {
		return nil, ErrNotAvailable
	}
	if opts.Image == "" {
		return nil, fmt.Errorf("sandbox: image is required")
	}

	poolName := opts.Pool
	if poolName == "" {
		poolName = m.defaultPool
	}

	pd := m.findPoolDef(poolName)
	if pd == nil {
		return nil, ErrPoolNotFound
	}

	if err := m.checkLimits(pd, opts.Owner); err != nil {
		return nil, err
	}

	id := opts.ID
	if id == "" {
		id = fmt.Sprintf("sb-%d", time.Now().UnixNano())
	}

	client, err := m.getPool(poolName)
	if err != nil {
		return nil, fmt.Errorf("sandbox: connect pool %q: %w", poolName, err)
	}

	access, refresh, err := CreateContainerTokens(id, opts.Owner, nil)
	if err != nil {
		return nil, fmt.Errorf("sandbox: create tokens: %w", err)
	}

	taiOpts := m.buildTaiCreateOptions(opts, pd, id, access, refresh)

	containerID, err := client.Sandbox().Create(ctx, taiOpts)
	if err != nil {
		return nil, fmt.Errorf("sandbox: create container: %w", err)
	}

	if err := client.Sandbox().Start(ctx, containerID); err != nil {
		client.Sandbox().Remove(ctx, containerID, true)
		return nil, fmt.Errorf("sandbox: start container: %w", err)
	}

	policy := opts.Policy
	if policy == "" {
		policy = Session
	}

	box := &Box{
		id:           id,
		containerID:  containerID,
		pool:         poolName,
		owner:        opts.Owner,
		policy:       policy,
		labels:       opts.Labels,
		idleTimeoutD: opts.IdleTimeout,
		createdAt:    time.Now(),
		refreshToken: refresh,
		manager:      m,
		vnc:          opts.VNC,
		image:        opts.Image,
	}
	box.lastCall.Store(time.Now().UnixMilli())

	m.boxes.Store(id, box)
	return box, nil
}

// Get returns an existing sandbox by ID.
func (m *Manager) Get(_ context.Context, id string) (*Box, error) {
	v, ok := m.boxes.Load(id)
	if !ok {
		return nil, ErrNotFound
	}
	return v.(*Box), nil
}

// GetOrCreate returns existing sandbox by ID or creates a new one.
func (m *Manager) GetOrCreate(ctx context.Context, opts CreateOptions) (*Box, error) {
	if opts.ID != "" {
		if v, ok := m.boxes.Load(opts.ID); ok {
			return v.(*Box), nil
		}
	}
	return m.Create(ctx, opts)
}

// List returns all sandboxes, optionally filtered.
func (m *Manager) List(_ context.Context, opts ListOptions) ([]*Box, error) {
	var result []*Box
	m.boxes.Range(func(_, value any) bool {
		b := value.(*Box)
		if opts.Owner != "" && b.owner != opts.Owner {
			return true
		}
		if opts.Pool != "" && b.pool != opts.Pool {
			return true
		}
		if len(opts.Labels) > 0 {
			for k, v := range opts.Labels {
				if b.labels[k] != v {
					return true
				}
			}
		}
		result = append(result, b)
		return true
	})
	return result, nil
}

// Remove stops and removes a sandbox.
func (m *Manager) Remove(ctx context.Context, id string) error {
	v, ok := m.boxes.Load(id)
	if !ok {
		return ErrNotFound
	}
	b := v.(*Box)

	client, err := m.getPool(b.pool)
	if err == nil {
		client.Sandbox().Stop(ctx, b.containerID, 10*time.Second)
		client.Sandbox().Remove(ctx, b.containerID, true)
	}

	if b.refreshToken != "" {
		RevokeContainerTokens(b.refreshToken)
	}

	m.boxes.Delete(id)
	return nil
}

// Cleanup removes idle/expired sandboxes.
func (m *Manager) Cleanup(ctx context.Context) error {
	now := time.Now()
	m.boxes.Range(func(key, value any) bool {
		b := value.(*Box)
		idle := now.Sub(b.lastActiveTime())

		switch b.policy {
		case OneShot:
			// handled after Exec
		case Session:
			if timeout := b.idleTimeout(); timeout > 0 && idle > timeout {
				m.Remove(ctx, b.id)
			}
		case LongRunning:
			if timeout := b.idleTimeout(); timeout > 0 && idle > timeout {
				if client, err := m.getPool(b.pool); err == nil {
					client.Sandbox().Stop(ctx, b.containerID, 10*time.Second)
				}
			}
			if lifetime := b.maxLifetime(); lifetime > 0 && now.Sub(b.createdAt) > lifetime {
				m.Remove(ctx, b.id)
			}
		case Persistent:
			// never auto-cleaned
		}
		return true
	})
	return nil
}

// Close stops the cleanup loop and releases all pool connections.
func (m *Manager) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, client := range m.pool {
		client.Close()
		delete(m.pool, name)
	}
	return nil
}

// SetGRPCPort sets the local gRPC port for container env injection.
func (m *Manager) SetGRPCPort(port int) {
	m.grpcPort = port
}

func (m *Manager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.Cleanup(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) getPool(name string) (*tai.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.pool[name]; ok {
		return client, nil
	}

	pd := m.findPoolDefLocked(name)
	if pd == nil {
		return nil, ErrPoolNotFound
	}

	client, err := tai.New(pd.Addr, pd.Options...)
	if err != nil {
		return nil, err
	}
	m.pool[name] = client
	return client, nil
}

func (m *Manager) findPoolDef(name string) *Pool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.findPoolDefLocked(name)
}

func (m *Manager) findPoolDefLocked(name string) *Pool {
	for i := range m.poolDefs {
		if m.poolDefs[i].Name == name {
			return &m.poolDefs[i]
		}
	}
	return nil
}

func (m *Manager) checkLimits(pd *Pool, owner string) error {
	if pd.MaxTotal > 0 {
		count := 0
		m.boxes.Range(func(_, value any) bool {
			if value.(*Box).pool == pd.Name {
				count++
			}
			return true
		})
		if count >= pd.MaxTotal {
			return ErrLimitExceeded
		}
	}

	if pd.MaxPerUser > 0 && owner != "" {
		count := 0
		m.boxes.Range(func(_, value any) bool {
			b := value.(*Box)
			if b.pool == pd.Name && b.owner == owner {
				count++
			}
			return true
		})
		if count >= pd.MaxPerUser {
			return ErrLimitExceeded
		}
	}
	return nil
}

func (m *Manager) buildTaiCreateOptions(opts CreateOptions, pd *Pool, sandboxID, access, refresh string) taisandbox.CreateOptions {
	env := make(map[string]string)
	for k, v := range opts.Env {
		env[k] = v
	}
	grpcEnv := BuildGRPCEnv(pd, sandboxID, access, refresh, m.grpcPort)
	for k, v := range grpcEnv {
		env[k] = v
	}

	labels := map[string]string{
		"managed-by":     "yao-sandbox",
		"sandbox-id":     sandboxID,
		"sandbox-owner":  opts.Owner,
		"sandbox-pool":   pd.Name,
		"sandbox-policy": string(opts.Policy),
	}
	for k, v := range opts.Labels {
		labels[k] = v
	}

	workDir := opts.WorkDir
	if workDir == "" {
		workDir = "/workspace"
	}

	cmd := []string{"sleep", "infinity"}

	var ports []taisandbox.PortMapping
	for _, p := range opts.Ports {
		ports = append(ports, taisandbox.PortMapping{
			ContainerPort: p.ContainerPort,
			HostPort:      p.HostPort,
			HostIP:        p.HostIP,
			Protocol:      p.Protocol,
		})
	}

	return taisandbox.CreateOptions{
		Name:       sandboxID,
		Image:      opts.Image,
		Cmd:        cmd,
		Env:        env,
		WorkingDir: workDir,
		User:       opts.User,
		Memory:     opts.Memory,
		CPUs:       opts.CPUs,
		VNC:        opts.VNC,
		Ports:      ports,
		Labels:     labels,
	}
}

func (m *Manager) recoverBoxes(ctx context.Context, pd *Pool, client *tai.Client) {
	containers, err := client.Sandbox().List(ctx, taisandbox.ListOptions{
		All:    true,
		Labels: map[string]string{"managed-by": "yao-sandbox"},
	})
	if err != nil {
		return
	}

	for _, c := range containers {
		sandboxID := c.Labels["sandbox-id"]
		if sandboxID == "" {
			continue
		}
		if _, loaded := m.boxes.Load(sandboxID); loaded {
			continue
		}

		box := &Box{
			id:          sandboxID,
			containerID: c.ID,
			pool:        c.Labels["sandbox-pool"],
			owner:       c.Labels["sandbox-owner"],
			policy:      LifecyclePolicy(c.Labels["sandbox-policy"]),
			labels:      c.Labels,
			createdAt:   time.Now(),
			image:       c.Image,
			manager:     m,
		}
		box.lastCall.Store(time.Now().UnixMilli())
		m.boxes.Store(sandboxID, box)
	}
}
