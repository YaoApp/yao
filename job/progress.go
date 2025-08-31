package job

import (
	"sync"

	"github.com/yaoapp/gou/model"
)

// Progress the progress manager struct
type Progress struct {
	ExecutionID string `json:"execution_id"`
	Progress    int    `json:"progress"`
	Message     string `json:"message"`
	mu          sync.RWMutex
}

// Progress Progress manager
func (j *Job) Progress() ProgressManager {
	return &Progress{
		ExecutionID: "", // Will be set when execution starts
		Progress:    0,
		Message:     "",
	}
}

// Set set the progress
func (p *Progress) Set(progress int, message string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Progress = progress
	p.Message = message

	// Update execution in database if execution ID is set
	if p.ExecutionID != "" {
		execution, err := GetExecution(p.ExecutionID, model.QueryParam{})
		if err == nil {
			execution.Progress = progress
			SaveExecution(execution)
		}
	}

	return nil
}

// Get get current progress
func (p *Progress) Get() (int, string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Progress, p.Message
}
