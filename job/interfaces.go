package job

// ProgressManager the progress manager
type ProgressManager interface {
	Set(progress int, message string) error
}

// JobManager the job manager
type JobManager interface{}
