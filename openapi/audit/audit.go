package audit

import (
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
)

// Entry represents a single audit log record.
type Entry struct {
	EventID        string         `json:"event_id,omitempty"`
	Operation      string         `json:"operation"`
	Category       string         `json:"category,omitempty"`
	Severity       string         `json:"severity,omitempty"`
	UserID         string         `json:"user_id"`
	TeamID         string         `json:"team_id,omitempty"`
	UserName       string         `json:"user_name,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	ClientIP       string         `json:"client_ip,omitempty"`
	TargetResource string         `json:"target_resource,omitempty"`
	ResourceType   string         `json:"resource_type,omitempty"`
	Source         string         `json:"source,omitempty"`
	Application    string         `json:"application,omitempty"`
	Success        bool           `json:"success"`
	Details        map[string]any `json:"details,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`
}

// Record writes an audit entry asynchronously via the audit_log model.
// Failures are logged but never block the caller.
func Record(entry Entry) {
	if entry.EventID == "" {
		entry.EventID = uuid.New().String()
	}
	if entry.Severity == "" {
		entry.Severity = "medium"
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("[audit] panic writing audit log: %v", r)
			}
		}()

		row := map[string]any{
			"event_id":        entry.EventID,
			"operation":       entry.Operation,
			"category":        entry.Category,
			"severity":        entry.Severity,
			"user_id":         entry.UserID,
			"team_id":         entry.TeamID,
			"user_name":       entry.UserName,
			"session_id":      entry.SessionID,
			"client_ip":       entry.ClientIP,
			"target_resource": entry.TargetResource,
			"resource_type":   entry.ResourceType,
			"source":          entry.Source,
			"application":     entry.Application,
			"success":         entry.Success,
			"details":         entry.Details,
			"error_message":   entry.ErrorMessage,
			"created_at":      time.Now(),
		}

		p := process.New("models.audit.Save", row)
		if _, err := p.Exec(); err != nil {
			log.Error("[audit] failed to save audit log: %v", err)
		}
	}()
}
