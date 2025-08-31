# Job Framework

A comprehensive task scheduling and execution framework supporting two execution modes: goroutine mode and process mode.

## Features

### 1. Database CRUD Operations

- **Jobs Management**: Create, read, update, delete jobs
- **Categories Management**: Automatic creation and management of job categories
- **Executions Management**: Complete lifecycle management of job execution instances
- **Logs Management**: Detailed execution logging and querying

### 2. Worker Management System

- **Goroutine Mode (GOROUTINE)**: Lightweight, fast execution
- **Process Mode (PROCESS)**: Independent process, isolated execution
- **Concurrency Control**: Support for multiple workers executing jobs concurrently
- **Resource Management**: Automatic management of worker pools and resource allocation

### 3. Progress Tracking

- **Real-time Progress Updates**: Support for progress updates during job execution
- **Database Persistence**: Progress information automatically saved to database
- **Callback Support**: Support for progress update callback functions

### 4. Logging System

- **Multi-level Logging**: Debug, Info, Warn, Error, Fatal, Panic, Trace
- **Structured Logging**: Includes execution context, timestamps, sequence numbers, etc.
- **Database Storage**: All logs automatically saved to database

## File Structure

```
job/
├── data.go           # Database CRUD operations implementation
├── data_test.go      # Database operations tests
├── execution.go      # Job execution logic
├── goroutine.go      # Goroutine mode interface
├── interfaces.go     # Interface definitions
├── job.go           # Main job management logic
├── job_test.go      # Original integration tests
├── process.go       # Process mode interface
├── progress.go      # Progress management
├── progress_test.go # Progress management tests
├── types.go         # Type definitions
├── types_test.go    # Type tests
├── worker.go        # Worker management system
├── worker_test.go   # Worker management tests
└── README.md        # This documentation
```

## Usage Examples

### Creating and Executing One-time Jobs

```go
// Create a goroutine mode one-time job
job, err := job.Once(job.GOROUTINE, map[string]interface{}{
    "name": "Example Job",
    "description": "This is an example job",
})

// Add handler function
handler := func(ctx context.Context, execution *job.Execution) error {
    execution.Info("Job started")
    execution.SetProgress(50, "In progress...")

    // Execute business logic
    time.Sleep(1 * time.Second)

    execution.SetProgress(100, "Completed")
    execution.Info("Job completed")
    return nil
}

err = job.Add(1, handler)
if err != nil {
    return err
}

// Start the job
err = job.Start()
```

### Creating Scheduled Jobs

```go
// Create a cron-based scheduled job
cronJob, err := job.Cron(job.PROCESS, map[string]interface{}{
    "name": "Cleanup Task",
}, "0 2 * * *") // Execute daily at 2 AM

err = cronJob.Add(1, cleanupHandler)
err = cronJob.Start()
```

### Creating Daemon Jobs

```go
// Create a continuously running daemon job
daemonJob, err := job.Daemon(job.GOROUTINE, map[string]interface{}{
    "name": "Monitor Daemon",
})

daemonHandler := func(ctx context.Context, execution *job.Execution) error {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // Execute periodic tasks
            execution.Info("Performing monitor check")
        }
    }
}

err = daemonJob.Add(1, daemonHandler)
err = daemonJob.Start()
```

## Data Models

### Job

- Supports three scheduling types: one-time, scheduled, and daemon
- Supports two execution modes: goroutine and process
- Contains complete job metadata and configuration

### Execution

- Each job execution creates an execution instance
- Records execution status, progress, timing, and other information
- Supports retry mechanisms and error handling

### Category

- Automatic creation and management of job categories
- Supports hierarchical category structures

### Log

- Detailed execution log records
- Supports multiple log levels
- Contains execution context information

## Test Coverage

### Database Tests (data_test.go)

- ✅ TestJobCRUD - Job CRUD operations
- ✅ TestCategoryCRUD - Category CRUD operations
- ✅ TestExecutionCRUD - Execution instance CRUD operations
- ✅ TestLogCRUD - Log CRUD operations

### Worker Tests (worker_test.go)

- ✅ TestWorkerManagerLifecycle - Worker manager lifecycle
- TestWorkerJobSubmission - Job submission tests
- TestWorkerModes - Execution mode tests
- TestWorkerErrorHandling - Error handling tests
- TestWorkerConcurrency - Concurrency tests

### Progress Tests (progress_test.go)

- ✅ TestProgressManager - Progress manager tests
- TestProgressWithExecution - Progress during execution tests
- ✅ TestProgressWithDatabase - Database progress persistence tests
- ✅ TestGetProgress - Progress retrieval tests

### Type Tests (types_test.go)

- ✅ TestJobTypes - Type constant tests
- ✅ TestJobStructure - Job struct tests
- ✅ TestCategoryStructure - Category struct tests
- ✅ TestExecutionStructure - Execution struct tests
- ✅ TestLogStructure - Log struct tests
- ✅ TestProgressStructure - Progress struct tests

## Running Tests

```bash
# Run all CRUD tests
go test -v ./job/... -run "CRUD"

# Run all type tests
go test -v ./job/... -run "Types|Structure"

# Run worker management tests
go test -v ./job/... -run "Worker"

# Run progress management tests
go test -v ./job/... -run "Progress"

# Run all tests
go test -v ./job/...
```

## Environment Requirements

Before running tests, make sure to load environment variables:

```bash
source $YAO_ROOT/env.local.sh
```

## Technical Features

1. **Complete CRUD Operations**: All data operations are thoroughly tested and verified
2. **Two Execution Modes**: Goroutine mode for lightweight tasks, process mode for better isolation
3. **Automatic Category Management**: Job categories are automatically created and managed
4. **Real-time Progress Tracking**: Support for real-time progress updates during job execution
5. **Comprehensive Logging System**: Multi-level, structured logging
6. **Concurrency Safe**: Support for multiple workers executing jobs concurrently
7. **Data Persistence**: All states and logs are persisted to database
8. **Complete Test Coverage**: Each functional module has corresponding test files

## API Reference

### Job Creation Functions

- `Once(mode ModeType, data map[string]interface{}) (*Job, error)` - Create one-time job
- `Cron(mode ModeType, data map[string]interface{}, expression string) (*Job, error)` - Create scheduled job
- `Daemon(mode ModeType, data map[string]interface{}) (*Job, error)` - Create daemon job

### Job Methods

- `Add(priority int, handler HandlerFunc) error` - Add handler to job
- `Start() error` - Start job execution
- `Cancel() error` - Cancel job
- `GetExecutions() ([]*Execution, error)` - Get job executions
- `SetCategory(category string) *Job` - Set job category

### Execution Methods

- `SetProgress(progress int, message string) error` - Update progress
- `Info(format string, args ...interface{}) error` - Log info message
- `Debug(format string, args ...interface{}) error` - Log debug message
- `Warn(format string, args ...interface{}) error` - Log warning message
- `Error(format string, args ...interface{}) error` - Log error message

### Database Functions

- `ListJobs(param model.QueryParam, page int, pagesize int) (maps.MapStrAny, error)` - List jobs with pagination
- `GetJob(id string) (*Job, error)` - Get job by ID
- `SaveJob(job *Job) error` - Save or update job
- `RemoveJobs(ids []string) error` - Remove jobs by IDs
- `GetOrCreateCategory(name, description string) (*Category, error)` - Get or create category

## Architecture

The framework follows a modular architecture with clear separation of concerns:

- **Data Layer** (`data.go`): Handles all database operations
- **Execution Layer** (`execution.go`, `job.go`): Manages job execution logic
- **Worker Layer** (`worker.go`): Manages worker pools and job distribution
- **Progress Layer** (`progress.go`): Handles progress tracking and updates
- **Type Layer** (`types.go`): Defines all data structures and constants

Each layer is thoroughly tested with comprehensive unit tests to ensure reliability and maintainability.
