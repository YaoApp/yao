package job

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

// WorkerManager manages job execution workers
type WorkerManager struct {
	maxWorkers    int
	activeWorkers map[string]*Worker
	workQueue     chan *WorkRequest
	workerPool    chan chan *WorkRequest
	quit          chan bool
	mu            sync.RWMutex
}

// Worker represents a single worker instance
type Worker struct {
	ID         string
	WorkerPool chan chan *WorkRequest
	JobChannel chan *WorkRequest
	Quit       chan bool
	Mode       ModeType
	ctx        context.Context
	cancel     context.CancelFunc
}

// WorkRequest represents a job execution request
type WorkRequest struct {
	Job       *Job
	Execution *Execution
	Handler   HandlerFunc
	Context   context.Context
}

// Global worker manager instance
var globalWorkerManager *WorkerManager
var workerManagerOnce sync.Once

// GetWorkerManager returns the global worker manager instance
func GetWorkerManager() *WorkerManager {
	workerManagerOnce.Do(func() {
		globalWorkerManager = NewWorkerManager(runtime.NumCPU() * 2) // Default to 2x CPU cores
	})
	return globalWorkerManager
}

// NewWorkerManagerForTest creates a new worker manager for testing (not singleton)
func NewWorkerManagerForTest(maxWorkers int) *WorkerManager {
	return NewWorkerManager(maxWorkers)
}

// NewWorkerManager creates a new worker manager
func NewWorkerManager(maxWorkers int) *WorkerManager {
	return &WorkerManager{
		maxWorkers:    maxWorkers,
		activeWorkers: make(map[string]*Worker),
		workQueue:     make(chan *WorkRequest, maxWorkers*2), // Buffer for queue
		workerPool:    make(chan chan *WorkRequest, maxWorkers),
		quit:          make(chan bool),
	}
}

// Start starts the worker manager
func (wm *WorkerManager) Start() {
	// Start workers
	for i := 0; i < wm.maxWorkers; i++ {
		worker := NewWorker(wm.workerPool, GOROUTINE)
		worker.Start()

		wm.mu.Lock()
		wm.activeWorkers[worker.ID] = worker
		wm.mu.Unlock()
	}

	// Start dispatcher
	go wm.dispatch()
	log.Info("Worker manager started with %d workers", wm.maxWorkers)
}

// Stop stops the worker manager
func (wm *WorkerManager) Stop() {
	log.Info("Stopping worker manager...")

	// Stop dispatcher first
	select {
	case <-wm.quit:
		// Already stopped
		return
	default:
		close(wm.quit)
	}

	// Stop all workers
	wm.mu.Lock()
	workers := make([]*Worker, 0, len(wm.activeWorkers))
	for _, worker := range wm.activeWorkers {
		workers = append(workers, worker)
	}
	wm.activeWorkers = make(map[string]*Worker)
	wm.mu.Unlock()

	// Stop workers and wait for them to finish
	for _, worker := range workers {
		worker.Stop()
	}

	// Give workers time to finish their current operations
	time.Sleep(200 * time.Millisecond)

	log.Info("Worker manager stopped")
}

// SubmitJob submits a job for execution
func (wm *WorkerManager) SubmitJob(job *Job, handler HandlerFunc) error {
	// Create execution record
	execution := &Execution{
		ExecutionID:     uuid.New().String(),
		JobID:           job.JobID,
		Status:          "queued",
		TriggerCategory: "manual", // Default trigger
		RetryAttempt:    0,
		Progress:        0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Save execution to database
	if err := SaveExecution(execution); err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	// Create work request
	workRequest := &WorkRequest{
		Job:       job,
		Execution: execution,
		Handler:   handler,
		Context:   context.Background(),
	}

	// Submit to work queue
	select {
	case wm.workQueue <- workRequest:
		log.Debug("Job %s submitted to work queue", job.JobID)
		return nil
	default:
		return fmt.Errorf("work queue is full")
	}
}

// dispatch dispatches work requests to available workers
func (wm *WorkerManager) dispatch() {
	for {
		select {
		case work := <-wm.workQueue:
			// Get an available worker
			select {
			case jobChannel := <-wm.workerPool:
				// Send work to worker
				jobChannel <- work
			case <-wm.quit:
				return
			}
		case <-wm.quit:
			return
		}
	}
}

// GetActiveWorkers returns the number of active workers
func (wm *WorkerManager) GetActiveWorkers() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.activeWorkers)
}

// NewWorker creates a new worker
func NewWorker(workerPool chan chan *WorkRequest, mode ModeType) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		ID:         uuid.New().String(),
		WorkerPool: workerPool,
		JobChannel: make(chan *WorkRequest),
		Quit:       make(chan bool),
		Mode:       mode,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the worker
func (w *Worker) Start() {
	go func() {
		for {
			// Register worker in the worker pool
			w.WorkerPool <- w.JobChannel

			select {
			case work := <-w.JobChannel:
				// Process the work
				w.processWork(work)
			case <-w.Quit:
				return
			}
		}
	}()
}

// Stop stops the worker
func (w *Worker) Stop() {
	w.cancel()
	select {
	case <-w.Quit:
		// Already stopped
		return
	default:
		close(w.Quit)
	}
}

// processWork processes a work request
func (w *Worker) processWork(work *WorkRequest) {
	log.Debug("Worker %s processing job %s", w.ID, work.Job.JobID)

	// Update execution status
	work.Execution.Status = "running"
	work.Execution.WorkerID = &w.ID
	work.Execution.StartedAt = &time.Time{}
	*work.Execution.StartedAt = time.Now()

	// Try to save execution, but don't fail if database is closed
	if err := SaveExecution(work.Execution); err != nil {
		log.Warn("Failed to save execution status (database may be closed): %v", err)
	}

	// Update job status
	work.Job.Status = "running"
	work.Job.CurrentExecutionID = &work.Execution.ExecutionID
	work.Job.LastRunAt = work.Execution.StartedAt

	// Try to save job, but don't fail if database is closed
	if err := SaveJob(work.Job); err != nil {
		log.Warn("Failed to save job status (database may be closed): %v", err)
	}

	// Create execution context with timeout
	ctx := work.Context
	if work.Job.DefaultTimeout != nil && *work.Job.DefaultTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(work.Context, time.Duration(*work.Job.DefaultTimeout)*time.Second)
		defer cancel()
	}

	// Set up progress tracking
	progress := &Progress{
		ExecutionID: work.Execution.ExecutionID,
		Progress:    0,
		Message:     "Starting execution",
	}

	// Update execution with progress manager
	work.Execution.Job = work.Job // Set job reference for progress updates

	var err error
	startTime := time.Now()

	// Execute based on mode
	switch w.Mode {
	case GOROUTINE:
		err = w.executeInGoroutine(ctx, work, progress)
	case PROCESS:
		err = w.executeInProcess(ctx, work, progress)
	default:
		err = fmt.Errorf("unsupported execution mode: %s", w.Mode)
	}

	// Calculate duration
	duration := int(time.Since(startTime).Milliseconds())
	endTime := time.Now()

	// Update execution with results
	work.Execution.EndedAt = &endTime
	work.Execution.Duration = &duration

	if err != nil {
		work.Execution.Status = "failed"
		errorInfo := map[string]interface{}{
			"error":  err.Error(),
			"time":   endTime,
			"worker": w.ID,
		}
		errorData, _ := jsoniter.Marshal(errorInfo)
		work.Execution.ErrorInfo = (*json.RawMessage)(&errorData)

		log.Error("Job %s execution failed: %v", work.Job.JobID, err)

		// Log error
		logEntry := &Log{
			JobID:       work.Job.JobID,
			Level:       "error",
			Message:     fmt.Sprintf("Execution failed: %v", err),
			ExecutionID: &work.Execution.ExecutionID,
			WorkerID:    &w.ID,
			Timestamp:   time.Now(),
			Sequence:    0,
		}
		if err := SaveLog(logEntry); err != nil {
			log.Warn("Failed to save error log (database may be closed): %v", err)
		}
	} else {
		work.Execution.Status = "completed"
		work.Execution.Progress = 100

		log.Info("Job %s execution completed successfully", work.Job.JobID)

		// Log completion
		logEntry := &Log{
			JobID:       work.Job.JobID,
			Level:       "info",
			Message:     "Execution completed successfully",
			ExecutionID: &work.Execution.ExecutionID,
			WorkerID:    &w.ID,
			Progress:    &work.Execution.Progress,
			Duration:    &duration,
			Timestamp:   time.Now(),
			Sequence:    1,
		}
		if err := SaveLog(logEntry); err != nil {
			log.Warn("Failed to save completion log (database may be closed): %v", err)
		}
	}

	// Update execution in database
	if err := SaveExecution(work.Execution); err != nil {
		log.Warn("Failed to save final execution status (database may be closed): %v", err)
	}

	// Update job status
	if work.Job.ScheduleType == string(ScheduleTypeOnce) {
		work.Job.Status = "completed"
	} else {
		work.Job.Status = "ready" // Ready for next execution
	}
	work.Job.CurrentExecutionID = nil
	if err := SaveJob(work.Job); err != nil {
		log.Warn("Failed to save final job status (database may be closed): %v", err)
	}

	log.Debug("Worker %s finished processing job %s", w.ID, work.Job.JobID)
}

// executeInGoroutine executes job in goroutine mode
func (w *Worker) executeInGoroutine(ctx context.Context, work *WorkRequest, progress *Progress) error {
	// Execute handler directly in current goroutine
	return work.Handler(ctx, work.Execution)
}

// executeInProcess executes job in process mode
func (w *Worker) executeInProcess(ctx context.Context, work *WorkRequest, progress *Progress) error {
	// For process mode, we would typically spawn a separate process
	// For now, we'll simulate this with a goroutine but with process isolation concepts

	// Set process ID (simulated)
	processID := fmt.Sprintf("proc_%s", uuid.New().String()[:8])
	work.Execution.ProcessID = &processID

	// Create a separate goroutine to simulate process isolation
	done := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("process panic: %v", r)
			}
		}()

		// Execute handler
		err := work.Handler(ctx, work.Execution)
		done <- err
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// executeInRealProcess executes job in a real separate process (future implementation)
func (w *Worker) executeInRealProcess(ctx context.Context, work *WorkRequest, progress *Progress) error {
	// This would be used for true process isolation
	// For now, it's a placeholder for future implementation

	// Create command to execute job in separate process
	cmd := exec.CommandContext(ctx, os.Args[0], "job-execute", work.Execution.ExecutionID)

	// Set environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("JOB_ID=%s", work.Job.JobID),
		fmt.Sprintf("EXECUTION_ID=%s", work.Execution.ExecutionID),
	)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("process execution failed: %v, output: %s", err, string(output))
	}

	return nil
}
