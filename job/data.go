package job

import (
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// ========================
// Jobs methods
// ========================

// ListJobs list jobs
func ListJobs(param model.QueryParam, page int, pagesize int) (maps.MapStrAny, error) {
	return nil, nil
}

// GetActiveJobs get active jobs
func GetActiveJobs() ([]*Job, error) {
	return nil, nil
}

// CountJobs count jobs
func CountJobs(param model.QueryParam) (int, error) {
	return 0, nil
}

// SaveJob save job
func SaveJob(job *Job) error {
	return nil
}

// RemoveJobs remove jobs
func RemoveJobs(ids []string) error {
	return nil
}

// GetJob get job
func GetJob(id string) (*Job, error) {
	return nil, nil
}

// ========================
// Categories methods
// ========================

// GetCategories get categories
func GetCategories(param model.QueryParam) ([]*Category, error) {
	return nil, nil
}

// CountCategories count categories
func CountCategories(param model.QueryParam) (int, error) {
	return 0, nil
}

// RemoveCategories remove categories
func RemoveCategories(ids []string) error {
	return nil
}

// SaveCategory save category
func SaveCategory(category *Category) error {
	return nil
}

// ========================
// Logs methods
// ========================

// ListLogs get logs
func ListLogs(id string, param model.QueryParam, page int, pagesize int) (maps.MapStrAny, error) {
	return nil, nil
}

// SaveLog save log
func SaveLog(log *Log) error {
	return nil
}

// RemoveLogs remove logs
func RemoveLogs(ids []string) error {
	return nil
}

// ========================
// Executions methods
// ========================

// GetExecutions get executions
func GetExecutions(id string) ([]*Execution, error) {
	return nil, nil
}

// CountExecutions count executions
func CountExecutions(id string, param model.QueryParam) (int, error) {
	return 0, nil
}

// RemoveExecutions remove executions
func RemoveExecutions(ids []string) error {
	return nil
}

// GetExecution get execution
func GetExecution(id string, param model.QueryParam) (*Execution, error) {
	return nil, nil
}

// ========================
// Live progress methods
// ========================

// GetProgress get progress with callback
func GetProgress(id string, cb func(progress *Progress)) (*Progress, error) {
	return nil, nil
}
