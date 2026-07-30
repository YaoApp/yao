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

// --- InboxStats ---

func TestInboxStatsFields(t *testing.T) {
	s := &inbox.InboxStats{All: 10, Bookmarked: 3, Input: 4, Completed: 2, Failed: 1, Archived: 2}
	if s.All != 10 {
		t.Errorf("All = %d, want 10", s.All)
	}
	if s.Bookmarked != 3 {
		t.Errorf("Bookmarked = %d, want 3", s.Bookmarked)
	}
}

// --- AgentMail struct ---

func TestAgentMailFieldBinding(t *testing.T) {
	m := &inbox.AgentMail{
		MailID: "mail-001", Type: "input", Priority: "high",
		Title: "Needs input", ChatID: "chat-x",
		Bookmarked: true, InboxPinned: true, HasUnread: true,
	}
	if m.Type != "input" {
		t.Error("Type")
	}
	if m.Priority != "high" {
		t.Error("Priority")
	}
	if !m.Bookmarked {
		t.Error("Bookmarked should be true")
	}
	if !m.InboxPinned {
		t.Error("InboxPinned should be true")
	}
	if !m.HasUnread {
		t.Error("HasUnread should be true")
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
	row := inbox.XunR{
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
	if m.CreatedAt == nil {
		t.Error("CreatedAt should not be nil")
	}
	if m.UpdatedAt == nil {
		t.Error("UpdatedAt should not be nil")
	}
}

func TestRowToMailMinimal(t *testing.T) {
	row := inbox.XunR{
		"mail_id":  "m1",
		"type":     "input",
		"priority": "high",
		"title":    "Need input",
		"chat_id":  "c1",
	}
	m := inbox.ExportRowToMail(row)
	if m.MailID != "m1" {
		t.Error("MailID")
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
	row := inbox.XunR{"k": "v", "nil": nil, "num": 42}
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
	row := inbox.XunR{"f": float64(10), "i64": int64(20), "i": 30}
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
	row := inbox.XunR{"t": true, "f": false, "f1": float64(1), "i0": int64(0)}
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
	row := inbox.XunR{
		"t":          now,
		"rfc":        now.Format(time.RFC3339),
		"rfc_nano":   now.Format(time.RFC3339Nano),
		"sqlite_tz":  now.Format("2006-01-02 15:04:05.999999999-07:00"),
		"sqlite_tz2": now.Format("2006-01-02 15:04:05-07:00"),
		"sqlite_ns":  now.Format("2006-01-02 15:04:05.999999999"),
		"dt":         now.Format("2006-01-02 15:04:05"),
		"bad":        "x",
		"nil":        nil,
	}
	if inbox.ExportGetTime(row, "t") == nil {
		t.Error("time.Time")
	}
	if inbox.ExportGetTime(row, "rfc") == nil {
		t.Error("RFC3339")
	}
	if inbox.ExportGetTime(row, "rfc_nano") == nil {
		t.Error("RFC3339Nano")
	}
	if inbox.ExportGetTime(row, "sqlite_tz") == nil {
		t.Error("sqlite with fractional seconds and timezone")
	}
	if inbox.ExportGetTime(row, "sqlite_tz2") == nil {
		t.Error("sqlite with timezone no fractional")
	}
	if inbox.ExportGetTime(row, "sqlite_ns") == nil {
		t.Error("sqlite with fractional seconds no timezone")
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
