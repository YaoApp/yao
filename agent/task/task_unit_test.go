//go:build unit

package task_test

import (
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/task"
)

// --- metaString ---

func TestMetaStringNilMap(t *testing.T) {
	if task.ExportMetaString(nil, "key") != "" {
		t.Error("nil map should return empty")
	}
}

func TestMetaStringMissingKey(t *testing.T) {
	m := map[string]any{"a": "b"}
	if task.ExportMetaString(m, "x") != "" {
		t.Error("missing key should return empty")
	}
}

func TestMetaStringNonString(t *testing.T) {
	m := map[string]any{"num": 123}
	if task.ExportMetaString(m, "num") != "" {
		t.Error("non-string value should return empty")
	}
}

func TestMetaStringValid(t *testing.T) {
	m := map[string]any{"id": "col-001"}
	if task.ExportMetaString(m, "id") != "col-001" {
		t.Error("valid string extraction failed")
	}
}

// --- getString ---

func TestGetString(t *testing.T) {
	row := map[string]interface{}{"name": "hello", "num": 42, "nil_v": nil}
	if task.ExportGetString(row, "name") != "hello" {
		t.Error("string value")
	}
	if task.ExportGetString(row, "num") != "" {
		t.Error("non-string should return empty")
	}
	if task.ExportGetString(row, "nil_v") != "" {
		t.Error("nil should return empty")
	}
	if task.ExportGetString(row, "missing") != "" {
		t.Error("missing key should return empty")
	}
}

// --- getStringDefault ---

func TestGetStringDefault(t *testing.T) {
	row := map[string]interface{}{"status": "running", "empty": ""}
	if task.ExportGetStringDefault(row, "status", "pending") != "running" {
		t.Error("existing value should override default")
	}
	if task.ExportGetStringDefault(row, "missing", "pending") != "pending" {
		t.Error("missing key should return default")
	}
	if task.ExportGetStringDefault(row, "empty", "pending") != "pending" {
		t.Error("empty string should return default")
	}
}

// --- getStringPtr ---

func TestGetStringPtr(t *testing.T) {
	row := map[string]interface{}{"val": "hello", "nil_v": nil}
	ptr := task.ExportGetStringPtr(row, "val")
	if ptr == nil || *ptr != "hello" {
		t.Errorf("got %v, want *hello", ptr)
	}
	if task.ExportGetStringPtr(row, "nil_v") != nil {
		t.Error("nil should return nil ptr")
	}
	if task.ExportGetStringPtr(row, "missing") != nil {
		t.Error("missing should return nil ptr")
	}
}

// --- getInt ---

func TestGetInt(t *testing.T) {
	row := map[string]interface{}{
		"f64": float64(42), "i64": int64(100), "i": 7, "str": "x", "nil_v": nil,
	}
	if task.ExportGetInt(row, "f64") != 42 {
		t.Error("float64")
	}
	if task.ExportGetInt(row, "i64") != 100 {
		t.Error("int64")
	}
	if task.ExportGetInt(row, "i") != 7 {
		t.Error("int")
	}
	if task.ExportGetInt(row, "str") != 0 {
		t.Error("string should be 0")
	}
	if task.ExportGetInt(row, "missing") != 0 {
		t.Error("missing should be 0")
	}
}

// --- getBool ---

func TestGetBool(t *testing.T) {
	row := map[string]interface{}{
		"t": true, "f": false, "f1": float64(1), "f0": float64(0), "i1": int64(1), "i0": int64(0),
	}
	if !task.ExportGetBool(row, "t") {
		t.Error("true")
	}
	if task.ExportGetBool(row, "f") {
		t.Error("false")
	}
	if !task.ExportGetBool(row, "f1") {
		t.Error("float64(1) should be true")
	}
	if task.ExportGetBool(row, "f0") {
		t.Error("float64(0) should be false")
	}
	if !task.ExportGetBool(row, "i1") {
		t.Error("int64(1) should be true")
	}
	if task.ExportGetBool(row, "i0") {
		t.Error("int64(0) should be false")
	}
	if task.ExportGetBool(row, "missing") {
		t.Error("missing should be false")
	}
}

// --- getTime ---

func TestGetTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	row := map[string]interface{}{
		"time_obj": now,
		"rfc3339":  now.Format(time.RFC3339),
		"datetime": now.Format("2006-01-02 15:04:05"),
		"bad_str":  "not-a-time",
		"nil_v":    nil,
	}
	if task.ExportGetTime(row, "time_obj") == nil {
		t.Error("time.Time object should parse")
	}
	if task.ExportGetTime(row, "rfc3339") == nil {
		t.Error("RFC3339 string should parse")
	}
	if task.ExportGetTime(row, "datetime") == nil {
		t.Error("datetime string should parse")
	}
	if task.ExportGetTime(row, "bad_str") != nil {
		t.Error("invalid string should return nil")
	}
	if task.ExportGetTime(row, "nil_v") != nil {
		t.Error("nil should return nil")
	}
	if task.ExportGetTime(row, "missing") != nil {
		t.Error("missing key should return nil")
	}
}

// --- rowToTask ---

func TestRowToTaskFullRow(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	row := map[string]interface{}{
		"chat_id":        "chat-abc",
		"column_id":      "col-001",
		"position":       float64(3),
		"pinned":         true,
		"priority":       "high",
		"run_status":     "running",
		"progress":       float64(50),
		"current_step":   "Compiling...",
		"error_message":  nil,
		"duration":       float64(120),
		"run_count":      float64(2),
		"computer_id":    "comp-001",
		"computer_mode":  "sandbox",
		"sandbox_type":   "docker",
		"title":          "My Task",
		"assistant_id":   "asst-001",
		"assistant_name": "CodeBot",
		"last_workspace": "/home/user",
		"last_connector": "gpt-4",
		"board_id":       "board-001",
		"started_at":     now,
		"completed_at":   nil,
		"created_at":     now.Format(time.RFC3339),
		"updated_at":     now.Format("2006-01-02 15:04:05"),
		"tags":           `["go","kanban"]`,
	}

	tk := task.ExportRowToTask(row)

	if tk.ChatID != "chat-abc" {
		t.Errorf("ChatID = %q", tk.ChatID)
	}
	if tk.ColumnID == nil || *tk.ColumnID != "col-001" {
		t.Errorf("ColumnID = %v", tk.ColumnID)
	}
	if tk.Position != 3 {
		t.Errorf("Position = %d", tk.Position)
	}
	if !tk.Pinned {
		t.Error("Pinned should be true")
	}
	if tk.Priority != "high" {
		t.Errorf("Priority = %q", tk.Priority)
	}
	if tk.RunStatus != "running" {
		t.Errorf("RunStatus = %q", tk.RunStatus)
	}
	if tk.Progress != 50 {
		t.Errorf("Progress = %d", tk.Progress)
	}
	if tk.CurrentStep == nil || *tk.CurrentStep != "Compiling..." {
		t.Error("CurrentStep")
	}
	if tk.ErrorMessage != nil {
		t.Error("ErrorMessage should be nil")
	}
	if tk.Duration != 120 {
		t.Errorf("Duration = %d", tk.Duration)
	}
	if tk.RunCount != 2 {
		t.Errorf("RunCount = %d", tk.RunCount)
	}
	if tk.ComputerID == nil || *tk.ComputerID != "comp-001" {
		t.Error("ComputerID")
	}
	if tk.ComputerMode == nil || *tk.ComputerMode != "sandbox" {
		t.Error("ComputerMode")
	}
	if tk.SandboxType == nil || *tk.SandboxType != "docker" {
		t.Error("SandboxType")
	}
	if tk.Title != "My Task" {
		t.Errorf("Title = %q", tk.Title)
	}
	if tk.AssistantID != "asst-001" {
		t.Errorf("AssistantID = %q", tk.AssistantID)
	}
	if tk.AssistantName != "CodeBot" {
		t.Errorf("AssistantName = %q", tk.AssistantName)
	}
	if tk.LastWorkspace == nil || *tk.LastWorkspace != "/home/user" {
		t.Error("LastWorkspace")
	}
	if tk.BoardID == nil || *tk.BoardID != "board-001" {
		t.Error("BoardID")
	}
	if tk.StartedAt == nil {
		t.Error("StartedAt should not be nil")
	}
	if tk.CompletedAt != nil {
		t.Error("CompletedAt should be nil")
	}
	if tk.CreatedAt == nil {
		t.Error("CreatedAt should not be nil")
	}
	if len(tk.Tags) != 2 || tk.Tags[0] != "go" || tk.Tags[1] != "kanban" {
		t.Errorf("Tags = %v", tk.Tags)
	}
}

func TestRowToTaskTagsParsing(t *testing.T) {
	base := map[string]interface{}{
		"chat_id": "c1", "position": float64(0), "pinned": false,
		"priority": "none", "run_status": "pending",
		"progress": float64(0), "duration": float64(0), "run_count": float64(0),
	}

	tests := []struct {
		name     string
		tags     interface{}
		expected int
	}{
		{"nil tags", nil, 0},
		{"json array", `["a","b","c"]`, 3},
		{"empty json", `[]`, 0},
		{"interface slice", []interface{}{"x", "y"}, 2},
		{"invalid json", "not json", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := make(map[string]interface{})
			for k, v := range base {
				row[k] = v
			}
			row["tags"] = tt.tags
			tk := task.ExportRowToTask(row)
			if len(tk.Tags) != tt.expected {
				t.Errorf("Tags len = %d, want %d", len(tk.Tags), tt.expected)
			}
		})
	}
}

func TestRowToTaskDefaults(t *testing.T) {
	row := map[string]interface{}{
		"chat_id": "c1", "position": float64(0), "pinned": false,
		"priority": "", "run_status": "",
		"progress": float64(0), "duration": float64(0), "run_count": float64(0),
	}
	tk := task.ExportRowToTask(row)
	if tk.Priority != "none" {
		t.Errorf("empty priority should default to 'none', got %q", tk.Priority)
	}
	if tk.RunStatus != "pending" {
		t.Errorf("empty run_status should default to 'pending', got %q", tk.RunStatus)
	}
	if tk.ColumnID != nil {
		t.Error("ColumnID should be nil when absent")
	}
}

// --- CreateFromWSReq metadata ---

func TestCreateFromWSReqMetadata(t *testing.T) {
	req := &task.CreateFromWSReq{
		ChatID: "chat-123",
		Title:  "Test",
		Metadata: map[string]any{
			"column_id":     "col-abc",
			"assistant_id":  "asst-xyz",
			"computer_id":   "comp-1",
			"computer_mode": "sandbox",
		},
	}
	if task.ExportMetaString(req.Metadata, "column_id") != "col-abc" {
		t.Error("column_id")
	}
	if task.ExportMetaString(req.Metadata, "assistant_id") != "asst-xyz" {
		t.Error("assistant_id")
	}
	if task.ExportMetaString(req.Metadata, "computer_id") != "comp-1" {
		t.Error("computer_id")
	}
}

func TestCreateFromWSReqNilMetadata(t *testing.T) {
	req := &task.CreateFromWSReq{ChatID: "c1", Title: "T"}
	if task.ExportMetaString(req.Metadata, "column_id") != "" {
		t.Error("nil metadata should always return empty")
	}
}
