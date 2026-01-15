package job

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/dbal"
)

// ========================
// Field Lists for SELECT queries
// ========================

// JobFields defines the fields to select for job queries
var JobFields = []interface{}{
	"id", "job_id", "name", "icon", "description", "category_id",
	"max_worker_nums", "status", "mode", "schedule_type", "schedule_expression",
	"max_retry_count", "default_timeout", "priority", "created_by",
	"next_run_at", "last_run_at", "current_execution_id", "config",
	"sort", "enabled", "system", "readonly", "created_at", "updated_at",
	"__yao_created_by", "__yao_updated_by", "__yao_team_id", "__yao_tenant_id",
}

// CategoryFields defines the fields to select for category queries
var CategoryFields = []interface{}{
	"id", "category_id", "name", "icon", "description",
	"sort", "system", "enabled", "readonly", "created_at", "updated_at",
}

// ExecutionFields defines the fields to select for execution queries
var ExecutionFields = []interface{}{
	"id", "execution_id", "job_id", "status", "trigger_category", "trigger_source",
	"trigger_context", "scheduled_at", "worker_id", "process_id", "retry_attempt",
	"parent_execution_id", "started_at", "ended_at", "timeout_seconds", "duration",
	"progress", "execution_config", "execution_options", "config_snapshot",
	"result", "error_info", "stack_trace", "metrics", "context", "created_at", "updated_at",
}

// LogFields defines the fields to select for log queries
var LogFields = []interface{}{
	"id", "job_id", "level", "message", "context", "source", "execution_id",
	"step", "progress", "duration", "error_code", "stack_trace",
	"worker_id", "process_id", "timestamp", "sequence", "created_at", "updated_at",
}

// ========================
// Jobs methods
// ========================

// ListJobs list jobs with pagination
func ListJobs(param model.QueryParam, page int, pagesize int) (maps.MapStrAny, error) {
	mod := model.Select("__yao.job")
	if mod == nil {
		return nil, fmt.Errorf("job model not found")
	}

	// Set select fields if not already specified
	if len(param.Select) == 0 {
		param.Select = JobFields
	}

	// Debug logging
	log.Debug("ListJobs called with param: %+v, page: %d, pagesize: %d", param, page, pagesize)

	result, err := mod.Paginate(param, page, pagesize)
	if err != nil {
		log.Error("ListJobs query error: %v", err)
		return nil, err
	}

	// Extract jobs data
	log.Debug("ListJobs raw result: %+v", result)

	jobsData, ok := result["data"].([]maps.MapStrAny)
	if !ok {
		log.Debug("Data type conversion failed, result[\"data\"] type: %T, value: %+v", result["data"], result["data"])
		// Try alternative type conversion
		if dataSlice, ok := result["data"].([]interface{}); ok {
			jobsData = make([]maps.MapStrAny, len(dataSlice))
			for i, item := range dataSlice {
				if mapItem, ok := item.(maps.MapStrAny); ok {
					jobsData[i] = mapItem
				} else if mapItem, ok := item.(map[string]interface{}); ok {
					jobsData[i] = maps.MapStrAny(mapItem)
				} else {
					log.Debug("Item %d type conversion failed: %T", i, item)
					return result, nil
				}
			}
		} else {
			log.Debug("Alternative conversion also failed")
			return result, nil
		}
	}

	if len(jobsData) == 0 {
		log.Debug("No jobs found in data")
		return result, nil
	}

	// Collect unique category IDs
	categoryIDs := make(map[string]bool)
	for _, job := range jobsData {
		if categoryID, exists := job["category_id"]; exists && categoryID != nil {
			if categoryIDStr, ok := categoryID.(string); ok && categoryIDStr != "" {
				categoryIDs[categoryIDStr] = true
			}
		}
	}

	// Query categories if we have category IDs
	categoryMap := make(map[string]string)
	if len(categoryIDs) > 0 {
		categoryIDList := make([]string, 0, len(categoryIDs))
		for categoryID := range categoryIDs {
			categoryIDList = append(categoryIDList, categoryID)
		}

		categoryMod := model.Select("__yao.job.category")
		if categoryMod != nil {
			categoryParam := model.QueryParam{
				Select: []interface{}{"category_id", "name"},
				Wheres: []model.QueryWhere{
					{Column: "category_id", OP: "in", Value: categoryIDList},
				},
			}

			categories, err := categoryMod.Get(categoryParam)
			if err != nil {
				log.Warn("Failed to fetch categories: %v", err)
			} else {
				for _, category := range categories {
					if categoryID, ok := category["category_id"].(string); ok {
						if categoryName, ok := category["name"].(string); ok {
							categoryMap[categoryID] = categoryName
						}
					}
				}
			}
		}
	}

	// Add category_name to jobs
	for i, job := range jobsData {
		if categoryID, exists := job["category_id"]; exists && categoryID != nil {
			if categoryIDStr, ok := categoryID.(string); ok {
				if categoryName, exists := categoryMap[categoryIDStr]; exists {
					jobsData[i]["category_name"] = categoryName
				} else {
					jobsData[i]["category_name"] = nil
				}
			}
		}
	}

	result["data"] = jobsData
	log.Debug("ListJobs result with categories: %+v", result)
	return result, nil
}

// GetActiveJobs get active jobs (running, ready status)
func GetActiveJobs() ([]*Job, error) {
	mod := model.Select("__yao.job")
	if mod == nil {
		return nil, fmt.Errorf("job model not found")
	}

	param := model.QueryParam{
		Select: JobFields,
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

	// If no CategoryID but CategoryName is provided, get or create category ID
	if job.CategoryID == "" && job.CategoryName != "" {
		categoryID, err := getCategoryIDByName(job.CategoryName)
		if err != nil {
			return fmt.Errorf("failed to get category ID by name '%s': %w", job.CategoryName, err)
		}
		job.CategoryID = categoryID
	}

	data := structToMap(job)
	now := time.Now()

	if job.ID == 0 {
		// Create new job
		if job.JobID == "" {
			var err error
			job.JobID, err = generateJobID()
			if err != nil {
				return fmt.Errorf("failed to generate job ID: %w", err)
			}
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
		Select: JobFields,
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

	jobData := results[0]

	// Query category name if category_id exists
	if categoryID, exists := jobData["category_id"]; exists && categoryID != nil {
		if categoryIDStr, ok := categoryID.(string); ok && categoryIDStr != "" {
			categoryMod := model.Select("__yao.job.category")
			if categoryMod != nil {
				categoryParam := model.QueryParam{
					Select: []interface{}{"name"},
					Wheres: []model.QueryWhere{
						{Column: "category_id", Value: categoryIDStr},
					},
					Limit: 1,
				}

				categoryResults, err := categoryMod.Get(categoryParam)
				if err == nil && len(categoryResults) > 0 {
					if categoryName, ok := categoryResults[0]["name"].(string); ok {
						jobData["category_name"] = categoryName
					}
				}
			}
		}
	}

	job := &Job{}
	if err := mapToStruct(jobData, job); err != nil {
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

	// Set select fields if not already specified
	if len(param.Select) == 0 {
		param.Select = CategoryFields
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
		// Create new category - but first check if name already exists
		if category.Name != "" {
			// Check if category with same name already exists
			param := model.QueryParam{
				Wheres: []model.QueryWhere{
					{Column: "name", Value: category.Name},
				},
				Limit: 1,
			}
			results, err := mod.Get(param)
			if err != nil {
				return fmt.Errorf("failed to check existing category: %w", err)
			}
			if len(results) > 0 {
				// Category with same name exists, update current category with existing data
				if err := mapToStruct(results[0], category); err != nil {
					return fmt.Errorf("failed to map existing category: %w", err)
				}
				return nil // Return the existing category
			}
		}

		// No existing category found, create new one
		if category.CategoryID == "" {
			var err error
			category.CategoryID, err = generateCategoryID()
			if err != nil {
				return fmt.Errorf("failed to generate category ID: %w", err)
			}
		}

		// Remove ID field from data to let database auto-increment
		delete(data, "id")
		data["category_id"] = category.CategoryID
		data["created_at"] = now
		data["updated_at"] = now

		id, err := mod.Create(data)
		if err != nil {
			// If creation failed due to duplicate name, try to find existing category
			if category.Name != "" {
				param := model.QueryParam{
					Wheres: []model.QueryWhere{
						{Column: "name", Value: category.Name},
					},
					Limit: 1,
				}
				results, findErr := mod.Get(param)
				if findErr == nil && len(results) > 0 {
					// Found existing category, use it
					if mapErr := mapToStruct(results[0], category); mapErr == nil {
						return nil // Successfully found and mapped existing category
					}
				}
			}
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
	categoryID, err := generateCategoryID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate category ID: %w", err)
	}

	category := &Category{
		CategoryID:  categoryID,
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

// getCategoryIDByName gets category ID by name, creates category if not exists
func getCategoryIDByName(categoryName string) (string, error) {
	category, err := ensureCategoryExists(categoryName)
	if err != nil {
		return "", err
	}
	return category.CategoryID, nil
}

// ensureCategoryExists ensures a category exists by name, creates if needed
func ensureCategoryExists(categoryName string) (*Category, error) {
	mod := model.Select("__yao.job.category")
	if mod == nil {
		return nil, fmt.Errorf("job category model not found")
	}

	// Try to find existing category by name (since external calls pass category name)
	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "name", Value: categoryName},
		},
		Limit: 1,
	}

	results, err := mod.Get(param)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		// Category exists, return it
		category := &Category{}
		if err := mapToStruct(results[0], category); err != nil {
			return nil, err
		}
		return category, nil
	}

	// Category doesn't exist, create it
	var categoryID, categoryDesc string

	if categoryName == "Default" {
		// Keep "default" as the category ID for the default category
		categoryID = "default"
		categoryDesc = "Default job category"
	} else {
		// For other categories, generate a new unique ID
		var err error
		categoryID, err = generateCategoryID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate category ID: %w", err)
		}
		categoryDesc = fmt.Sprintf("Auto-created category: %s", categoryName)
	}

	category := &Category{
		CategoryID:  categoryID,
		Name:        categoryName,
		Description: &categoryDesc,
		Sort:        0,
		System:      categoryName == "Default", // Mark default as system category
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

	// Set select fields if not already specified
	if len(param.Select) == 0 {
		param.Select = LogFields
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
		Select: ExecutionFields,
		Wheres: []model.QueryWhere{
			{Column: "job_id", Value: jobID},
		},
		Orders: []model.QueryOrder{
			{Column: "created_at", Option: "desc"},
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

		// Restore ExecutionConfig from ConfigSnapshot if available
		if execution.ConfigSnapshot != nil && len(*execution.ConfigSnapshot) > 0 {
			var config ExecutionConfig
			if err := jsoniter.Unmarshal(*execution.ConfigSnapshot, &config); err == nil {
				execution.ExecutionConfig = &config
			}
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

	// Set select fields if not already specified
	if len(param.Select) == 0 {
		param.Select = ExecutionFields
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

	// Restore ExecutionConfig from ConfigSnapshot if available
	if execution.ConfigSnapshot != nil && len(*execution.ConfigSnapshot) > 0 {
		var config ExecutionConfig
		if err := jsoniter.Unmarshal(*execution.ConfigSnapshot, &config); err == nil {
			execution.ExecutionConfig = &config
		}
	}

	return execution, nil
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m maps.MapStrAny) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// SaveExecution save or update execution
func SaveExecution(execution *Execution) error {
	mod := model.Select("__yao.job.execution")
	if mod == nil {
		return fmt.Errorf("job execution model not found")
	}

	data := structToMap(execution)

	// SQLite compatibility: ensure JSON fields are strings
	if data["config_snapshot"] != nil {
		if rawMsg, ok := data["config_snapshot"].(*jsoniter.RawMessage); ok && rawMsg != nil {
			data["config_snapshot"] = string(*rawMsg)
		}
	}
	if data["execution_options"] != nil {
		if rawMsg, ok := data["execution_options"].(*jsoniter.RawMessage); ok && rawMsg != nil {
			data["execution_options"] = string(*rawMsg)
		}
	}
	if data["result"] != nil {
		if rawMsg, ok := data["result"].(*jsoniter.RawMessage); ok && rawMsg != nil {
			data["result"] = string(*rawMsg)
		}
	}
	if data["error_info"] != nil {
		if rawMsg, ok := data["error_info"].(*jsoniter.RawMessage); ok && rawMsg != nil {
			data["error_info"] = string(*rawMsg)
		}
	}

	now := time.Now()

	if execution.ID == 0 {
		// Create new execution
		if execution.ExecutionID == "" {
			var err error
			execution.ExecutionID, err = generateExecutionID()
			if err != nil {
				return fmt.Errorf("failed to generate execution ID: %w", err)
			}
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

		affected, err := mod.UpdateWhere(param, data)
		if err != nil {
			return fmt.Errorf("failed to update execution: %w", err)
		}

		// Check if the update actually affected any rows
		if affected == 0 {
			log.Warn("Update execution %s affected 0 rows - execution may have been deleted", execution.ExecutionID)
		}
	}

	// Update related Job information after execution changes
	if err := updateJobProgress(execution.JobID); err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
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
// ID Generation methods
// ========================

// generateJobID generates a unique job_id using nanoid with duplicate checking
func generateJobID() (string, error) {
	const maxRetries = 10
	const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
	const length = 12

	mod := model.Select("__yao.job")
	if mod == nil {
		return "", fmt.Errorf("job model not found")
	}

	for i := 0; i < maxRetries; i++ {
		// Generate new ID using nanoid
		id, err := gonanoid.Generate(alphabet, length)
		if err != nil {
			return "", fmt.Errorf("failed to generate nanoid: %w", err)
		}

		// Check if ID already exists
		param := model.QueryParam{
			Select: []interface{}{"id"}, // Just get primary key, minimal data
			Wheres: []model.QueryWhere{
				{Column: "job_id", Value: id},
			},
			Limit: 1,
		}

		results, err := mod.Get(param)
		if err != nil {
			return "", fmt.Errorf("failed to check job_id existence: %w", err)
		}

		if len(results) == 0 {
			return id, nil // Found unique ID
		}

		// ID exists, retry with new generation
	}

	return "", fmt.Errorf("failed to generate unique job_id after %d retries", maxRetries)
}

// generateCategoryID generates a unique category_id using nanoid with duplicate checking
func generateCategoryID() (string, error) {
	const maxRetries = 10
	const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
	const length = 12

	mod := model.Select("__yao.job.category")
	if mod == nil {
		return "", fmt.Errorf("job category model not found")
	}

	for i := 0; i < maxRetries; i++ {
		// Generate new ID using nanoid
		id, err := gonanoid.Generate(alphabet, length)
		if err != nil {
			return "", fmt.Errorf("failed to generate nanoid: %w", err)
		}

		// Check if ID already exists
		param := model.QueryParam{
			Select: []interface{}{"id"}, // Just get primary key, minimal data
			Wheres: []model.QueryWhere{
				{Column: "category_id", Value: id},
			},
			Limit: 1,
		}

		results, err := mod.Get(param)
		if err != nil {
			return "", fmt.Errorf("failed to check category_id existence: %w", err)
		}

		if len(results) == 0 {
			return id, nil // Found unique ID
		}

		// ID exists, retry with new generation
	}

	return "", fmt.Errorf("failed to generate unique category_id after %d retries", maxRetries)
}

// generateExecutionID generates a unique execution_id using nanoid with duplicate checking
func generateExecutionID() (string, error) {
	const maxRetries = 10
	const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
	const length = 16 // Slightly longer for executions as they are more frequent

	mod := model.Select("__yao.job.execution")
	if mod == nil {
		return "", fmt.Errorf("job execution model not found")
	}

	for i := 0; i < maxRetries; i++ {
		// Generate new ID using nanoid
		id, err := gonanoid.Generate(alphabet, length)
		if err != nil {
			return "", fmt.Errorf("failed to generate nanoid: %w", err)
		}

		// Check if ID already exists
		param := model.QueryParam{
			Select: []interface{}{"id"}, // Just get primary key, minimal data
			Wheres: []model.QueryWhere{
				{Column: "execution_id", Value: id},
			},
			Limit: 1,
		}

		results, err := mod.Get(param)
		if err != nil {
			return "", fmt.Errorf("failed to check execution_id existence: %w", err)
		}

		if len(results) == 0 {
			return id, nil // Found unique ID
		}

		// ID exists, retry with new generation
	}

	return "", fmt.Errorf("failed to generate unique execution_id after %d retries", maxRetries)
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
		case "created_at", "updated_at", "next_run_at", "last_run_at", "scheduled_at", "started_at", "finished_at", "ended_at", "timestamp":
			// Handle time fields - support multiple time formats for SQLite/MySQL compatibility
			if str, ok := value.(string); ok && str != "" {
				// Try multiple time formats
				formats := []string{
					"2006-01-02 15:04:05",                 // MySQL format
					"2006-01-02T15:04:05Z07:00",           // RFC3339
					"2006-01-02T15:04:05.999999999Z07:00", // RFC3339 with nanoseconds
					time.RFC3339,
					time.RFC3339Nano,
				}

				for _, format := range formats {
					if t, err := time.Parse(format, str); err == nil {
						cleanMap[key] = t.Format(time.RFC3339)
						break
					}
				}

				// If no format worked, keep original value
				if cleanMap[key] == nil {
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

// updateJobProgress updates job progress and status based on its executions
func updateJobProgress(jobID string) error {
	// Skip if jobID is empty
	if jobID == "" {
		return nil
	}

	// Get all executions for this job
	executions, err := GetExecutions(jobID)
	if err != nil {
		return fmt.Errorf("failed to get executions for job %s: %w", jobID, err)
	}

	if len(executions) == 0 {
		return nil // No executions to process
	}

	// Calculate overall job progress and status
	totalExecutions := len(executions)
	completedCount := 0
	failedCount := 0
	runningCount := 0
	cancelledCount := 0
	totalProgress := 0

	for _, execution := range executions {
		totalProgress += execution.Progress

		switch execution.Status {
		case "completed":
			completedCount++
		case "failed":
			failedCount++
		case "running":
			runningCount++
		case "cancelled":
			cancelledCount++
		}
	}

	// Calculate average progress
	averageProgress := totalProgress / totalExecutions

	// Determine job status
	var jobStatus string
	if cancelledCount == totalExecutions {
		jobStatus = "cancelled" // All executions are cancelled
	} else if completedCount == totalExecutions {
		jobStatus = "completed"
	} else if failedCount > 0 && runningCount == 0 && completedCount+failedCount+cancelledCount == totalExecutions {
		jobStatus = "failed"
	} else if runningCount > 0 || completedCount > 0 {
		jobStatus = "running"
	} else if cancelledCount > 0 && cancelledCount+completedCount+failedCount == totalExecutions {
		// Mix of cancelled with completed/failed, no running
		jobStatus = "cancelled"
	} else {
		jobStatus = "ready" // All executions are queued
	}

	// Update job in database
	jobMod := model.Select("__yao.job")
	if jobMod == nil {
		return fmt.Errorf("job model not found")
	}

	updateData := map[string]interface{}{
		"status":     jobStatus,
		"updated_at": time.Now(),
	}

	// Add progress field if Job model supports it
	// Note: This assumes Job model has a progress field, you may need to add it to the schema
	updateData["progress"] = averageProgress

	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "job_id", Value: jobID},
		},
		Limit: 1,
	}

	_, err = jobMod.UpdateWhere(param, updateData)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}
