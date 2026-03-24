package pool

import (
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/robot/logger"
	"github.com/yaoapp/yao/agent/robot/types"
)

var log = logger.New("pool")

// Worker represents a worker goroutine that processes jobs
type Worker struct {
	id       int
	pool     *Pool
	stopChan chan struct{}
	wg       *sync.WaitGroup
}

// newWorker creates a new worker
func newWorker(id int, pool *Pool, wg *sync.WaitGroup) *Worker {
	return &Worker{
		id:       id,
		pool:     pool,
		stopChan: make(chan struct{}),
		wg:       wg,
	}
}

// start starts the worker goroutine
func (w *Worker) start() {
	w.wg.Add(1)
	go w.run()
}

// stop signals the worker to stop
func (w *Worker) stop() {
	close(w.stopChan)
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // poll queue every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return

		case <-ticker.C:
			// Try to get a job from the queue
			item := w.pool.queue.Dequeue()
			if item == nil {
				continue // queue empty, wait for next tick
			}

			// Execute the job
			w.execute(item)
		}
	}
}

// execute processes a single queue item
func (w *Worker) execute(item *QueueItem) {
	// Pre-check if robot can run (non-atomic, just for early rejection).
	// Skip for jobs whose slot was pre-acquired by Tick — they already hold
	// a reserved slot and will pass TryAcquireSlot idempotently.
	if item.Robot.GetExecution(item.ExecID) == nil && !item.Robot.CanRun() {
		w.requeue(item, "quota pre-check failed")
		return
	}

	// Mark as running (only when actually executing)
	w.pool.incrementRunning()
	defer w.pool.decrementRunning()

	// Get executor based on mode (uses factory if available, otherwise default)
	exec := w.pool.GetExecutor(item.ExecutorMode)

	// Execute via Executor interface with pre-generated ID and control
	// Note: Executor.ExecuteWithControl() does atomic quota check via TryAcquireSlot()
	// The control parameter allows executor to check pause state during execution
	execution, err := exec.ExecuteWithControl(item.Ctx, item.Robot, item.Trigger, item.Data, item.ExecID, item.Control)

	if err != nil {
		// Check if it's a quota error (race condition - another worker got the slot)
		if err == types.ErrQuotaExceeded {
			w.requeue(item, "quota exceeded (race)")
			return
		}

		// Suspended execution: state is persisted, worker slot released gracefully.
		// Do NOT call onComplete — the execution stays in robot.Executions and execController
		// so that Resume can find it later (§16.1).
		if err == types.ErrExecutionSuspended {
			if execution != nil {
				log.Info("Worker %d: Execution %s suspended for robot %s (waiting for input)",
					w.id, execution.ID, item.Robot.MemberID)
			}
			return
		}

		log.Error("Worker %d: Execution failed for robot %s: %v",
			w.id, item.Robot.MemberID, err)
		// Notify completion callback with appropriate status
		if w.pool.onComplete != nil {
			status := types.ExecFailed
			if err == types.ErrExecutionCancelled {
				status = types.ExecCancelled
			}
			w.pool.onComplete(item.ExecID, item.Robot.MemberID, status)
		}
		return
	}

	if execution != nil {
		log.Info("Worker %d: Execution %s completed for robot %s (status: %s)",
			w.id, execution.ID, item.Robot.MemberID, execution.Status)
		// Notify completion callback
		if w.pool.onComplete != nil {
			w.pool.onComplete(execution.ID, item.Robot.MemberID, execution.Status)
		}
	}
}

// requeue attempts to put the item back in the queue
func (w *Worker) requeue(item *QueueItem, reason string) {
	// Queue length is our system load threshold:
	// - If queue has space: task waits for robot quota
	// - If queue is full: system is overloaded, drop task
	if !w.pool.queue.Enqueue(item) {
		log.Warn("Worker %d: Task for robot %s dropped (queue full, %s)",
			w.id, item.Robot.MemberID, reason)
	}
}
