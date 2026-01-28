package pool

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/robot/utils"
)

// Default configuration values
const (
	DefaultWorkerSize = 10  // default number of workers
	DefaultQueueSize  = 100 // default global queue size
)

// Config holds pool configuration
type Config struct {
	WorkerSize int // number of workers (default: 10)
	QueueSize  int // global queue size (default: 100)
}

// DefaultConfig returns default pool configuration
func DefaultConfig() *Config {
	return &Config{
		WorkerSize: DefaultWorkerSize,
		QueueSize:  DefaultQueueSize,
	}
}

// ExecutorFactory creates an executor based on the mode
type ExecutorFactory func(mode types.ExecutorMode) types.Executor

// OnCompleteCallback is called when an execution completes (success or failure)
// Parameters: execID, memberID, status
type OnCompleteCallback func(execID, memberID string, status types.ExecStatus)

// Pool implements types.Pool interface
// Manages a pool of workers that execute robot jobs from a priority queue
type Pool struct {
	size            int                // number of workers
	queue           *PriorityQueue     // priority queue for pending jobs
	executor        types.Executor     // default executor for running jobs
	executorFactory ExecutorFactory    // optional: factory for mode-specific executors
	onComplete      OnCompleteCallback // optional: callback when execution completes
	workers         []*Worker          // worker goroutines
	running         atomic.Int32       // number of currently running jobs
	wg              sync.WaitGroup     // wait group for graceful shutdown
	started         bool               // whether pool has been started
	mu              sync.RWMutex       // protects started flag
}

// New creates a new pool instance with default configuration
func New() *Pool {
	return NewWithConfig(nil)
}

// NewWithConfig creates a new pool instance with custom configuration
func NewWithConfig(config *Config) *Pool {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply defaults for zero values
	workerSize := config.WorkerSize
	if workerSize <= 0 {
		workerSize = DefaultWorkerSize
	}

	queueSize := config.QueueSize
	if queueSize <= 0 {
		queueSize = DefaultQueueSize
	}

	return &Pool{
		size:  workerSize,
		queue: NewPriorityQueue(queueSize),
	}
}

// SetExecutor sets the default executor for the pool
// Must be called before Start()
func (p *Pool) SetExecutor(executor types.Executor) {
	p.executor = executor
}

// SetExecutorFactory sets the executor factory for mode-specific executors
// If set, the factory is used to create executors based on ExecutorMode
func (p *Pool) SetExecutorFactory(factory ExecutorFactory) {
	p.executorFactory = factory
}

// SetOnComplete sets the callback for execution completion
// Called when an execution finishes (completed, failed, or cancelled)
func (p *Pool) SetOnComplete(callback OnCompleteCallback) {
	p.onComplete = callback
}

// GetExecutor returns the appropriate executor for the given mode
// If factory is set and mode is specified, uses factory; otherwise uses default
func (p *Pool) GetExecutor(mode types.ExecutorMode) types.Executor {
	// If factory is set and mode is specified, use factory
	if p.executorFactory != nil && mode != "" {
		return p.executorFactory(mode)
	}
	// Otherwise use default executor
	return p.executor
}

// Start starts the worker pool
func (p *Pool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return fmt.Errorf("pool already started")
	}

	if p.executor == nil {
		return fmt.Errorf("executor not set, call SetExecutor() first")
	}

	// Create and start workers
	p.workers = make([]*Worker, p.size)
	for i := 0; i < p.size; i++ {
		worker := newWorker(i+1, p, &p.wg)
		p.workers[i] = worker
		worker.start()
	}

	p.started = true
	return nil
}

// Stop stops the worker pool gracefully
// Waits for all running jobs to complete
func (p *Pool) Stop() error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil // already stopped or never started
	}
	p.started = false
	p.mu.Unlock()

	// Stop all workers
	for _, worker := range p.workers {
		worker.stop()
	}

	// Wait for all workers to finish
	p.wg.Wait()

	return nil
}

// Submit submits a robot execution to the pool
// Returns execution ID if successfully queued, error otherwise
func (p *Pool) Submit(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (string, error) {
	return p.SubmitWithMode(ctx, robot, trigger, data, "")
}

// GenerateExecID generates a new execution ID
// Exported so Manager can pre-generate IDs for tracking
func GenerateExecID() string {
	return utils.NewID()
}

// SubmitWithMode submits a robot execution with specified executor mode
// executorMode: optional, overrides robot's config if provided
// Returns execution ID if successfully queued, error otherwise
// Note: This method does not support execution control (pause/resume)
func (p *Pool) SubmitWithMode(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, executorMode types.ExecutorMode) (string, error) {
	execID := GenerateExecID()
	return p.submitWithIDAndMode(ctx, robot, trigger, data, execID, executorMode, nil)
}

// SubmitWithID submits a robot execution with a pre-generated execution ID
// This is used when the caller needs to track the execution before submission
func (p *Pool) SubmitWithID(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string, control types.ExecutionControl) (string, error) {
	return p.submitWithIDAndMode(ctx, robot, trigger, data, execID, "", control)
}

// submitWithIDAndMode is the internal implementation that handles both cases
func (p *Pool) submitWithIDAndMode(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string, executorMode types.ExecutorMode, control types.ExecutionControl) (string, error) {
	p.mu.RLock()
	if !p.started {
		p.mu.RUnlock()
		return "", fmt.Errorf("pool not started")
	}
	p.mu.RUnlock()

	if robot == nil {
		return "", fmt.Errorf("robot cannot be nil")
	}

	// Create queue item with the provided ID and control
	item := &QueueItem{
		Robot:        robot,
		Ctx:          ctx,
		Trigger:      trigger,
		Data:         data,
		ExecutorMode: executorMode,
		ExecID:       execID,
		Control:      control,
	}

	// Try to add to queue
	if !p.queue.Enqueue(item) {
		return "", fmt.Errorf("queue full (max %d items)", p.queue.maxSize)
	}

	return execID, nil
}

// Running returns number of currently running jobs
func (p *Pool) Running() int {
	return int(p.running.Load())
}

// Queued returns number of queued jobs
func (p *Pool) Queued() int {
	return p.queue.Size()
}

// incrementRunning increments the running counter
func (p *Pool) incrementRunning() {
	p.running.Add(1)
}

// decrementRunning decrements the running counter
func (p *Pool) decrementRunning() {
	p.running.Add(-1)
}

// Size returns the configured pool size
func (p *Pool) Size() int {
	return p.size
}

// QueueSize returns the configured queue size
func (p *Pool) QueueSize() int {
	return p.queue.maxSize
}

// IsStarted returns true if the pool has been started
func (p *Pool) IsStarted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.started
}
