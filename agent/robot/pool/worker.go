package pool

import (
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

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
	// Pre-check if robot can run (non-atomic, just for early rejection)
	// The actual atomic check happens inside Executor.Execute() via TryAcquireSlot()
	if !item.Robot.CanRun() {
		// Robot likely at quota, re-enqueue for later
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
		fmt.Printf("Worker %d: Execution failed for robot %s: %v\n",
			w.id, item.Robot.MemberID, err)
		// Notify completion callback with appropriate status
		if w.pool.onComplete != nil {
			// Determine status based on error type
			status := types.ExecFailed
			if err == types.ErrExecutionCancelled {
				status = types.ExecCancelled
			}
			w.pool.onComplete(item.ExecID, item.Robot.MemberID, status)
		}
		return
	}

	if execution != nil {
		fmt.Printf("Worker %d: Execution %s completed for robot %s (status: %s)\n",
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
		// Queue full = system overloaded, drop task (protective discard)
		fmt.Printf("Worker %d: Task for robot %s dropped (queue full, %s)\n",
			w.id, item.Robot.MemberID, reason)
	}
}
