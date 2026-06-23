package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/event"
	"github.com/yaoapp/yao/workspace"
)

// List returns a paginated list of tasks with derived fields from JOINs
func List(ctx context.Context, auth *process.AuthorizedInfo, q *ListQuery) (*ListResult, error) {
	if q.PageSize <= 0 {
		q.PageSize = 50
	}
	if q.Page <= 0 {
		q.Page = 1
	}

	// Build count query (separate builder to avoid state pollution)
	countQB := capsule.Global.Query()
	countQB.Table(tableTask()+" as t").
		LeftJoin(tableChat()+" as c", "t.chat_id", "=", "c.chat_id").
		LeftJoin(tableBoardColumn()+" as col", "t.column_id", "=", "col.column_id").
		WhereNull("t.deleted_at")

	if auth.Constraints.TeamOnly {
		countQB.Where("t.__yao_team_id", "=", auth.TeamID)
	}
	if auth.Constraints.CreatorOnly {
		countQB.Where("t.__yao_created_by", "=", auth.UserID)
	}
	if q.RunStatus != "" {
		countQB.Where("t.run_status", "=", q.RunStatus)
	}
	if q.AssistantID != "" {
		countQB.Where("c.assistant_id", "=", q.AssistantID)
	}
	if q.BoardID != "" {
		countQB.Where("col.board_id", "=", q.BoardID)
	}

	total, err := countQB.Count()
	if err != nil {
		return nil, fmt.Errorf("task.List count: %w", err)
	}

	// Build data query (fresh builder)
	qb := capsule.Global.Query()
	qb.Table(tableTask()+" as t").
		Select(
			"t.*",
			"c.title", "c.assistant_id", "c.last_workspace", "c.last_connector",
			"col.board_id",
			"a.name as assistant_name",
		).
		LeftJoin(tableChat()+" as c", "t.chat_id", "=", "c.chat_id").
		LeftJoin(tableBoardColumn()+" as col", "t.column_id", "=", "col.column_id").
		LeftJoin(tableAssistant()+" as a", "c.assistant_id", "=", "a.assistant_id").
		WhereNull("t.deleted_at")

	if auth.Constraints.TeamOnly {
		qb.Where("t.__yao_team_id", "=", auth.TeamID)
	}
	if auth.Constraints.CreatorOnly {
		qb.Where("t.__yao_created_by", "=", auth.UserID)
	}
	if q.RunStatus != "" {
		qb.Where("t.run_status", "=", q.RunStatus)
	}
	if q.AssistantID != "" {
		qb.Where("c.assistant_id", "=", q.AssistantID)
	}
	if q.BoardID != "" {
		qb.Where("col.board_id", "=", q.BoardID)
	}

	offset := (q.Page - 1) * q.PageSize
	rows, err := qb.OrderBy("t.position", "asc").
		Offset(offset).
		Limit(q.PageSize).
		Get()
	if err != nil {
		return nil, fmt.Errorf("task.List query: %w", err)
	}

	tasks := make([]*Task, 0, len(rows))
	for _, row := range rows {
		t := rowToTask(row)
		tasks = append(tasks, t)
	}

	resolveWorkspaceNames(ctx, tasks)

	if q.Locale != "" {
		for _, t := range tasks {
			if t.AssistantName != "" && t.AssistantID != "" {
				if translated := i18n.Translate(t.AssistantID, q.Locale, t.AssistantName); translated != nil {
					if s, ok := translated.(string); ok {
						t.AssistantName = s
					}
				}
			}
		}
	}

	return &ListResult{
		Tasks:    tasks,
		Total:    int64(total),
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

// Get retrieves a single task by chat_id
func Get(ctx context.Context, auth *process.AuthorizedInfo, chatID string) (*Task, error) {
	qb := capsule.Global.Query()

	row, err := qb.Table(tableTask()+" as t").
		Select(
			"t.*",
			"c.title", "c.assistant_id", "c.last_workspace", "c.last_connector",
			"col.board_id",
			"a.name as assistant_name",
		).
		LeftJoin(tableChat()+" as c", "t.chat_id", "=", "c.chat_id").
		LeftJoin(tableBoardColumn()+" as col", "t.column_id", "=", "col.column_id").
		LeftJoin(tableAssistant()+" as a", "c.assistant_id", "=", "a.assistant_id").
		WhereNull("t.deleted_at").
		Where("t.chat_id", "=", chatID).
		First()
	if err != nil {
		return nil, fmt.Errorf("task.Get: %w", err)
	}
	if row == nil {
		return nil, fmt.Errorf("task.Get: task %s not found", chatID)
	}

	// Permission check
	if auth.Constraints.TeamOnly {
		if getString(row, "__yao_team_id") != auth.TeamID {
			return nil, fmt.Errorf("task.Get: permission denied")
		}
	}
	if auth.Constraints.CreatorOnly {
		if getString(row, "__yao_created_by") != auth.UserID {
			return nil, fmt.Errorf("task.Get: permission denied")
		}
	}

	t := rowToTask(row)
	resolveWorkspaceNames(ctx, []*Task{t})
	return t, nil
}

// Create creates a new task with its associated chat
func Create(ctx context.Context, auth *process.AuthorizedInfo, req *CreateReq) (*Task, error) {
	chatID := req.ChatID
	if chatID == "" {
		chatID = uuid.New().String()
	}

	// Validate column_id exists
	colRow, err := capsule.Global.Query().Table(tableBoardColumn()).
		Where("column_id", "=", req.ColumnID).
		WhereNull("deleted_at").
		First()
	if err != nil || colRow == nil {
		return nil, fmt.Errorf("task.Create: column %s not found", req.ColumnID)
	}

	now := time.Now()

	// Get max position in target column
	maxPos := 0
	posResult, _ := capsule.Global.Query().Table(tableTask()).
		Where("column_id", "=", req.ColumnID).
		WhereNull("deleted_at").
		Max("position")
	if posResult.Number != nil {
		switch v := posResult.Number.(type) {
		case float64:
			maxPos = int(v)
		case int64:
			maxPos = int(v)
		case int:
			maxPos = v
		}
	}

	// INSERT agent_chat
	err = capsule.Global.Query().Table(tableChat()).Insert(map[string]interface{}{
		"chat_id":          chatID,
		"title":            req.Title,
		"assistant_id":     req.AssistantID,
		"status":           "active",
		"__yao_created_by": auth.UserID,
		"__yao_team_id":    auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	if err != nil {
		return nil, fmt.Errorf("task.Create insert chat: %w", err)
	}

	// INSERT agent_task
	err = capsule.Global.Query().Table(tableTask()).Insert(map[string]interface{}{
		"chat_id":          chatID,
		"column_id":        req.ColumnID,
		"position":         maxPos + 1,
		"run_status":       "pending",
		"priority":         "none",
		"pinned":           false,
		"progress":         0,
		"duration":         0,
		"run_count":        0,
		"__yao_created_by": auth.UserID,
		"__yao_team_id":    auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	})
	if err != nil {
		return nil, fmt.Errorf("task.Create insert task: %w", err)
	}

	// Push event
	event.Push(ctx, "task.created", map[string]any{
		"chat_id":       chatID,
		"column_id":     req.ColumnID,
		"title":         req.Title,
		"__yao_team_id": auth.TeamID,
	})

	return Get(ctx, auth, chatID)
}

// Update partially updates a task (only non-nil fields).
// Note: run_status cannot be modified via Update — it is controlled by the execution engine.
func Update(ctx context.Context, auth *process.AuthorizedInfo, chatID string, req *UpdateReq) (*Task, error) {
	// Verify existence and permission
	_, err := Get(ctx, auth, chatID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	taskUpdates := map[string]interface{}{"updated_at": now}
	chatUpdates := map[string]interface{}{"updated_at": now}

	// Task-level fields
	if req.ColumnID != nil {
		taskUpdates["column_id"] = *req.ColumnID
	}
	if req.Pinned != nil {
		taskUpdates["pinned"] = *req.Pinned
	}
	if req.Priority != nil {
		taskUpdates["priority"] = *req.Priority
	}
	if req.Tags != nil {
		tagsJSON, _ := jsoniter.MarshalToString(req.Tags)
		taskUpdates["tags"] = tagsJSON
	}
	if req.ComputerID != nil {
		taskUpdates["computer_id"] = *req.ComputerID
	}
	if req.ComputerMode != nil {
		taskUpdates["computer_mode"] = *req.ComputerMode
	}
	if req.SandboxType != nil {
		taskUpdates["sandbox_type"] = *req.SandboxType
	}
	if req.Instruction != nil {
		taskUpdates["instruction"] = *req.Instruction
	}
	if req.Summary != nil {
		taskUpdates["summary"] = *req.Summary
	}
	if req.Outputs != nil {
		outputsJSON, _ := jsoniter.MarshalToString(req.Outputs)
		taskUpdates["outputs"] = outputsJSON
	}
	if req.Metadata != nil {
		metaJSON, _ := jsoniter.MarshalToString(req.Metadata)
		taskUpdates["metadata"] = metaJSON
	}

	// Cross-table sync: fields that belong to agent_chat
	if req.Title != nil {
		chatUpdates["title"] = *req.Title
	}
	if req.AssistantID != nil {
		chatUpdates["assistant_id"] = *req.AssistantID
	}
	if req.LastWorkspace != nil {
		chatUpdates["last_workspace"] = *req.LastWorkspace
	}

	// Update agent_task
	if len(taskUpdates) > 1 {
		_, err = capsule.Global.Query().Table(tableTask()).
			Where("chat_id", "=", chatID).
			Update(taskUpdates)
		if err != nil {
			return nil, fmt.Errorf("task.Update agent_task: %w", err)
		}
	}

	// Update agent_chat
	if len(chatUpdates) > 1 {
		_, err = capsule.Global.Query().Table(tableChat()).
			Where("chat_id", "=", chatID).
			Update(chatUpdates)
		if err != nil {
			return nil, fmt.Errorf("task.Update agent_chat: %w", err)
		}
	}

	return Get(ctx, auth, chatID)
}

// Delete soft-deletes a task and archives its chat
func Delete(ctx context.Context, auth *process.AuthorizedInfo, chatID string) error {
	// Verify existence and permission
	_, err := Get(ctx, auth, chatID)
	if err != nil {
		return err
	}

	now := time.Now()

	// Soft delete agent_task
	_, err = capsule.Global.Query().Table(tableTask()).
		Where("chat_id", "=", chatID).
		Update(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		})
	if err != nil {
		return fmt.Errorf("task.Delete agent_task: %w", err)
	}

	// Archive agent_chat
	_, err = capsule.Global.Query().Table(tableChat()).
		Where("chat_id", "=", chatID).
		Update(map[string]interface{}{
			"status":     "archived",
			"updated_at": now,
		})
	if err != nil {
		return fmt.Errorf("task.Delete agent_chat: %w", err)
	}

	// Push event
	event.Push(ctx, "task.deleted", map[string]any{
		"chat_id":       chatID,
		"__yao_team_id": auth.TeamID,
	})

	return nil
}

// CreateFromWS creates a task from WebSocket first message (atomic: chat + task with running status).
// Task parameters (column_id, assistant_id, etc.) are extracted from req.Metadata,
// consistent with Stream/Interact interface design.
func CreateFromWS(ctx context.Context, auth *process.AuthorizedInfo, req *CreateFromWSReq) (*Task, error) {
	chatID := req.ChatID
	if chatID == "" {
		chatID = uuid.New().String()
	}

	// Extract parameters from metadata
	columnID := metaString(req.Metadata, "column_id")
	assistantID := metaString(req.Metadata, "assistant_id")
	computerID := metaString(req.Metadata, "computer_id")
	computerMode := metaString(req.Metadata, "computer_mode")
	workspaceID := metaString(req.Metadata, "workspace_id")

	// Validate column_id if provided
	if columnID != "" {
		colRow, err := capsule.Global.Query().Table(tableBoardColumn()).
			Where("column_id", "=", columnID).
			WhereNull("deleted_at").
			First()
		if err != nil || colRow == nil {
			return nil, fmt.Errorf("task.CreateFromWS: column %s not found", columnID)
		}
	}

	now := time.Now()

	// Get max position in target column (0 if no column)
	maxPos := 0
	if columnID != "" {
		posResult, _ := capsule.Global.Query().Table(tableTask()).
			Where("column_id", "=", columnID).
			WhereNull("deleted_at").
			Max("position")
		if posResult.Number != nil {
			switch v := posResult.Number.(type) {
			case float64:
				maxPos = int(v)
			case int64:
				maxPos = int(v)
			case int:
				maxPos = v
			}
		}
	}

	// INSERT agent_chat
	chatData := map[string]interface{}{
		"chat_id":          chatID,
		"title":            req.Title,
		"status":           "active",
		"__yao_created_by": auth.UserID,
		"__yao_team_id":    auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	}
	if assistantID != "" {
		chatData["assistant_id"] = assistantID
	}
	if workspaceID != "" {
		chatData["last_workspace"] = workspaceID
	}
	err := capsule.Global.Query().Table(tableChat()).Insert(chatData)
	if err != nil {
		return nil, fmt.Errorf("task.CreateFromWS insert chat: %w", err)
	}

	// INSERT agent_task with run_status=running
	taskData := map[string]interface{}{
		"chat_id":          chatID,
		"position":         maxPos + 1,
		"run_status":       "running",
		"priority":         "none",
		"pinned":           false,
		"progress":         0,
		"duration":         0,
		"run_count":        1,
		"started_at":       now,
		"__yao_created_by": auth.UserID,
		"__yao_team_id":    auth.TeamID,
		"created_at":       now,
		"updated_at":       now,
	}
	if columnID != "" {
		taskData["column_id"] = columnID
	}
	if computerID != "" {
		taskData["computer_id"] = computerID
	}
	if computerMode != "" {
		taskData["computer_mode"] = computerMode
	}
	err = capsule.Global.Query().Table(tableTask()).Insert(taskData)
	if err != nil {
		return nil, fmt.Errorf("task.CreateFromWS insert task: %w", err)
	}

	logTaskCreated(chatID, columnID, assistantID)

	// Push event
	event.Push(ctx, "task.created", map[string]any{
		"chat_id":       chatID,
		"column_id":     columnID,
		"title":         req.Title,
		"run_status":    "running",
		"__yao_team_id": auth.TeamID,
	})

	return Get(ctx, auth, chatID)
}

// metaString safely extracts a string value from metadata map
func metaString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// rowToTask converts a database row map to a Task struct
func rowToTask(row map[string]interface{}) *Task {
	t := &Task{
		ChatID:    getString(row, "chat_id"),
		Position:  getInt(row, "position"),
		Pinned:    getBool(row, "pinned"),
		Priority:  getStringDefault(row, "priority", "none"),
		RunStatus: getStringDefault(row, "run_status", "pending"),
		Progress:  getInt(row, "progress"),
		Duration:  getInt(row, "duration"),
		RunCount:  getInt(row, "run_count"),

		// Derived
		Title:         getString(row, "title"),
		AssistantID:   getString(row, "assistant_id"),
		AssistantName: getString(row, "assistant_name"),
	}

	if v := getStringPtr(row, "column_id"); v != nil {
		t.ColumnID = v
	}
	if v := getStringPtr(row, "current_step"); v != nil {
		t.CurrentStep = v
	}
	if v := getStringPtr(row, "error_message"); v != nil {
		t.ErrorMessage = v
	}
	if v := getStringPtr(row, "computer_id"); v != nil {
		t.ComputerID = v
	}
	if v := getStringPtr(row, "computer_mode"); v != nil {
		t.ComputerMode = v
	}
	if v := getStringPtr(row, "sandbox_type"); v != nil {
		t.SandboxType = v
	}
	t.Instruction = getString(row, "instruction")
	t.Summary = getString(row, "summary")
	if outputsRaw, ok := row["outputs"]; ok && outputsRaw != nil {
		switch v := outputsRaw.(type) {
		case string:
			var outputs any
			jsoniter.UnmarshalFromString(v, &outputs)
			t.Outputs = outputs
		default:
			t.Outputs = v
		}
	}
	if v := getStringPtr(row, "last_workspace"); v != nil {
		t.LastWorkspace = v
	}
	if v := getStringPtr(row, "last_connector"); v != nil {
		t.LastConnector = v
	}
	if v := getStringPtr(row, "board_id"); v != nil {
		t.BoardID = v
	}
	if v := getTime(row, "started_at"); v != nil {
		t.StartedAt = v
	}
	if v := getTime(row, "completed_at"); v != nil {
		t.CompletedAt = v
	}
	if v := getTime(row, "created_at"); v != nil {
		t.CreatedAt = v
	}
	if v := getTime(row, "updated_at"); v != nil {
		t.UpdatedAt = v
	}

	// Parse tags JSON
	if tagsRaw, ok := row["tags"]; ok && tagsRaw != nil {
		switch v := tagsRaw.(type) {
		case string:
			var tags []string
			jsoniter.UnmarshalFromString(v, &tags)
			t.Tags = tags
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					t.Tags = append(t.Tags, s)
				}
			}
		}
	}

	return t
}

// Helper functions for safe type conversion from row maps

func getString(row map[string]interface{}, key string) string {
	if v, ok := row[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStringDefault(row map[string]interface{}, key, def string) string {
	s := getString(row, key)
	if s == "" {
		return def
	}
	return s
}

func getStringPtr(row map[string]interface{}, key string) *string {
	if v, ok := row[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return &s
		}
	}
	return nil
}

func getInt(row map[string]interface{}, key string) int {
	if v, ok := row[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func getBool(row map[string]interface{}, key string) bool {
	if v, ok := row[key]; ok && v != nil {
		switch b := v.(type) {
		case bool:
			return b
		case float64:
			return b != 0
		case int64:
			return b != 0
		}
	}
	return false
}

func getTime(row map[string]interface{}, key string) *time.Time {
	if v, ok := row[key]; ok && v != nil {
		switch t := v.(type) {
		case time.Time:
			return &t
		case string:
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				return &parsed
			}
			if parsed, err := time.Parse("2006-01-02 15:04:05", t); err == nil {
				return &parsed
			}
		}
	}
	return nil
}

// resolveWorkspaceNames batch-resolves workspace names for tasks that have a
// LastWorkspace value. Errors are silently ignored (name stays empty).
func resolveWorkspaceNames(ctx context.Context, tasks []*Task) {
	seen := make(map[string]string)
	for _, t := range tasks {
		if t.LastWorkspace == nil || *t.LastWorkspace == "" {
			continue
		}
		wsID := *t.LastWorkspace
		if name, ok := seen[wsID]; ok {
			t.WorkspaceName = name
			continue
		}
		ws, err := workspace.M().Get(ctx, wsID)
		if err == nil && ws != nil {
			seen[wsID] = ws.Name
			t.WorkspaceName = ws.Name
		} else {
			seen[wsID] = ""
		}
	}
}
