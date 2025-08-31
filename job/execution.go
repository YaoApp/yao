package job

// Add add a new execution to the job
func (j *Job) Add(priority int, handler HandlerFunc) error {
	return nil
}

// GetExecutions get executions
func (j *Job) GetExecutions() ([]*Execution, error) {
	return nil, nil
}

// GetExecution get execution
func (j *Job) GetExecution(id string) (*Execution, error) {
	return nil, nil
}

// Log log
func (e *Execution) Log(level LogLevel, format string, args ...interface{}) error {
	return nil
}

// Info info log
func (e *Execution) Info(format string, args ...interface{}) error {
	return e.Log(Info, format, args...)
}

// Debug debug log
func (e *Execution) Debug(format string, args ...interface{}) error {
	return e.Log(Debug, format, args...)
}

// Warn warn log
func (e *Execution) Warn(format string, args ...interface{}) error {
	return e.Log(Warn, format, args...)
}

// Error error log
func (e *Execution) Error(format string, args ...interface{}) error {
	return e.Log(Error, format, args...)
}

// Fatal fatal log
func (e *Execution) Fatal(format string, args ...interface{}) error {
	return e.Log(Fatal, format, args...)
}

// Panic panic log
func (e *Execution) Panic(format string, args ...interface{}) error {
	return e.Log(Panic, format, args...)
}

// Trace trace log
func (e *Execution) Trace(format string, args ...interface{}) error {
	return e.Log(Trace, format, args...)
}

// SetProgress set the progress
func (e *Execution) SetProgress(progress int, message string) error {
	p := e.Job.Progress()
	p.Set(progress, message)
	return nil
}
