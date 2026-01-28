package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/robot/cache"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/trigger"
	"github.com/yaoapp/yao/agent/robot/types"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// Default configuration values
const (
	DefaultTickInterval = time.Minute // default tick interval for clock checking
)

// Config holds manager configuration
type Config struct {
	TickInterval time.Duration  // how often to check clock triggers (default: 1 minute)
	PoolConfig   *pool.Config   // worker pool configuration
	Executor     types.Executor // optional: custom executor (default: real executor)
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
	executor types.Executor

	// Execution control for pause/resume/stop
	execController *trigger.ExecutionController

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
	ec := trigger.NewExecutionController()

	// Use custom executor if provided, otherwise create default
	var e types.Executor
	if config.Executor != nil {
		e = config.Executor
	} else {
		e = executor.New()
	}

	// Wire up pool with executor
	p.SetExecutor(e)

	// Create shared executor instances for each mode
	// These are reused across all executions to maintain accurate counters
	dryRunExecutor := executor.NewDryRun()

	// Set executor factory for mode-specific executors
	p.SetExecutorFactory(func(mode types.ExecutorMode) types.Executor {
		switch mode {
		case types.ExecutorDryRun:
			return dryRunExecutor
		case types.ExecutorSandbox:
			// Sandbox not implemented, fall back to DryRun
			return dryRunExecutor
		default:
			// Standard mode or empty - use the configured executor
			return e
		}
	})

	return &Manager{
		config:         config,
		cache:          c,
		pool:           p,
		executor:       e,
		execController: ec,
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

	// Set completion callback to clean up ExecutionController when execution finishes
	m.pool.SetOnComplete(func(execID, memberID string, status types.ExecStatus) {
		// Remove from ExecutionController (cleans up in-memory tracking)
		m.execController.Untrack(execID)
		// Remove from robot's in-memory execution list
		if robot := m.cache.Get(memberID); robot != nil {
			robot.RemoveExecution(execID)
		}
	})

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
			// Perform tick - context is created per-robot in Tick()
			_ = m.Tick(m.ctx, now)
		}
	}
}

// Tick processes a clock tick
// 1. Get all cached robots
// 2. For each robot with clock trigger enabled
// 3. Check if should execute based on clock config
// 4. Submit to pool with robot's own identity
func (m *Manager) Tick(parentCtx context.Context, now time.Time) error {
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

		// Pre-generate execution ID and track for pause/resume/stop
		// We need to track BEFORE submit so we can pass the cancellable context to the executor
		execID := pool.GenerateExecID()
		ctrlExec := m.execController.Track(execID, robot.MemberID, robot.TeamID)

		// Create context with robot's own identity and cancellable context
		// Clock-triggered executions run as the robot itself
		robotAuth := m.buildRobotAuth(robot)
		execCtx := types.NewContext(ctrlExec.Context(), robotAuth)

		// Create clock context for P0 inspiration
		clockCtx := types.NewClockContext(now, robot.Config.Clock.TZ)

		// Submit to pool with the cancellable context and execution control
		_, err := m.pool.SubmitWithID(execCtx, robot, types.TriggerClock, clockCtx, execID, ctrlExec)
		if err != nil {
			// If submission failed, untrack the execution
			m.execController.Untrack(execID)
			// Log error but continue with other robots
			// In production, this would be logged properly
			continue
		}

		// Update robot's last run time
		robot.LastRun = now
	}

	return nil
}

// buildRobotAuth creates AuthorizedInfo for a robot's own identity
// Used when robot executes autonomously (clock trigger)
func (m *Manager) buildRobotAuth(robot *types.Robot) *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: robot.MemberID,
		TeamID: robot.TeamID,
		// ClientID could be set to a special "robot-agent" identifier if needed
		ClientID: "robot-agent",
	}
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
				lastRunInLoc := robot.LastRun.In(now.Location())
				if lastRunInLoc.Format("15:04") == currentTime && lastRunInLoc.Day() == now.Day() {
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
// For non-autonomous robots: lazy-loads from DB, executes, then unloads
func (m *Manager) TriggerManual(ctx *types.Context, memberID string, trigger types.TriggerType, data interface{}) (string, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return "", fmt.Errorf("manager not started")
	}
	m.mu.RUnlock()

	// Get robot from cache, or lazy-load if not found
	robot, lazyLoaded, err := m.getOrLoadRobot(ctx, memberID)
	if err != nil {
		return "", err
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

	// Pre-generate execution ID and track for pause/resume/stop
	// We need to track BEFORE submit so we can pass the cancellable context to the executor
	execID := pool.GenerateExecID()
	ctrlExec := m.execController.Track(execID, memberID, robot.TeamID)

	// Create a new context with the cancellable context from ExecutionController
	// This allows Stop() to propagate cancellation to the executor
	execCtx := types.NewContext(ctrlExec.Context(), ctx.Auth)

	// Submit to pool with the cancellable context and execution control
	// The control interface allows executor to check pause state and wait if paused
	_, err = m.pool.SubmitWithID(execCtx, robot, trigger, data, execID, ctrlExec)
	if err != nil {
		// If submission failed, untrack the execution
		m.execController.Untrack(execID)
		// If lazy-loaded and submission failed, remove from cache
		if lazyLoaded {
			m.cache.Remove(memberID)
		}
		return "", err
	}

	// For lazy-loaded robots, schedule cleanup after execution completes
	if lazyLoaded {
		m.scheduleCleanup(robot)
	}

	return execID, nil
}

// ==================== Human Intervention & Event Triggers ====================

// Intervene processes a human intervention request
// Human intervention skips P0 (inspiration) and goes directly to P1 (goals)
// For non-autonomous robots: lazy-loads from DB, executes, then unloads
func (m *Manager) Intervene(ctx *types.Context, req *types.InterveneRequest) (*types.ExecutionResult, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return nil, fmt.Errorf("manager not started")
	}
	m.mu.RUnlock()

	// Validate request
	if err := trigger.ValidateIntervention(req); err != nil {
		return nil, err
	}

	// Get robot from cache, or lazy-load if not found
	robot, lazyLoaded, err := m.getOrLoadRobot(ctx, req.MemberID)
	if err != nil {
		return nil, err
	}

	// Check robot status
	if robot.Status == types.RobotPaused {
		return nil, types.ErrRobotPaused
	}

	// Check if human trigger is enabled
	if robot.Config != nil && robot.Config.Triggers != nil {
		if !robot.Config.Triggers.IsEnabled(types.TriggerHuman) {
			return nil, types.ErrTriggerDisabled
		}
	}

	// Build trigger input
	triggerInput := &types.TriggerInput{
		Action:   req.Action,
		Messages: req.Messages,
		UserID:   ctx.UserID(),
	}

	// Handle plan.add action - schedule for later
	if req.Action == types.ActionPlanAdd && req.PlanTime != nil {
		// If lazy-loaded but not executing, remove immediately
		if lazyLoaded {
			m.cache.Remove(req.MemberID)
		}
		// TODO: Add to plan queue (Phase 11.3)
		return &types.ExecutionResult{
			Status:  types.ExecPending,
			Message: fmt.Sprintf("Planned for %s (plan queue not implemented yet)", req.PlanTime.Format(time.RFC3339)),
		}, nil
	}

	// Determine executor mode: request > robot config > default
	executorMode := m.resolveExecutorMode(req.ExecutorMode, robot)

	// Submit to pool with executor mode
	execID, err := m.pool.SubmitWithMode(ctx, robot, types.TriggerHuman, triggerInput, executorMode)
	if err != nil {
		// If lazy-loaded and submission failed, remove from cache
		if lazyLoaded {
			m.cache.Remove(req.MemberID)
		}
		return nil, err
	}

	// Track execution for pause/resume/stop
	m.execController.Track(execID, req.MemberID, req.TeamID)

	// For lazy-loaded robots, schedule cleanup after execution completes
	if lazyLoaded {
		m.scheduleCleanup(robot)
	}

	return &types.ExecutionResult{
		ExecutionID: execID,
		Status:      types.ExecPending,
		Message:     fmt.Sprintf("Human intervention (%s) submitted", req.Action),
	}, nil
}

// HandleEvent processes an event trigger request
// Event trigger skips P0 (inspiration) and goes directly to P1 (goals)
// For non-autonomous robots: lazy-loads from DB, executes, then unloads
func (m *Manager) HandleEvent(ctx *types.Context, req *types.EventRequest) (*types.ExecutionResult, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return nil, fmt.Errorf("manager not started")
	}
	m.mu.RUnlock()

	// Validate request
	if err := trigger.ValidateEvent(req); err != nil {
		return nil, err
	}

	// Get robot from cache, or lazy-load if not found
	robot, lazyLoaded, err := m.getOrLoadRobot(ctx, req.MemberID)
	if err != nil {
		return nil, err
	}

	// Check robot status
	if robot.Status == types.RobotPaused {
		return nil, types.ErrRobotPaused
	}

	// Check if event trigger is enabled
	if robot.Config != nil && robot.Config.Triggers != nil {
		if !robot.Config.Triggers.IsEnabled(types.TriggerEvent) {
			return nil, types.ErrTriggerDisabled
		}
	}

	// Build trigger input
	triggerInput := trigger.BuildEventInput(req)

	// Determine executor mode: request > robot config > default
	executorMode := m.resolveExecutorMode(req.ExecutorMode, robot)

	// Submit to pool with executor mode
	execID, err := m.pool.SubmitWithMode(ctx, robot, types.TriggerEvent, triggerInput, executorMode)
	if err != nil {
		// If lazy-loaded and submission failed, remove from cache
		if lazyLoaded {
			m.cache.Remove(req.MemberID)
		}
		return nil, err
	}

	// Track execution for pause/resume/stop
	m.execController.Track(execID, req.MemberID, "")

	// For lazy-loaded robots, schedule cleanup after execution completes
	if lazyLoaded {
		m.scheduleCleanup(robot)
	}

	return &types.ExecutionResult{
		ExecutionID: execID,
		Status:      types.ExecPending,
		Message:     fmt.Sprintf("Event trigger (%s: %s) submitted", req.Source, req.EventType),
	}, nil
}

// ==================== Execution Control ====================

// PauseExecution pauses a running execution
func (m *Manager) PauseExecution(ctx *types.Context, execID string) error {
	// Get execution info before pausing
	exec := m.execController.Get(execID)
	if exec == nil {
		return fmt.Errorf("execution not found: %s", execID)
	}

	// Pause the execution
	if err := m.execController.Pause(execID); err != nil {
		return err
	}

	// Remove from robot's in-memory execution list (paused doesn't count as running)
	if robot := m.cache.Get(exec.MemberID); robot != nil {
		robot.RemoveExecution(execID)
	}

	return nil
}

// ResumeExecution resumes a paused execution
func (m *Manager) ResumeExecution(ctx *types.Context, execID string) error {
	// Get execution info before resuming
	exec := m.execController.Get(execID)
	if exec == nil {
		return fmt.Errorf("execution not found: %s", execID)
	}

	// Resume the execution
	if err := m.execController.Resume(execID); err != nil {
		return err
	}

	// Add back to robot's in-memory execution list
	if robot := m.cache.Get(exec.MemberID); robot != nil {
		robot.AddExecution(&types.Execution{
			ID:       execID,
			MemberID: exec.MemberID,
			TeamID:   exec.TeamID,
			Status:   types.ExecRunning,
		})
	}

	return nil
}

// StopExecution stops a running execution
func (m *Manager) StopExecution(ctx *types.Context, execID string) error {
	// Get execution info before stopping
	exec := m.execController.Get(execID)
	if exec == nil {
		return fmt.Errorf("execution not found: %s", execID)
	}

	// Stop the execution
	if err := m.execController.Stop(execID); err != nil {
		return err
	}

	// Remove from robot's in-memory execution list
	if robot := m.cache.Get(exec.MemberID); robot != nil {
		robot.RemoveExecution(execID)
	}

	return nil
}

// GetExecutionStatus returns the status of an execution
func (m *Manager) GetExecutionStatus(execID string) (*trigger.ControlledExecution, error) {
	exec := m.execController.Get(execID)
	if exec == nil {
		return nil, fmt.Errorf("execution not found: %s", execID)
	}
	return exec, nil
}

// ListExecutions returns all tracked executions
func (m *Manager) ListExecutions() []*trigger.ControlledExecution {
	return m.execController.List()
}

// ListExecutionsByMember returns all executions for a specific robot
func (m *Manager) ListExecutionsByMember(memberID string) []*trigger.ControlledExecution {
	return m.execController.ListByMember(memberID)
}

// ==================== Helper Methods ====================

// getOrLoadRobot gets a robot from cache, or lazy-loads from DB if not found
// Returns: robot, wasLazyLoaded, error
func (m *Manager) getOrLoadRobot(ctx *types.Context, memberID string) (*types.Robot, bool, error) {
	// Try cache first
	robot := m.cache.Get(memberID)
	if robot != nil {
		return robot, false, nil
	}

	// Not in cache - lazy load from database
	robot, err := m.cache.LoadByID(ctx, memberID)
	if err != nil {
		return nil, false, err
	}

	// Add to cache temporarily for execution tracking
	m.cache.Add(robot)

	// Return with lazyLoaded=true to indicate cleanup needed after execution
	return robot, true, nil
}

// scheduleCleanup schedules removal of a lazy-loaded robot after all executions complete
// This runs in a goroutine that monitors the robot's execution count
func (m *Manager) scheduleCleanup(robot *types.Robot) {
	go func() {
		memberID := robot.MemberID

		// Poll every 5 seconds to check if all executions are done
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Timeout after 24 hours to prevent memory leaks
		timeout := time.After(24 * time.Hour)

		for {
			select {
			case <-timeout:
				// Timeout - force cleanup
				m.cache.Remove(memberID)
				return

			case <-ticker.C:
				// Check if robot still exists in cache
				r := m.cache.Get(memberID)
				if r == nil {
					// Already removed
					return
				}

				// Check if all executions are done
				if r.RunningCount() == 0 {
					// Only remove if still non-autonomous
					// (user might have changed it during execution)
					if !r.AutonomousMode {
						m.cache.Remove(memberID)
					}
					return
				}
			}
		}
	}()
}

// resolveExecutorMode determines the executor mode to use
// Priority: request > robot config > default (standard)
func (m *Manager) resolveExecutorMode(requestMode types.ExecutorMode, robot *types.Robot) types.ExecutorMode {
	// Request mode takes precedence
	if requestMode != "" && requestMode.IsValid() {
		return requestMode
	}

	// Robot config mode
	if robot != nil && robot.Config != nil && robot.Config.Executor != nil {
		return robot.Config.Executor.GetMode()
	}

	// Default: standard
	return types.ExecutorStandard
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
func (m *Manager) Executor() types.Executor {
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
