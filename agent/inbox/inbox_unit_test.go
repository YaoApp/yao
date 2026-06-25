//go:build unit

package inbox_test

import (
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/inbox"
)

// --- ListQuery ---

func TestListQueryDefaults(t *testing.T) {
	q := &inbox.ListQuery{}
	if q.Size != 0 {
		t.Errorf("Size = %d, want 0 before service call", q.Size)
	}
	if q.Filter != "" {
		t.Errorf("Filter = %q, want empty", q.Filter)
	}
	if q.Page != 0 {
		t.Errorf("Page = %d, want 0", q.Page)
	}
	if q.ChatID != "" {
		t.Errorf("ChatID = %q, want empty", q.ChatID)
	}
}

func TestListQueryChatIDField(t *testing.T) {
	q := &inbox.ListQuery{
		Filter: "all",
		ChatID: "chat-abc-123",
		Page:   1,
		Size:   50,
	}
	if q.ChatID != "chat-abc-123" {
		t.Errorf("ChatID = %q, want %q", q.ChatID, "chat-abc-123")
	}
	if q.Filter != "all" {
		t.Errorf("Filter = %q, want %q", q.Filter, "all")
	}
	if q.Size != 50 {
		t.Errorf("Size = %d, want 50", q.Size)
	}
}

// --- Counts ---

func TestCountsAggregation(t *testing.T) {
	c := &inbox.Counts{Input: 3, Completed: 5, Failed: 2, Total: 10}
	sum := c.Input + c.Completed + c.Failed
	if c.Total != sum {
		t.Errorf("Total=%d != Input+Completed+Failed=%d", c.Total, sum)
	}
}

// --- AgentMail struct ---

func TestAgentMailFieldBinding(t *testing.T) {
	m := &inbox.AgentMail{
		MailID: "mail-001", Type: "input", Priority: "high",
		Title: "Needs input", ChatID: "chat-x",
		Read: false, Starred: true, Pinned: true,
	}
	if m.Type != "input" {
		t.Error("Type")
	}
	if m.Priority != "high" {
		t.Error("Priority")
	}
	if m.Read {
		t.Error("Read should be false")
	}
	if !m.Starred {
		t.Error("Starred should be true")
	}
	if !m.Pinned {
		t.Error("Pinned should be true")
	}
}

// --- AgentTask trigger struct ---

func TestAgentTaskActiveVsDeleted(t *testing.T) {
	active := &inbox.AgentTask{ChatID: "c1", CreatedBy: "u1", TeamID: "t1"}
	if active.DeletedAt != nil {
		t.Error("active task DeletedAt should be nil")
	}

	now := time.Now()
	deleted := &inbox.AgentTask{ChatID: "c2", DeletedAt: &now}
	if deleted.DeletedAt == nil {
		t.Error("deleted task should have non-nil DeletedAt")
	}
}

// --- rowToMail ---

func TestRowToMailFull(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	row := map[string]interface{}{
		"mail_id":      "mail-abc",
		"type":         "completed",
		"priority":     "low",
		"title":        "Task Done",
		"body":         "Successfully finished",
		"chat_id":      "chat-xyz",
		"assistant_id": "asst-001",
		"source_type":  "kanban",
		"source_id":    "board-001",
		"source_name":  "Dev Board",
		"read":         true,
		"starred":      true,
		"pinned":       false,
		"read_at":      now,
		"created_at":   now.Format(time.RFC3339),
		"updated_at":   now.Format("2006-01-02 15:04:05"),
	}

	m := inbox.ExportRowToMail(row)

	if m.MailID != "mail-abc" {
		t.Errorf("MailID = %q", m.MailID)
	}
	if m.Type != "completed" {
		t.Errorf("Type = %q", m.Type)
	}
	if m.Priority != "low" {
		t.Errorf("Priority = %q", m.Priority)
	}
	if m.Title != "Task Done" {
		t.Errorf("Title = %q", m.Title)
	}
	if m.Body != "Successfully finished" {
		t.Errorf("Body = %q", m.Body)
	}
	if m.ChatID != "chat-xyz" {
		t.Errorf("ChatID = %q", m.ChatID)
	}
	if m.AssistantID != "asst-001" {
		t.Errorf("AssistantID = %q", m.AssistantID)
	}
	if m.SourceType != "kanban" {
		t.Errorf("SourceType = %q", m.SourceType)
	}
	if m.SourceName != "Dev Board" {
		t.Errorf("SourceName = %q", m.SourceName)
	}
	if !m.Read {
		t.Error("Read should be true")
	}
	if !m.Starred {
		t.Error("Starred should be true")
	}
	if m.Pinned {
		t.Error("Pinned should be false")
	}
	if m.ReadAt == nil {
		t.Error("ReadAt should not be nil")
	}
	if m.CreatedAt == nil {
		t.Error("CreatedAt should not be nil")
	}
	if m.UpdatedAt == nil {
		t.Error("UpdatedAt should not be nil")
	}
}

func TestRowToMailMinimal(t *testing.T) {
	row := map[string]interface{}{
		"mail_id": "m1", "type": "input", "priority": "high",
		"title": "Need input", "chat_id": "c1",
		"read": false, "starred": false, "pinned": false,
	}
	m := inbox.ExportRowToMail(row)
	if m.MailID != "m1" {
		t.Error("MailID")
	}
	if m.ReadAt != nil {
		t.Error("ReadAt should be nil")
	}
	if m.Body != "" {
		t.Errorf("Body should be empty, got %q", m.Body)
	}
	if m.AssistantID != "" {
		t.Error("AssistantID should be empty")
	}
}

// --- helper functions ---

func TestHelperGetString(t *testing.T) {
	row := map[string]interface{}{"k": "v", "nil": nil, "num": 42}
	if inbox.ExportGetString(row, "k") != "v" {
		t.Error("valid string")
	}
	if inbox.ExportGetString(row, "nil") != "" {
		t.Error("nil")
	}
	if inbox.ExportGetString(row, "num") != "" {
		t.Error("non-string")
	}
	if inbox.ExportGetString(row, "missing") != "" {
		t.Error("missing")
	}
}

func TestHelperGetInt(t *testing.T) {
	row := map[string]interface{}{"f": float64(10), "i64": int64(20), "i": 30}
	if inbox.ExportGetInt(row, "f") != 10 {
		t.Error("float64")
	}
	if inbox.ExportGetInt(row, "i64") != 20 {
		t.Error("int64")
	}
	if inbox.ExportGetInt(row, "i") != 30 {
		t.Error("int")
	}
	if inbox.ExportGetInt(row, "missing") != 0 {
		t.Error("missing")
	}
}

func TestHelperGetBool(t *testing.T) {
	row := map[string]interface{}{"t": true, "f": false, "f1": float64(1), "i0": int64(0)}
	if !inbox.ExportGetBool(row, "t") {
		t.Error("true")
	}
	if inbox.ExportGetBool(row, "f") {
		t.Error("false")
	}
	if !inbox.ExportGetBool(row, "f1") {
		t.Error("float64(1)")
	}
	if inbox.ExportGetBool(row, "i0") {
		t.Error("int64(0)")
	}
}

func TestHelperGetTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	row := map[string]interface{}{
		"t": now, "rfc": now.Format(time.RFC3339),
		"dt": now.Format("2006-01-02 15:04:05"), "bad": "x", "nil": nil,
	}
	if inbox.ExportGetTime(row, "t") == nil {
		t.Error("time.Time")
	}
	if inbox.ExportGetTime(row, "rfc") == nil {
		t.Error("RFC3339")
	}
	if inbox.ExportGetTime(row, "dt") == nil {
		t.Error("datetime")
	}
	if inbox.ExportGetTime(row, "bad") != nil {
		t.Error("invalid")
	}
	if inbox.ExportGetTime(row, "nil") != nil {
		t.Error("nil")
	}
}
