package events

import (
	"context"
	"strings"
	"sync"

	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// TriggerFunc is the callback for triggering a robot execution.
// Injected by the api package at startup to break the import cycle.
// Returns (executionID, accepted, error).
type TriggerFunc func(ctx *robottypes.Context, memberID string, triggerType robottypes.TriggerType, data interface{}) (string, bool, error)

// ReplyFunc is the callback for replying to the originating channel.
// Injected by the dispatcher at startup. The implementation routes the
// reply to the correct adapter based on metadata.Channel.
// msg is the standard assistant message (text, Content Parts, media, etc.).
type ReplyFunc func(ctx context.Context, msg *agentcontext.Message, metadata *MessageMetadata) error

var (
	triggerFn   TriggerFunc
	triggerFnMu sync.RWMutex

	replyFn   ReplyFunc
	replyFnMu sync.RWMutex
)

// RegisterTriggerFunc sets the function used by handleMessage to trigger
// robot execution when a confirmed action is detected.
func RegisterTriggerFunc(fn TriggerFunc) {
	triggerFnMu.Lock()
	defer triggerFnMu.Unlock()
	triggerFn = fn
}

func getTriggerFunc() TriggerFunc {
	triggerFnMu.RLock()
	defer triggerFnMu.RUnlock()
	return triggerFn
}

// RegisterReplyFunc sets the function used by handleMessage to reply
// to the originating channel after processing.
func RegisterReplyFunc(fn ReplyFunc) {
	replyFnMu.Lock()
	defer replyFnMu.Unlock()
	replyFn = fn
}

func getReplyFunc() ReplyFunc {
	replyFnMu.RLock()
	defer replyFnMu.RUnlock()
	return replyFn
}

// Robot event type constants for event.Push integration.
// Events are fire-and-forget; handlers are registered via event.Register().
const (
	TaskNeedInput = "robot.task.need_input"
	TaskFailed    = "robot.task.failed"
	TaskCompleted = "robot.task.completed"
	ExecWaiting   = "robot.exec.waiting"
	ExecResumed   = "robot.exec.resumed"
	ExecCompleted = "robot.exec.completed"
	ExecFailed    = "robot.exec.failed"
	ExecCancelled = "robot.exec.cancelled"
	ExecRecovered = "robot.exec.recovered"
	Delivery      = "robot.delivery"
	Message       = "robot.message"
)

// Robot configuration change events (used by integrations Receiver).
const (
	RobotConfigCreated = "robot.config.created"
	RobotConfigUpdated = "robot.config.updated"
	RobotConfigDeleted = "robot.config.deleted"
)

// NeedInputPayload is the event payload for TaskNeedInput / ExecWaiting events.
type NeedInputPayload struct {
	ExecutionID string `json:"execution_id"`
	MemberID    string `json:"member_id"`
	TeamID      string `json:"team_id"`
	TaskID      string `json:"task_id"`
	Question    string `json:"question"`
	ChatID      string `json:"chat_id,omitempty"`
}

// ExecPayload is a generic execution event payload.
type ExecPayload struct {
	ExecutionID string `json:"execution_id"`
	MemberID    string `json:"member_id"`
	TeamID      string `json:"team_id"`
	Status      string `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
}

// TaskPayload is the event payload for TaskFailed / TaskCompleted events.
type TaskPayload struct {
	ExecutionID string `json:"execution_id"`
	MemberID    string `json:"member_id"`
	TeamID      string `json:"team_id"`
	TaskID      string `json:"task_id"`
	Error       string `json:"error,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
}

// DeliveryPayload is the event payload for Delivery events.
type DeliveryPayload struct {
	ExecutionID string                          `json:"execution_id"`
	MemberID    string                          `json:"member_id"`
	TeamID      string                          `json:"team_id"`
	ChatID      string                          `json:"chat_id,omitempty"`
	Content     *robottypes.DeliveryContent     `json:"content,omitempty"`
	Preferences *robottypes.DeliveryPreferences `json:"preferences,omitempty"`
	Extra       map[string]any                  `json:"extra,omitempty"`
}

// MessagePayload is the event payload for Message events (external channel messages).
type MessagePayload struct {
	RobotID  string                 `json:"robot_id"`
	Messages []agentcontext.Message `json:"messages"`
	Metadata *MessageMetadata       `json:"metadata"`
}

// MessageMetadata carries channel-specific information for routing and deduplication.
type MessageMetadata struct {
	Channel    string         `json:"channel"`
	MessageID  string         `json:"message_id,omitempty"`
	AppID      string         `json:"app_id,omitempty"`
	ChatID     string         `json:"chat_id,omitempty"`
	SenderID   string         `json:"sender_id,omitempty"`
	SenderName string         `json:"sender_name,omitempty"`
	Locale     string         `json:"locale,omitempty"`
	ReplyTo    string         `json:"reply_to,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// MessageResult is the result returned from handleMessage via event.Call.
type MessageResult struct {
	Message     *agentcontext.Message `json:"message,omitempty"`
	Action      *ActionResult         `json:"action,omitempty"`
	ExecutionID string                `json:"execution_id,omitempty"`
	Metadata    *MessageMetadata      `json:"metadata,omitempty"`
}

// ActionResult describes a detected action from the Host Agent's Next hook.
type ActionResult struct {
	Name    string `json:"name"`
	Payload any    `json:"payload,omitempty"`
}

// RobotConfigPayload is the event payload for robot.config.* events.
type RobotConfigPayload struct {
	MemberID string `json:"member_id"`
	TeamID   string `json:"team_id"`
}

// NormalizeLocale converts various language code formats (IETF BCP 47, etc.)
// into the lowercase hyphenated form used by agentcontext (e.g. "zh-cn", "en-us").
//
// Mapping rules:
//
//	"zh-hans", "zh-cn"           → "zh-cn"
//	"zh-hant", "zh-tw", "zh-hk" → "zh-tw"
//	"zh"                         → "zh-cn"
//	"en-us"                      → "en-us"
//	"en-gb"                      → "en-gb"
//	"en"                         → "en"
//	""                           → "en"  (default)
//	other                        → lowercased as-is
func NormalizeLocale(raw string) string {
	code := strings.ToLower(strings.TrimSpace(raw))
	if code == "" {
		return "en"
	}

	// Normalize underscore to hyphen (e.g. zh_CN → zh-cn)
	code = strings.ReplaceAll(code, "_", "-")

	switch code {
	case "zh-hans", "zh-cn":
		return "zh-cn"
	case "zh-hant", "zh-tw", "zh-hk":
		return "zh-tw"
	case "zh":
		return "zh-cn"
	default:
		return code
	}
}
