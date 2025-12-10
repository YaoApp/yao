package context

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Chat Buffer - Buffers messages and steps during execution for batch saving
// =============================================================================

// ChatBuffer buffers messages and resume steps during agent execution
// All data is held in memory and batch-written at the end of Stream()
type ChatBuffer struct {
	// Identity
	chatID      string
	requestID   string
	assistantID string
	connector   string // Current connector ID (for data analysis)
	mode        string // Current chat mode (chat or task)

	// Message buffer
	messages    []*BufferedMessage
	msgSequence int

	// Step buffer (for Resume)
	steps        []*BufferedStep
	currentStep  *BufferedStep
	stepSequence int

	// Space snapshot (captured when step starts, for recovery)
	spaceSnapshot map[string]interface{}

	mu sync.Mutex
}

// BufferedMessage represents a message waiting to be saved
type BufferedMessage struct {
	MessageID   string                 `json:"message_id"`
	ChatID      string                 `json:"chat_id"`
	RequestID   string                 `json:"request_id,omitempty"`
	Role        string                 `json:"role"` // "user" or "assistant"
	Type        string                 `json:"type"` // "text", "image", "loading", "tool_call", "retrieval", etc.
	Props       map[string]interface{} `json:"props"`
	BlockID     string                 `json:"block_id,omitempty"`
	ThreadID    string                 `json:"thread_id,omitempty"`
	AssistantID string                 `json:"assistant_id,omitempty"`
	Connector   string                 `json:"connector,omitempty"` // Connector ID used for this message
	Mode        string                 `json:"mode,omitempty"`      // Chat mode used for this message (chat or task)
	Sequence    int                    `json:"sequence"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	IsStreaming bool                   `json:"-"` // Internal flag: true if message is still streaming (not saved until End)
}

// BufferedStep represents an execution step waiting to be saved (for Resume)
// Only saved when request is interrupted or failed
type BufferedStep struct {
	ResumeID      string                 `json:"resume_id"`
	ChatID        string                 `json:"chat_id"`
	RequestID     string                 `json:"request_id"`
	AssistantID   string                 `json:"assistant_id"`
	StackID       string                 `json:"stack_id"`
	StackParentID string                 `json:"stack_parent_id,omitempty"`
	StackDepth    int                    `json:"stack_depth"`
	Type          string                 `json:"type"`   // "input", "hook_create", "llm", "tool", "hook_next", "delegate"
	Status        string                 `json:"status"` // "running", "completed", "failed", "interrupted"
	Input         map[string]interface{} `json:"input,omitempty"`
	Output        map[string]interface{} `json:"output,omitempty"`
	SpaceSnapshot map[string]interface{} `json:"space_snapshot,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Sequence      int                    `json:"sequence"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// Step status constants (internal use only, not stored in database)
const (
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
)

// Step type constants
const (
	StepTypeInput      = "input"
	StepTypeHookCreate = "hook_create"
	StepTypeLLM        = "llm"
	StepTypeTool       = "tool"
	StepTypeHookNext   = "hook_next"
	StepTypeDelegate   = "delegate"
)

// Resume status constants (for database storage)
const (
	ResumeStatusFailed      = "failed"
	ResumeStatusInterrupted = "interrupted"
)

// NewChatBuffer creates a new chat buffer
func NewChatBuffer(chatID, requestID, assistantID, connector, mode string) *ChatBuffer {
	return &ChatBuffer{
		chatID:      chatID,
		requestID:   requestID,
		assistantID: assistantID,
		connector:   connector,
		mode:        mode,
		messages:    make([]*BufferedMessage, 0),
		steps:       make([]*BufferedStep, 0),
	}
}

// =============================================================================
// Message Buffer Methods
// =============================================================================

// AddMessage adds a message to the buffer
func (b *ChatBuffer) AddMessage(msg *BufferedMessage) {
	if msg == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Auto-generate IDs if not provided
	if msg.MessageID == "" {
		msg.MessageID = uuid.New().String()
	}
	if msg.ChatID == "" {
		msg.ChatID = b.chatID
	}
	if msg.RequestID == "" {
		msg.RequestID = b.requestID
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// Set mode from buffer if not provided
	if msg.Mode == "" && b.mode != "" {
		msg.Mode = b.mode
	}

	// Auto-increment sequence
	b.msgSequence++
	msg.Sequence = b.msgSequence

	b.messages = append(b.messages, msg)
}

// AddUserInput adds user input message to the buffer
func (b *ChatBuffer) AddUserInput(content interface{}, name string) {
	props := map[string]interface{}{
		"content": content,
		"role":    "user",
	}
	if name != "" {
		props["name"] = name
	}

	b.AddMessage(&BufferedMessage{
		Role:  "user",
		Type:  "user_input",
		Props: props,
	})
}

// AddAssistantMessage adds an assistant message to the buffer
// This is called by ctx.Send() to buffer messages for batch saving
func (b *ChatBuffer) AddAssistantMessage(messageID, msgType string, props map[string]interface{}, blockID, threadID, assistantID string, metadata map[string]interface{}) {
	// Skip event type messages (transient, not stored)
	if msgType == "event" {
		return
	}

	b.AddMessage(&BufferedMessage{
		MessageID:   messageID, // Use the same MessageID as sent to client
		Role:        "assistant",
		Type:        msgType,
		Props:       props,
		BlockID:     blockID,
		ThreadID:    threadID,
		AssistantID: assistantID,
		Connector:   b.connector, // Use current connector
		Metadata:    metadata,
	})
}

// AddStreamingMessage adds a streaming message to the buffer
// Streaming messages are not saved until CompleteStreamingMessage is called
// This is called by ctx.SendStream() to start a streaming message
func (b *ChatBuffer) AddStreamingMessage(messageID, msgType string, props map[string]interface{}, blockID, threadID, assistantID string, metadata map[string]interface{}) {
	// Skip event type messages (transient, not stored)
	if msgType == "event" {
		return
	}

	// Deep copy props to avoid mutation issues
	propsCopy := make(map[string]interface{})
	for k, v := range props {
		propsCopy[k] = v
	}

	b.AddMessage(&BufferedMessage{
		MessageID:   messageID, // Use provided message ID
		Role:        "assistant",
		Type:        msgType,
		Props:       propsCopy,
		BlockID:     blockID,
		ThreadID:    threadID,
		AssistantID: assistantID,
		Connector:   b.connector,
		Metadata:    metadata,
		IsStreaming: true, // Mark as streaming
	})
}

// AppendMessageContent appends content to a streaming message
// This is called by ctx.Append() to accumulate content
func (b *ChatBuffer) AppendMessageContent(messageID string, content string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Find the message by ID
	for _, msg := range b.messages {
		if msg.MessageID == messageID && msg.IsStreaming {
			// Append to existing content
			if msg.Props == nil {
				msg.Props = make(map[string]interface{})
			}
			if existing, ok := msg.Props["content"].(string); ok {
				msg.Props["content"] = existing + content
			} else {
				msg.Props["content"] = content
			}
			return true
		}
	}
	return false
}

// CompleteStreamingMessage marks a streaming message as complete
// This is called by ctx.End() to finalize the message
// Returns the complete content for the message_end event
func (b *ChatBuffer) CompleteStreamingMessage(messageID string) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Find the message by ID
	for _, msg := range b.messages {
		if msg.MessageID == messageID && msg.IsStreaming {
			msg.IsStreaming = false
			// Return the accumulated content
			if content, ok := msg.Props["content"].(string); ok {
				return content, true
			}
			return "", true
		}
	}
	return "", false
}

// GetStreamingMessage returns a streaming message by ID
func (b *ChatBuffer) GetStreamingMessage(messageID string) *BufferedMessage {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, msg := range b.messages {
		if msg.MessageID == messageID && msg.IsStreaming {
			return msg
		}
	}
	return nil
}

// GetMessages returns all buffered messages
func (b *ChatBuffer) GetMessages() []*BufferedMessage {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]*BufferedMessage, len(b.messages))
	copy(result, b.messages)
	return result
}

// GetMessageCount returns the number of buffered messages
func (b *ChatBuffer) GetMessageCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.messages)
}

// =============================================================================
// Step Buffer Methods (for Resume)
// =============================================================================

// BeginStep starts tracking a new execution step
// Returns the step for further updates
func (b *ChatBuffer) BeginStep(stepType string, input map[string]interface{}, stack *Stack) *BufferedStep {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stepSequence++

	step := &BufferedStep{
		ResumeID:    uuid.New().String(),
		ChatID:      b.chatID,
		RequestID:   b.requestID,
		AssistantID: b.assistantID,
		Type:        stepType,
		Status:      StepStatusRunning,
		Input:       input,
		Sequence:    b.stepSequence,
		CreatedAt:   time.Now(),
	}

	// Set stack information if available
	if stack != nil {
		step.StackID = stack.ID
		step.StackParentID = stack.ParentID
		step.StackDepth = stack.Depth
	}

	// Capture current space snapshot
	if b.spaceSnapshot != nil {
		step.SpaceSnapshot = copyMap(b.spaceSnapshot)
	}

	b.steps = append(b.steps, step)
	b.currentStep = step

	return step
}

// CompleteStep marks the current step as completed
func (b *ChatBuffer) CompleteStep(output map[string]interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.currentStep != nil {
		b.currentStep.Output = output
		b.currentStep.Status = StepStatusCompleted
		b.currentStep = nil
	}
}

// FailCurrentStep marks the current step as failed or interrupted
func (b *ChatBuffer) FailCurrentStep(status string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.currentStep != nil && b.currentStep.Status == StepStatusRunning {
		b.currentStep.Status = status
		if err != nil {
			b.currentStep.Error = err.Error()
		}
	}
}

// GetCurrentStep returns the current running step
func (b *ChatBuffer) GetCurrentStep() *BufferedStep {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currentStep
}

// GetStepsForResume returns steps that need to be saved for resume
// Only returns steps with failed or interrupted status
func (b *ChatBuffer) GetStepsForResume(finalStatus string) []*BufferedStep {
	b.mu.Lock()
	defer b.mu.Unlock()

	// If completed successfully, no steps need to be saved
	if finalStatus == StepStatusCompleted {
		return nil
	}

	// Mark current running step with final status
	if b.currentStep != nil && b.currentStep.Status == StepStatusRunning {
		b.currentStep.Status = finalStatus
	}

	// Return all steps (they will all have the context for recovery)
	result := make([]*BufferedStep, len(b.steps))
	copy(result, b.steps)
	return result
}

// GetAllSteps returns all buffered steps (for debugging/testing)
func (b *ChatBuffer) GetAllSteps() []*BufferedStep {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]*BufferedStep, len(b.steps))
	copy(result, b.steps)
	return result
}

// =============================================================================
// Space Snapshot Methods
// =============================================================================

// SetSpaceSnapshot sets the space snapshot for recovery
// Should be called when space data changes
func (b *ChatBuffer) SetSpaceSnapshot(snapshot map[string]interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.spaceSnapshot = copyMap(snapshot)
}

// GetSpaceSnapshot returns the current space snapshot
func (b *ChatBuffer) GetSpaceSnapshot() map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	return copyMap(b.spaceSnapshot)
}

// =============================================================================
// Identity Methods
// =============================================================================

// ChatID returns the chat ID
func (b *ChatBuffer) ChatID() string {
	return b.chatID
}

// RequestID returns the request ID
func (b *ChatBuffer) RequestID() string {
	return b.requestID
}

// AssistantID returns the assistant ID
func (b *ChatBuffer) AssistantID() string {
	return b.assistantID
}

// SetAssistantID updates the assistant ID (for A2A calls)
func (b *ChatBuffer) SetAssistantID(assistantID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.assistantID = assistantID
}

// Connector returns the current connector ID
func (b *ChatBuffer) Connector() string {
	return b.connector
}

// SetConnector updates the connector ID (when user switches connector)
func (b *ChatBuffer) SetConnector(connector string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connector = connector
}

// Mode returns the current chat mode
func (b *ChatBuffer) Mode() string {
	return b.mode
}

// SetMode updates the chat mode (when user switches mode)
func (b *ChatBuffer) SetMode(mode string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mode = mode
}

// =============================================================================
// Helper Functions
// =============================================================================

// copyMap creates a shallow copy of a map
func copyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
