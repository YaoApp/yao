package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/robot/cache"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
)

// Default configuration values
const (
	DefaultTickInterval = time.Minute // default tick interval for clock checking
)

// Config holds manager configuration
type Config struct {
	TickInterval time.Duration // how often to check clock triggers (default: 1 minute)
	PoolConfig   *pool.Config  // worker pool configuration
}

// DefaultConfig returns default manager configuration
func DefaultConfig() *Config {
	return &Config{
		TickInterval: DefaultTickInterval,
		PoolConfig:   pool.DefaultConfig(),
	}
}

// Manager implements types.Manager interface
// Orchestrates the robot scheduling system: Cache -> Dedup -> Pool -> Executor
type Manager struct {
	config   *Config
	cache    *cache.Cache
	pool     *pool.Pool
	executor *executor.Executor

	// Ticker for clock trigger checking
	ticker     *time.Ticker
	tickerDone chan struct{}

	// State
	started bool
	mu      sync.RWMutex

	// Context for background operations
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new manager instance with default configuration
func New() *Manager {
	return NewWithConfig(nil)
}

// NewWithConfig creates a new manager instance with custom configuration
func NewWithConfig(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply defaults for zero values
	if config.TickInterval <= 0 {
		config.TickInterval = DefaultTickInterval
	}

	// Create components
	c := cache.New()
	p := pool.NewWithConfig(config.PoolConfig)
	e := executor.New()

	// Wire up pool with executor
	p.SetExecutor(e)

	return &Manager{
		config:   config,
		cache:    c,
		pool:     p,
		executor: e,
	}
}

// Start starts the manager
// 1. Load robots into cache
// 2. Start worker pool
// 3. Start clock ticker goroutine
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("manager already started")
	}

	// Create background context
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Load robots into cache
	ctx := types.NewContext(m.ctx, nil)
	if err := m.cache.Load(ctx); err != nil {
		return fmt.Errorf("failed to load robots: %w", err)
	}

	// Start worker pool
	if err := m.pool.Start(); err != nil {
		return fmt.Errorf("failed to start pool: %w", err)
	}

	// Start clock ticker
	m.ticker = time.NewTicker(m.config.TickInterval)
	m.tickerDone = make(chan struct{})

	go m.tickerLoop()

	// Start cache auto-refresh (every hour)
	m.cache.StartAutoRefresh(ctx, nil)

	m.started = true
	return nil
}

// Stop stops the manager gracefully
// 1. Stop clock ticker
// 2. Stop cache auto-refresh
// 3. Stop worker pool (waits for running jobs)
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return nil
	}
	m.started = false
	m.mu.Unlock()

	// Stop ticker
	if m.tickerDone != nil {
		close(m.tickerDone)
	}

	// Stop cache auto-refresh
	m.cache.StopAutoRefresh()

	// Stop pool (waits for running jobs)
	if err := m.pool.Stop(); err != nil {
		return fmt.Errorf("failed to stop pool: %w", err)
	}

	// Cancel background context
	if m.cancel != nil {
		m.cancel()
	}

	return nil
}

// tickerLoop is the main ticker goroutine
func (m *Manager) tickerLoop() {
	for {
		select {
		case <-m.tickerDone:
			m.ticker.Stop()
			return
		case now := <-m.ticker.C:
			// Perform tick
			ctx := types.NewContext(m.ctx, nil)
			_ = m.Tick(ctx, now)
		}
	}
}

// Tick processes a clock tick
// 1. Get all cached robots
// 2. For each robot with clock trigger enabled
// 3. Check if should execute based on clock config
// 4. Submit to pool
func (m *Manager) Tick(ctx *types.Context, now time.Time) error {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	// Get all cached robots
	robots := m.cache.ListAll()

	for _, robot := range robots {
		// Skip if robot is not active
		if robot.Status == types.RobotPaused || robot.Status == types.RobotError || robot.Status == types.RobotMaintenance {
			continue
		}

		// Skip if clock trigger is disabled
		if robot.Config == nil || robot.Config.Triggers == nil {
			continue
		}
		if !robot.Config.Triggers.IsEnabled(types.TriggerClock) {
			continue
		}

		// Skip if no clock config
		if robot.Config.Clock == nil {
			continue
		}

		// Check if should trigger based on clock config
		if !m.shouldTrigger(robot, now) {
			continue
		}

		// TODO: dedup check (Phase 11.1)
		// result, err := m.dedup.Check(ctx, robot.MemberID, types.TriggerClock)
		// if err != nil || result == types.DedupSkip {
		//     continue
		// }

		// Create clock context for P0 inspiration
		clockCtx := types.NewClockContext(now, robot.Config.Clock.TZ)

		// Submit to pool
		_, err := m.pool.Submit(ctx, robot, types.TriggerClock, clockCtx)
		if err != nil {
			// Log error but continue with other robots
			// In production, this would be logged properly
			continue
		}

		// Update robot's last run time
		robot.LastRun = now
	}

	return nil
}

// shouldTrigger checks if a robot should be triggered based on its clock config
func (m *Manager) shouldTrigger(robot *types.Robot, now time.Time) bool {
	clock := robot.Config.Clock
	if clock == nil {
		return false
	}

	// Get time in robot's timezone
	loc := clock.GetLocation()
	localNow := now.In(loc)

	switch clock.Mode {
	case types.ClockTimes:
		return m.shouldTriggerTimes(robot, clock, localNow)
	case types.ClockInterval:
		return m.shouldTriggerInterval(robot, clock, localNow)
	case types.ClockDaemon:
		return m.shouldTriggerDaemon(robot, clock, localNow)
	default:
		return false
	}
}

// shouldTriggerTimes checks if current time matches any configured times
// times mode: run at specific times (e.g., ["09:00", "14:00", "17:00"])
func (m *Manager) shouldTriggerTimes(robot *types.Robot, clock *types.Clock, now time.Time) bool {
	// Check day of week first
	if !m.matchesDay(clock, now) {
		return false
	}

	// Check if current time matches any configured time
	currentTime := now.Format("15:04")
	for _, t := range clock.Times {
		if t == currentTime {
			// Check if already triggered in this minute
			if !robot.LastRun.IsZero() {
				lastRunMinute := robot.LastRun.In(now.Location()).Format("15:04")
				if lastRunMinute == currentTime && robot.LastRun.Day() == now.Day() {
					return false // Already triggered this minute today
				}
			}
			return true
		}
	}
	return false
}

// shouldTriggerInterval checks if enough time has passed since last run
// interval mode: run every X duration (e.g., "30m", "2h")
func (m *Manager) shouldTriggerInterval(robot *types.Robot, clock *types.Clock, now time.Time) bool {
	interval, err := time.ParseDuration(clock.Every)
	if err != nil {
		return false
	}

	// First run if never executed
	if robot.LastRun.IsZero() {
		return true
	}

	// Check if interval has passed
	return now.Sub(robot.LastRun) >= interval
}

// shouldTriggerDaemon checks if robot can restart immediately after last run
// daemon mode: restart immediately after each run completes
func (m *Manager) shouldTriggerDaemon(robot *types.Robot, clock *types.Clock, now time.Time) bool {
	// Daemon mode: trigger if not currently running
	// CanRun() checks if robot has available execution slots
	return robot.CanRun()
}

// matchesDay checks if current day matches the configured days
func (m *Manager) matchesDay(clock *types.Clock, now time.Time) bool {
	// Empty days or ["*"] means all days
	if len(clock.Days) == 0 {
		return true
	}

	for _, day := range clock.Days {
		if day == "*" {
			return true
		}
		// Match day name (Mon, Tue, Wed, Thu, Fri, Sat, Sun)
		// or full name (Monday, Tuesday, etc.)
		weekday := now.Weekday().String()
		shortDay := weekday[:3] // Mon, Tue, etc.
		if day == weekday || day == shortDay {
			return true
		}
	}
	return false
}

// TriggerManual manually triggers a robot execution (for testing or API calls)
// This bypasses clock checking and directly submits to pool
func (m *Manager) TriggerManual(ctx *types.Context, memberID string, trigger types.TriggerType, data interface{}) (string, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return "", fmt.Errorf("manager not started")
	}
	m.mu.RUnlock()

	// Get robot from cache
	robot := m.cache.Get(memberID)
	if robot == nil {
		return "", types.ErrRobotNotFound
	}

	// Check robot status
	if robot.Status == types.RobotPaused {
		return "", types.ErrRobotPaused
	}

	// Check if trigger type is enabled
	if robot.Config != nil && robot.Config.Triggers != nil {
		if !robot.Config.Triggers.IsEnabled(trigger) {
			return "", types.ErrTriggerDisabled
		}
	}

	// Submit to pool
	execID, err := m.pool.Submit(ctx, robot, trigger, data)
	if err != nil {
		return "", err
	}

	return execID, nil
}

// ==================== Getters for internal components ====================
// These are exposed for testing and advanced use cases

// Cache returns the internal cache
func (m *Manager) Cache() *cache.Cache {
	return m.cache
}

// Pool returns the internal pool
func (m *Manager) Pool() *pool.Pool {
	return m.pool
}

// Executor returns the internal executor
func (m *Manager) Executor() *executor.Executor {
	return m.executor
}

// IsStarted returns true if manager is started
func (m *Manager) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// Running returns number of currently running jobs
func (m *Manager) Running() int {
	return m.pool.Running()
}

// Queued returns number of queued jobs
func (m *Manager) Queued() int {
	return m.pool.Queued()
}

// CachedRobots returns number of cached robots
func (m *Manager) CachedRobots() int {
	return m.cache.Count()
}
