package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ExecutionRecord - persistent storage for robot execution history
// Maps to __yao.agent_execution model
type ExecutionRecord struct {
	ID          int64             `json:"id,omitempty"`     // Auto-increment primary key
	ExecutionID string            `json:"execution_id"`     // Unique execution identifier
	MemberID    string            `json:"member_id"`        // Robot member ID (globally unique)
	TeamID      string            `json:"team_id"`          // Team ID
	JobID       string            `json:"job_id,omitempty"` // Linked job.Job ID
	TriggerType types.TriggerType `json:"trigger_type"`     // clock | human | event

	// Status tracking (synced with runtime Execution)
	Status  types.ExecStatus `json:"status"` // pending | running | completed | failed | cancelled
	Phase   types.Phase      `json:"phase"`  // Current phase
	Current *CurrentState    `json:"current,omitempty"`
	Error   string           `json:"error,omitempty"`

	// Trigger input
	Input *types.TriggerInput `json:"input,omitempty"`

	// Phase outputs (P0-P5)
	Inspiration *types.InspirationReport `json:"inspiration,omitempty"`
	Goals       *types.Goals             `json:"goals,omitempty"`
	Tasks       []types.Task             `json:"tasks,omitempty"`
	Results     []types.TaskResult       `json:"results,omitempty"`
	Delivery    *types.DeliveryResult    `json:"delivery,omitempty"`
	Learning    []types.LearningEntry    `json:"learning,omitempty"`

	// Timestamps
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// CurrentState - current executing state (for JSON storage)
type CurrentState struct {
	TaskIndex int    `json:"task_index"`         // index in Tasks slice
	Progress  string `json:"progress,omitempty"` // human-readable progress (e.g., "2/5 tasks")
}

// ListOptions - options for listing execution records
type ListOptions struct {
	MemberID    string            `json:"member_id,omitempty"` // Filter by robot member ID
	TeamID      string            `json:"team_id,omitempty"`
	Status      types.ExecStatus  `json:"status,omitempty"`
	TriggerType types.TriggerType `json:"trigger_type,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
	OrderBy     string            `json:"order_by,omitempty"` // e.g., "start_time desc"
}

// ExecutionStore - persistent storage for robot execution records
type ExecutionStore struct {
	modelID string
}

// NewExecutionStore creates a new execution store instance
func NewExecutionStore() *ExecutionStore {
	return &ExecutionStore{
		modelID: "__yao.agent.execution",
	}
}

// Save creates or updates an execution record
func (s *ExecutionStore) Save(ctx context.Context, record *ExecutionRecord) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	data := s.recordToMap(record)

	// Check if record exists by execution_id
	existing, err := s.Get(ctx, record.ExecutionID)
	if err == nil && existing != nil {
		// Update existing record
		_, err = mod.UpdateWhere(
			model.QueryParam{
				Wheres: []model.QueryWhere{
					{Column: "execution_id", Value: record.ExecutionID},
				},
			},
			data,
		)
		if err != nil {
			return fmt.Errorf("failed to update execution record: %w", err)
		}
		return nil
	}

	// Create new record
	_, err = mod.Create(data)
	if err != nil {
		return fmt.Errorf("failed to create execution record: %w", err)
	}
	return nil
}

// Get retrieves an execution record by execution_id
func (s *ExecutionStore) Get(ctx context.Context, executionID string) (*ExecutionRecord, error) {
	mod := model.Select(s.modelID)
	if mod == nil {
		return nil, fmt.Errorf("model %s not found", s.modelID)
	}

	rows, err := mod.Get(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "execution_id", Value: executionID},
		},
		Limit: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get execution record: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	return s.mapToRecord(rows[0])
}

// List retrieves execution records with filters
func (s *ExecutionStore) List(ctx context.Context, opts *ListOptions) ([]*ExecutionRecord, error) {
	mod := model.Select(s.modelID)
	if mod == nil {
		return nil, fmt.Errorf("model %s not found", s.modelID)
	}

	params := model.QueryParam{}

	// Build where conditions
	var wheres []model.QueryWhere
	if opts != nil {
		if opts.MemberID != "" {
			wheres = append(wheres, model.QueryWhere{Column: "member_id", Value: opts.MemberID})
		}
		if opts.TeamID != "" {
			wheres = append(wheres, model.QueryWhere{Column: "team_id", Value: opts.TeamID})
		}
		if opts.Status != "" {
			wheres = append(wheres, model.QueryWhere{Column: "status", Value: string(opts.Status)})
		}
		if opts.TriggerType != "" {
			wheres = append(wheres, model.QueryWhere{Column: "trigger_type", Value: string(opts.TriggerType)})
		}

		params.Limit = opts.Limit
		if params.Limit == 0 {
			params.Limit = 100 // default limit
		}

		// Note: model.QueryParam doesn't have Offset, use Page instead
		if opts.Offset > 0 && opts.Limit > 0 {
			params.Page = (opts.Offset / opts.Limit) + 1
		}

		if opts.OrderBy != "" {
			// Parse OrderBy: "column desc" or "column asc" or just "column"
			parts := splitOrderBy(opts.OrderBy)
			params.Orders = []model.QueryOrder{{Column: parts[0], Option: parts[1]}}
		} else {
			params.Orders = []model.QueryOrder{{Column: "start_time", Option: "desc"}}
		}
	} else {
		params.Limit = 100
		params.Orders = []model.QueryOrder{{Column: "start_time", Option: "desc"}}
	}

	params.Wheres = wheres

	rows, err := mod.Get(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list execution records: %w", err)
	}

	records := make([]*ExecutionRecord, 0, len(rows))
	for _, row := range rows {
		record, err := s.mapToRecord(row)
		if err != nil {
			continue // skip invalid records
		}
		records = append(records, record)
	}

	return records, nil
}

// UpdatePhase updates the current phase and its data
func (s *ExecutionStore) UpdatePhase(ctx context.Context, executionID string, phase types.Phase, data interface{}) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	updateData := map[string]interface{}{
		"phase": string(phase),
	}

	// Set the appropriate phase output field
	switch phase {
	case types.PhaseInspiration:
		if data != nil {
			updateData["inspiration"] = data
		}
	case types.PhaseGoals:
		if data != nil {
			updateData["goals"] = data
		}
	case types.PhaseTasks:
		if data != nil {
			updateData["tasks"] = data
		}
	case types.PhaseRun:
		if data != nil {
			updateData["results"] = data
		}
	case types.PhaseDelivery:
		if data != nil {
			updateData["delivery"] = data
		}
	case types.PhaseLearning:
		if data != nil {
			updateData["learning"] = data
		}
	}

	_, err := mod.UpdateWhere(
		model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "execution_id", Value: executionID},
			},
		},
		updateData,
	)
	if err != nil {
		return fmt.Errorf("failed to update phase: %w", err)
	}

	return nil
}

// UpdateStatus updates the execution status
func (s *ExecutionStore) UpdateStatus(ctx context.Context, executionID string, status types.ExecStatus, errorMsg string) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	updateData := map[string]interface{}{
		"status": string(status),
	}

	if errorMsg != "" {
		updateData["error"] = errorMsg
	}

	// Set end_time for terminal states
	if status == types.ExecCompleted || status == types.ExecFailed || status == types.ExecCancelled {
		now := time.Now()
		updateData["end_time"] = now
	}

	_, err := mod.UpdateWhere(
		model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "execution_id", Value: executionID},
			},
		},
		updateData,
	)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// UpdateCurrent updates the current executing state
func (s *ExecutionStore) UpdateCurrent(ctx context.Context, executionID string, current *CurrentState) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	updateData := map[string]interface{}{
		"current": current,
	}

	_, err := mod.UpdateWhere(
		model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "execution_id", Value: executionID},
			},
		},
		updateData,
	)
	if err != nil {
		return fmt.Errorf("failed to update current state: %w", err)
	}

	return nil
}

// Delete removes an execution record by execution_id
func (s *ExecutionStore) Delete(ctx context.Context, executionID string) error {
	mod := model.Select(s.modelID)
	if mod == nil {
		return fmt.Errorf("model %s not found", s.modelID)
	}

	_, err := mod.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "execution_id", Value: executionID},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete execution record: %w", err)
	}

	return nil
}

// recordToMap converts ExecutionRecord to map for model operations
func (s *ExecutionStore) recordToMap(record *ExecutionRecord) map[string]interface{} {
	data := map[string]interface{}{
		"execution_id": record.ExecutionID,
		"member_id":    record.MemberID,
		"team_id":      record.TeamID,
		"trigger_type": string(record.TriggerType),
		"status":       string(record.Status),
		"phase":        string(record.Phase),
	}

	if record.JobID != "" {
		data["job_id"] = record.JobID
	}
	if record.Error != "" {
		data["error"] = record.Error
	}
	if record.Current != nil {
		data["current"] = record.Current
	}
	if record.Input != nil {
		data["input"] = record.Input
	}
	if record.Inspiration != nil {
		data["inspiration"] = record.Inspiration
	}
	if record.Goals != nil {
		data["goals"] = record.Goals
	}
	if record.Tasks != nil {
		data["tasks"] = record.Tasks
	}
	if record.Results != nil {
		data["results"] = record.Results
	}
	if record.Delivery != nil {
		data["delivery"] = record.Delivery
	}
	if record.Learning != nil {
		data["learning"] = record.Learning
	}
	if record.StartTime != nil {
		data["start_time"] = *record.StartTime
	}
	if record.EndTime != nil {
		data["end_time"] = *record.EndTime
	}

	return data
}

// mapToRecord converts a model row to ExecutionRecord
func (s *ExecutionStore) mapToRecord(row map[string]interface{}) (*ExecutionRecord, error) {
	record := &ExecutionRecord{}

	// Basic fields
	if v, ok := row["id"]; ok {
		switch id := v.(type) {
		case float64:
			record.ID = int64(id)
		case int64:
			record.ID = id
		case int:
			record.ID = int64(id)
		}
	}
	if v, ok := row["execution_id"].(string); ok {
		record.ExecutionID = v
	}
	if v, ok := row["member_id"].(string); ok {
		record.MemberID = v
	}
	if v, ok := row["team_id"].(string); ok {
		record.TeamID = v
	}
	if v, ok := row["job_id"].(string); ok {
		record.JobID = v
	}
	if v, ok := row["trigger_type"].(string); ok {
		record.TriggerType = types.TriggerType(v)
	}
	if v, ok := row["status"].(string); ok {
		record.Status = types.ExecStatus(v)
	}
	if v, ok := row["phase"].(string); ok {
		record.Phase = types.Phase(v)
	}
	if v, ok := row["error"].(string); ok {
		record.Error = v
	}

	// JSON fields - need to unmarshal
	if v := row["current"]; v != nil {
		record.Current = s.parseCurrentState(v)
	}
	if v := row["input"]; v != nil {
		record.Input = s.parseTriggerInput(v)
	}
	if v := row["inspiration"]; v != nil {
		record.Inspiration = s.parseInspirationReport(v)
	}
	if v := row["goals"]; v != nil {
		record.Goals = s.parseGoals(v)
	}
	if v := row["tasks"]; v != nil {
		record.Tasks = s.parseTasks(v)
	}
	if v := row["results"]; v != nil {
		record.Results = s.parseResults(v)
	}
	if v := row["delivery"]; v != nil {
		record.Delivery = s.parseDeliveryResult(v)
	}
	if v := row["learning"]; v != nil {
		record.Learning = s.parseLearningEntries(v)
	}

	// Timestamps
	if v := row["start_time"]; v != nil {
		record.StartTime = s.parseTime(v)
	}
	if v := row["end_time"]; v != nil {
		record.EndTime = s.parseTime(v)
	}
	if v := row["created_at"]; v != nil {
		record.CreatedAt = s.parseTime(v)
	}
	if v := row["updated_at"]; v != nil {
		record.UpdatedAt = s.parseTime(v)
	}

	return record, nil
}

// Helper functions for parsing JSON fields

func (s *ExecutionStore) parseCurrentState(v interface{}) *CurrentState {
	data, err := s.toJSON(v)
	if err != nil {
		return nil
	}
	var state CurrentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return &state
}

func (s *ExecutionStore) parseTriggerInput(v interface{}) *types.TriggerInput {
	data, err := s.toJSON(v)
	if err != nil {
		return nil
	}
	var input types.TriggerInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil
	}
	return &input
}

func (s *ExecutionStore) parseInspirationReport(v interface{}) *types.InspirationReport {
	data, err := s.toJSON(v)
	if err != nil {
		return nil
	}
	var report types.InspirationReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil
	}
	return &report
}

func (s *ExecutionStore) parseGoals(v interface{}) *types.Goals {
	data, err := s.toJSON(v)
	if err != nil {
		return nil
	}
	var goals types.Goals
	if err := json.Unmarshal(data, &goals); err != nil {
		return nil
	}
	return &goals
}

func (s *ExecutionStore) parseTasks(v interface{}) []types.Task {
	data, err := s.toJSON(v)
	if err != nil {
		return nil
	}
	var tasks []types.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil
	}
	return tasks
}

func (s *ExecutionStore) parseResults(v interface{}) []types.TaskResult {
	data, err := s.toJSON(v)
	if err != nil {
		return nil
	}
	var results []types.TaskResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil
	}
	return results
}

func (s *ExecutionStore) parseDeliveryResult(v interface{}) *types.DeliveryResult {
	data, err := s.toJSON(v)
	if err != nil {
		return nil
	}
	var result types.DeliveryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return &result
}

func (s *ExecutionStore) parseLearningEntries(v interface{}) []types.LearningEntry {
	data, err := s.toJSON(v)
	if err != nil {
		return nil
	}
	var entries []types.LearningEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}
	return entries
}

func (s *ExecutionStore) toJSON(v interface{}) ([]byte, error) {
	switch data := v.(type) {
	case []byte:
		return data, nil
	case string:
		return []byte(data), nil
	case map[string]interface{}, []interface{}:
		return json.Marshal(data)
	default:
		return json.Marshal(v)
	}
}

// splitOrderBy parses "column desc" or "column asc" or just "column"
// Returns [column, option] where option defaults to "desc"
func splitOrderBy(orderBy string) [2]string {
	parts := [2]string{"", "desc"}
	if orderBy == "" {
		return parts
	}

	// Split by space
	for i, c := range orderBy {
		if c == ' ' {
			parts[0] = orderBy[:i]
			rest := orderBy[i+1:]
			if rest == "asc" || rest == "ASC" {
				parts[1] = "asc"
			} else if rest == "desc" || rest == "DESC" {
				parts[1] = "desc"
			}
			return parts
		}
	}

	// No space found, just column name
	parts[0] = orderBy
	return parts
}

func (s *ExecutionStore) parseTime(v interface{}) *time.Time {
	switch t := v.(type) {
	case time.Time:
		return &t
	case *time.Time:
		return t
	case string:
		// Try parsing common time formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
		}
		for _, format := range formats {
			if parsed, err := time.Parse(format, t); err == nil {
				return &parsed
			}
		}
	}
	return nil
}

// FromExecution creates an ExecutionRecord from a runtime Execution
func FromExecution(exec *types.Execution) *ExecutionRecord {
	record := &ExecutionRecord{
		ExecutionID: exec.ID,
		MemberID:    exec.MemberID,
		TeamID:      exec.TeamID,
		JobID:       exec.JobID,
		TriggerType: exec.TriggerType,
		Status:      exec.Status,
		Phase:       exec.Phase,
		Error:       exec.Error,
		Input:       exec.Input,
		Inspiration: exec.Inspiration,
		Goals:       exec.Goals,
		Tasks:       exec.Tasks,
		Results:     exec.Results,
		Delivery:    exec.Delivery,
		Learning:    exec.Learning,
	}

	// Convert timestamps
	if !exec.StartTime.IsZero() {
		record.StartTime = &exec.StartTime
	}
	if exec.EndTime != nil {
		record.EndTime = exec.EndTime
	}

	// Convert CurrentState
	if exec.Current != nil {
		record.Current = &CurrentState{
			TaskIndex: exec.Current.TaskIndex,
			Progress:  exec.Current.Progress,
		}
	}

	return record
}

// ToExecution converts an ExecutionRecord to a runtime Execution
func (r *ExecutionRecord) ToExecution() *types.Execution {
	exec := &types.Execution{
		ID:          r.ExecutionID,
		MemberID:    r.MemberID,
		TeamID:      r.TeamID,
		JobID:       r.JobID,
		TriggerType: r.TriggerType,
		Status:      r.Status,
		Phase:       r.Phase,
		Error:       r.Error,
		Input:       r.Input,
		Inspiration: r.Inspiration,
		Goals:       r.Goals,
		Tasks:       r.Tasks,
		Results:     r.Results,
		Delivery:    r.Delivery,
		Learning:    r.Learning,
	}

	// Convert timestamps
	if r.StartTime != nil {
		exec.StartTime = *r.StartTime
	}
	if r.EndTime != nil {
		exec.EndTime = r.EndTime
	}

	// Convert CurrentState
	if r.Current != nil {
		exec.Current = &types.CurrentState{
			TaskIndex: r.Current.TaskIndex,
			Progress:  r.Current.Progress,
		}
	}

	return exec
}
