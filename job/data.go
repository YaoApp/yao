package job

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/dbal"
)

// ========================
// Jobs methods
// ========================

// ListJobs list jobs with pagination
func ListJobs(param model.QueryParam, page int, pagesize int) (maps.MapStrAny, error) {
	mod := model.Select("__yao.job")
	if mod == nil {
		return nil, fmt.Errorf("job model not found")
	}
	return mod.Paginate(param, page, pagesize)
}

// GetActiveJobs get active jobs (running, ready status)
func GetActiveJobs() ([]*Job, error) {
	mod := model.Select("__yao.job")
	if mod == nil {
		return nil, fmt.Errorf("job model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "status", OP: "in", Value: []string{"ready", "running"}},
			{Column: "enabled", Value: true},
		},
	}

	results, err := mod.Get(param)
	if err != nil {
		return nil, err
	}

	jobs := make([]*Job, 0, len(results))
	for _, result := range results {
		job := &Job{}
		if err := mapToStruct(result, job); err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// CountJobs count jobs
func CountJobs(param model.QueryParam) (int, error) {
	mod := model.Select("__yao.job")
	if mod == nil {
		return 0, fmt.Errorf("job model not found")
	}

	// Use dbal.Raw to count
	countParam := model.QueryParam{
		Select: []interface{}{dbal.Raw("COUNT(*) as count")},
		Wheres: param.Wheres,
	}

	result, err := mod.Get(countParam)
	if err != nil {
		return 0, fmt.Errorf("failed to count jobs: %w", err)
	}

	if len(result) == 0 {
		return 0, nil
	}

	// Extract count from result
	countValue, exists := result[0]["count"]
	if !exists {
		return 0, fmt.Errorf("count field not found in result")
	}

	// Convert to int
	switch v := countValue.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("unexpected count type: %T", v)
	}
}

// SaveJob save or update job
func SaveJob(job *Job) error {
	mod := model.Select("__yao.job")
	if mod == nil {
		return fmt.Errorf("job model not found")
	}

	data := structToMap(job)
	now := time.Now()

	if job.ID == 0 {
		// Create new job
		if job.JobID == "" {
			job.JobID = uuid.New().String()
		}

		// Remove ID field from data to let database auto-increment
		delete(data, "id")
		data["job_id"] = job.JobID
		data["created_at"] = now
		data["updated_at"] = now

		id, err := mod.Create(data)
		if err != nil {
			return fmt.Errorf("failed to create job: %w", err)
		}
		job.ID = uint(id)
	} else {
		// Update existing job
		data["updated_at"] = now
		delete(data, "id")         // Remove ID from update data
		delete(data, "job_id")     // Don't update job_id
		delete(data, "created_at") // Don't update created_at

		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "job_id", Value: job.JobID},
			},
			Limit: 1,
		}

		_, err := mod.UpdateWhere(param, data)
		if err != nil {
			return fmt.Errorf("failed to update job: %w", err)
		}
	}

	return nil
}

// RemoveJobs remove jobs by IDs
func RemoveJobs(ids []string) error {
	mod := model.Select("__yao.job")
	if mod == nil {
		return fmt.Errorf("job model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "job_id", OP: "in", Value: ids},
		},
	}

	_, err := mod.DeleteWhere(param)
	return err
}

// GetJob get job by job_id
func GetJob(jobID string) (*Job, error) {
	mod := model.Select("__yao.job")
	if mod == nil {
		return nil, fmt.Errorf("job model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "job_id", Value: jobID},
		},
		Limit: 1,
	}

	results, err := mod.Get(param)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	job := &Job{}
	if err := mapToStruct(results[0], job); err != nil {
		return nil, err
	}

	return job, nil
}

// ========================
// Categories methods
// ========================

// GetCategories get categories
func GetCategories(param model.QueryParam) ([]*Category, error) {
	mod := model.Select("__yao.job.category")
	if mod == nil {
		return nil, fmt.Errorf("job category model not found")
	}

	results, err := mod.Get(param)
	if err != nil {
		return nil, err
	}

	categories := make([]*Category, 0, len(results))
	for _, result := range results {
		category := &Category{}
		if err := mapToStruct(result, category); err != nil {
			continue
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// CountCategories count categories
func CountCategories(param model.QueryParam) (int, error) {
	mod := model.Select("__yao.job.category")
	if mod == nil {
		return 0, fmt.Errorf("job category model not found")
	}

	countParam := model.QueryParam{
		Select: []interface{}{dbal.Raw("COUNT(*) as count")},
		Wheres: param.Wheres,
	}

	result, err := mod.Get(countParam)
	if err != nil {
		return 0, fmt.Errorf("failed to count categories: %w", err)
	}

	if len(result) == 0 {
		return 0, nil
	}

	// Extract count from result
	countValue, exists := result[0]["count"]
	if !exists {
		return 0, fmt.Errorf("count field not found in result")
	}

	// Convert to int
	switch v := countValue.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("unexpected count type: %T", v)
	}
}

// RemoveCategories remove categories by category_id
func RemoveCategories(ids []string) error {
	mod := model.Select("__yao.job.category")
	if mod == nil {
		return fmt.Errorf("job category model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "category_id", OP: "in", Value: ids},
		},
	}

	_, err := mod.DeleteWhere(param)
	return err
}

// SaveCategory save or update category
func SaveCategory(category *Category) error {
	mod := model.Select("__yao.job.category")
	if mod == nil {
		return fmt.Errorf("job category model not found")
	}

	data := structToMap(category)
	now := time.Now()

	if category.ID == 0 {
		// Create new category
		if category.CategoryID == "" {
			category.CategoryID = uuid.New().String()
		}

		// Remove ID field from data to let database auto-increment
		delete(data, "id")
		data["category_id"] = category.CategoryID
		data["created_at"] = now
		data["updated_at"] = now

		id, err := mod.Create(data)
		if err != nil {
			return fmt.Errorf("failed to create category: %w", err)
		}
		category.ID = uint(id)
	} else {
		// Update existing category
		data["updated_at"] = now
		delete(data, "id")          // Remove ID from update data
		delete(data, "category_id") // Don't update category_id
		delete(data, "created_at")  // Don't update created_at

		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "category_id", Value: category.CategoryID},
			},
			Limit: 1,
		}

		_, err := mod.UpdateWhere(param, data)
		if err != nil {
			return fmt.Errorf("failed to update category: %w", err)
		}
	}

	return nil
}

// GetOrCreateCategory get or create category by name
func GetOrCreateCategory(name, description string) (*Category, error) {
	mod := model.Select("__yao.job.category")
	if mod == nil {
		return nil, fmt.Errorf("job category model not found")
	}

	// Try to find existing category
	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "name", Value: name},
		},
		Limit: 1,
	}

	results, err := mod.Get(param)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		// Category exists
		category := &Category{}
		if err := mapToStruct(results[0], category); err != nil {
			return nil, err
		}
		return category, nil
	}

	// Create new category
	category := &Category{
		CategoryID:  uuid.New().String(),
		Name:        name,
		Description: &description,
		Sort:        0,
		System:      false,
		Enabled:     true,
		Readonly:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := SaveCategory(category); err != nil {
		return nil, err
	}

	return category, nil
}

// ========================
// Logs methods
// ========================

// ListLogs get logs with pagination
func ListLogs(jobID string, param model.QueryParam, page int, pagesize int) (maps.MapStrAny, error) {
	mod := model.Select("__yao.job.log")
	if mod == nil {
		return nil, fmt.Errorf("job log model not found")
	}

	// Add job_id filter
	param.Wheres = append(param.Wheres, model.QueryWhere{
		Column: "job_id",
		Value:  jobID,
	})

	// Order by timestamp desc by default
	if len(param.Orders) == 0 {
		param.Orders = []model.QueryOrder{
			{Column: "timestamp", Option: "desc"},
		}
	}

	return mod.Paginate(param, page, pagesize)
}

// SaveLog save log (always creates new log entry)
func SaveLog(log *Log) error {
	mod := model.Select("__yao.job.log")
	if mod == nil {
		return fmt.Errorf("job log model not found")
	}

	data := structToMap(log)
	now := time.Now()

	// Always create new log entry
	// Remove ID field from data to let database auto-increment
	delete(data, "id")
	data["created_at"] = now
	data["updated_at"] = now
	if log.Timestamp.IsZero() {
		log.Timestamp = now
		data["timestamp"] = now
	}

	id, err := mod.Create(data)
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	log.ID = uint(id)

	return nil
}

// RemoveLogs remove logs by IDs
func RemoveLogs(ids []string) error {
	mod := model.Select("__yao.job.log")
	if mod == nil {
		return fmt.Errorf("job log model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "id", OP: "in", Value: ids},
		},
	}

	_, err := mod.DeleteWhere(param)
	return err
}

// ========================
// Executions methods
// ========================

// GetExecutions get executions by job_id
func GetExecutions(jobID string) ([]*Execution, error) {
	mod := model.Select("__yao.job.execution")
	if mod == nil {
		return nil, fmt.Errorf("job execution model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "job_id", Value: jobID},
		},
		Orders: []model.QueryOrder{
			{Column: "started_at", Option: "desc"},
		},
	}

	results, err := mod.Get(param)
	if err != nil {
		return nil, err
	}

	executions := make([]*Execution, 0, len(results))
	for _, result := range results {
		execution := &Execution{}
		if err := mapToStruct(result, execution); err != nil {
			continue
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// CountExecutions count executions
func CountExecutions(jobID string, param model.QueryParam) (int, error) {
	mod := model.Select("__yao.job.execution")
	if mod == nil {
		return 0, fmt.Errorf("job execution model not found")
	}

	// Add job_id filter
	param.Wheres = append(param.Wheres, model.QueryWhere{
		Column: "job_id",
		Value:  jobID,
	})

	countParam := model.QueryParam{
		Select: []interface{}{dbal.Raw("COUNT(*) as count")},
		Wheres: param.Wheres,
	}

	result, err := mod.Get(countParam)
	if err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}

	if len(result) == 0 {
		return 0, nil
	}

	// Extract count from result
	countValue, exists := result[0]["count"]
	if !exists {
		return 0, fmt.Errorf("count field not found in result")
	}

	// Convert to int
	switch v := countValue.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("unexpected count type: %T", v)
	}
}

// RemoveExecutions remove executions by execution_id
func RemoveExecutions(ids []string) error {
	mod := model.Select("__yao.job.execution")
	if mod == nil {
		return fmt.Errorf("job execution model not found")
	}

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "execution_id", OP: "in", Value: ids},
		},
	}

	_, err := mod.DeleteWhere(param)
	return err
}

// GetExecution get execution by execution_id
func GetExecution(executionID string, param model.QueryParam) (*Execution, error) {
	mod := model.Select("__yao.job.execution")
	if mod == nil {
		return nil, fmt.Errorf("job execution model not found")
	}

	param.Wheres = append(param.Wheres, model.QueryWhere{
		Column: "execution_id",
		Value:  executionID,
	})
	param.Limit = 1

	results, err := mod.Get(param)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	execution := &Execution{}
	if err := mapToStruct(results[0], execution); err != nil {
		return nil, err
	}

	return execution, nil
}

// SaveExecution save or update execution
func SaveExecution(execution *Execution) error {
	mod := model.Select("__yao.job.execution")
	if mod == nil {
		return fmt.Errorf("job execution model not found")
	}

	data := structToMap(execution)
	now := time.Now()

	if execution.ID == 0 {
		// Create new execution
		if execution.ExecutionID == "" {
			execution.ExecutionID = uuid.New().String()
		}

		// Remove ID field from data to let database auto-increment
		delete(data, "id")
		data["execution_id"] = execution.ExecutionID
		data["created_at"] = now
		data["updated_at"] = now

		id, err := mod.Create(data)
		if err != nil {
			return fmt.Errorf("failed to create execution: %w", err)
		}
		execution.ID = uint(id)
	} else {
		// Update existing execution
		data["updated_at"] = now
		delete(data, "id")
		delete(data, "execution_id")
		delete(data, "created_at")

		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "execution_id", Value: execution.ExecutionID},
			},
			Limit: 1,
		}

		_, err := mod.UpdateWhere(param, data)
		if err != nil {
			return fmt.Errorf("failed to update execution: %w", err)
		}
	}

	return nil
}

// ========================
// Live progress methods
// ========================

// GetProgress get progress with callback (for live updates)
func GetProgress(executionID string, cb func(progress *Progress)) (*Progress, error) {
	// This would typically involve websockets or SSE for live updates
	// For now, return current progress from execution
	execution, err := GetExecution(executionID, model.QueryParam{})
	if err != nil {
		return nil, err
	}

	progress := &Progress{
		ExecutionID: executionID,
		Progress:    execution.Progress,
		Message:     "", // Could be extracted from latest log
	}

	if cb != nil {
		cb(progress)
	}

	return progress, nil
}

// ========================
// Helper methods
// ========================

// structToMap converts struct to map for database operations
func structToMap(v interface{}) maps.MapStrAny {
	// This is a simplified implementation
	// In production, you might want to use reflection or a JSON marshal/unmarshal approach
	result := make(maps.MapStrAny)

	// Use JSON marshal/unmarshal for conversion
	data, _ := jsoniter.Marshal(v)
	_ = jsoniter.Unmarshal(data, &result)

	// Remove nil values and empty slices
	for key, value := range result {
		if value == nil {
			delete(result, key)
		}
	}

	return result
}

// mapToStruct converts map to struct
func mapToStruct(m maps.MapStr, v interface{}) error {
	// Clean up the map data to handle database type conversions
	cleanMap := make(map[string]interface{})
	for key, value := range m {
		switch key {
		case "enabled", "system", "readonly":
			// Convert numeric values to proper types for boolean fields
			switch val := value.(type) {
			case int:
				cleanMap[key] = val != 0
			case int64:
				cleanMap[key] = val != 0
			case float64:
				cleanMap[key] = val != 0
			case string:
				cleanMap[key] = val == "true" || val == "1"
			default:
				cleanMap[key] = value
			}
		case "created_at", "updated_at", "next_run_at", "last_run_at", "scheduled_at", "started_at", "finished_at", "timestamp":
			// Handle time fields - convert database time format to RFC3339
			if str, ok := value.(string); ok && str != "" {
				// Try to parse database time format "2006-01-02 15:04:05"
				if t, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
					cleanMap[key] = t.Format(time.RFC3339)
				} else {
					cleanMap[key] = value
				}
			} else {
				cleanMap[key] = value
			}
		default:
			cleanMap[key] = value
		}
	}

	// Use JSON marshal/unmarshal for conversion
	data, err := jsoniter.Marshal(cleanMap)
	if err != nil {
		return err
	}
	return jsoniter.Unmarshal(data, v)
}
