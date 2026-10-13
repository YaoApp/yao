package assistant

import (
	"time"

	"github.com/google/uuid"
	agentcontext "github.com/yaoapp/yao/agent/context"
	storetypes "github.com/yaoapp/yao/agent/store/types"
)

// InitializeConversation prepares conversation context (synchronous)
func (ast *Assistant) InitializeConversation(ctx *agentcontext.Context, options ...*agentcontext.Options) error {
	// Reserved for future conversation initialization logic
	return nil
}

// InitializeConversationAsync prepares conversation context asynchronously
func (ast *Assistant) InitializeConversationAsync(ctx *agentcontext.Context, options ...*agentcontext.Options) {
	go ast.InitializeConversation(ctx, options...)
}

// =============================================================================
// Chat Buffer Integration
// =============================================================================

// InitBuffer initializes the chat buffer for the context
// Should be called at the start of Stream() for root stack only
func (ast *Assistant) InitBuffer(ctx *agentcontext.Context) {
	// Only initialize for root stack
	if ctx.Stack == nil || !ctx.Stack.IsRoot() {
		return
	}

	// Skip if buffer already exists
	if ctx.Buffer != nil {
		return
	}

	// Skip if History is disabled in options
	if ctx.Stack.Options != nil && ctx.Stack.Options.Skip != nil && ctx.Stack.Options.Skip.History {
		ctx.Logger.Debug("Buffer skipped: Skip.History is true")
		return
	}

	// Generate request ID if not set
	requestID := ctx.RequestID()
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Get connector and mode from options
	connector := ""
	mode := ""
	if ctx.Stack.Options != nil {
		connector = ctx.Stack.Options.Connector
		mode = ctx.Stack.Options.Mode
	}

	ctx.Buffer = agentcontext.NewChatBuffer(ctx.ChatID, requestID, ast.ID, connector, mode)
	ctx.Logger.Debug("Buffer initialized: chatID=%s, requestID=%s, assistantID=%s", ctx.ChatID, requestID, ast.ID)
}

// BufferUserInput adds user input messages to the buffer
// Should be called after InitBuffer
func (ast *Assistant) BufferUserInput(ctx *agentcontext.Context, inputMessages []agentcontext.Message) {
	if ctx.Buffer == nil {
		return
	}

	// Only root stack should buffer user input
	// Delegated agents share the same buffer but should not duplicate user input
	if ctx.Stack != nil && !ctx.Stack.IsRoot() {
		return
	}

	// Convert input messages to buffer format
	for _, msg := range inputMessages {
		// Extract content from message
		var content interface{}
		var name string

		content = msg.Content
		if msg.Name != nil {
			name = *msg.Name
		}

		ctx.Buffer.AddUserInput(content, name)
	}
}

// UpdateSpaceSnapshot updates the context memory snapshot in the buffer
// Only captures Context-level memory (request-scoped temporary data) for recovery
func (ast *Assistant) UpdateSpaceSnapshot(ctx *agentcontext.Context) {
	if ctx.Buffer == nil || ctx.Memory == nil || ctx.Memory.Context == nil {
		return
	}

	snapshot := ctx.Memory.Context.Snapshot()
	ctx.Buffer.SetSpaceSnapshot(snapshot)
}

// BeginStep starts tracking an execution step
// Returns the step for further updates
func (ast *Assistant) BeginStep(ctx *agentcontext.Context, stepType string, input map[string]interface{}) *agentcontext.BufferedStep {
	if ctx.Buffer == nil {
		return nil
	}

	// Update space snapshot before beginning step
	ast.UpdateSpaceSnapshot(ctx)

	return ctx.Buffer.BeginStep(stepType, input, ctx.Stack)
}

// CompleteStep marks the current step as completed
func (ast *Assistant) CompleteStep(ctx *agentcontext.Context, output map[string]interface{}) {
	if ctx.Buffer == nil {
		return
	}
	ctx.Buffer.CompleteStep(output)
}

// FlushBuffer saves all buffered data to the database
// Should be called in defer block at the end of Stream()
func (ast *Assistant) FlushBuffer(ctx *agentcontext.Context, finalStatus string, err error) {
	if ctx.Buffer == nil {
		return
	}

	// Only flush for root stack
	if ctx.Stack == nil || !ctx.Stack.IsRoot() {
		return
	}

	// Get chat store
	chatStore := GetChatStore()
	if chatStore == nil {
		ctx.Logger.Error("Chat store not available, cannot flush buffer")
		return
	}

	// Mark current step as failed/interrupted if needed
	if finalStatus != agentcontext.StepStatusCompleted && err != nil {
		ctx.Buffer.FailCurrentStep(finalStatus, err)
	}

	// 1. Save all messages (user input + assistant responses)
	messages := ast.convertBufferedMessages(ctx.Buffer.GetMessages())
	if len(messages) > 0 {
		if saveErr := chatStore.SaveMessages(ctx.ChatID, messages); saveErr != nil {
			ctx.Logger.Error("Failed to save messages: %v", saveErr)
		} else {
			ctx.Logger.Debug("Saved %d messages for chat=%s", len(messages), ctx.ChatID)
		}
	}

	// 2. Update chat last_message_at, last_connector, and last_mode
	if len(messages) > 0 {
		now := time.Now()
		updates := map[string]interface{}{
			"last_message_at": now,
		}
		// Also update last_connector if available
		if connector := ctx.Buffer.Connector(); connector != "" {
			updates["last_connector"] = connector
		}
		// Also update last_mode if available
		if mode := ctx.Buffer.Mode(); mode != "" {
			updates["last_mode"] = mode
		}
		// Also update last_workspace if available in metadata
		if ctx.Metadata != nil {
			if ws, ok := ctx.Metadata["workspace_id"].(string); ok && ws != "" {
				updates["last_workspace"] = ws
			}
		}
		if updateErr := chatStore.UpdateChat(ctx.ChatID, updates); updateErr != nil {
			ctx.Logger.Debug("Failed to update chat: %v", updateErr)
		}
	}

	// 3. Only save resume steps on error/interrupt (not on success)
	if finalStatus != agentcontext.StepStatusCompleted {
		steps := ast.convertBufferedSteps(ctx.Buffer.GetStepsForResume(finalStatus))
		if len(steps) > 0 {
			if saveErr := chatStore.SaveResume(steps); saveErr != nil {
				ctx.Logger.Error("Failed to save resume steps: %v", saveErr)
			} else {
				ctx.Logger.Debug("Saved %d resume steps for chat=%s (status=%s)", len(steps), ctx.ChatID, finalStatus)
			}
		}
	}

	// 4. Close SafeWriter to flush remaining writes (root stack only)
	// This ensures all pending SSE messages are sent before the response completes
	ctx.CloseSafeWriter()
}

// convertBufferedMessages converts BufferedMessage slice to store Message slice.
// Deduplicates by messageID, keeping the last (most complete) entry — this
// prevents unique-index conflicts when the same tool emits multiple terminal
// statuses (e.g. cleanup "error" after a normal "completed").
func (ast *Assistant) convertBufferedMessages(buffered []*agentcontext.BufferedMessage) []*storetypes.Message {
	if len(buffered) == 0 {
		return nil
	}

	seen := make(map[string]int, len(buffered))
	deduped := make([]*agentcontext.BufferedMessage, 0, len(buffered))
	for _, msg := range buffered {
		if idx, ok := seen[msg.MessageID]; ok {
			deduped[idx] = msg
		} else {
			seen[msg.MessageID] = len(deduped)
			deduped = append(deduped, msg)
		}
	}

	messages := make([]*storetypes.Message, len(deduped))
	for i, msg := range deduped {
		messages[i] = &storetypes.Message{
			MessageID:   msg.MessageID,
			ChatID:      msg.ChatID,
			RequestID:   msg.RequestID,
			Role:        msg.Role,
			Type:        msg.Type,
			Props:       msg.Props,
			BlockID:     msg.BlockID,
			ThreadID:    msg.ThreadID,
			AssistantID: msg.AssistantID,
			Connector:   msg.Connector,
			Mode:        msg.Mode,
			Sequence:    msg.Sequence,
			Metadata:    msg.Metadata,
			CreatedAt:   msg.CreatedAt,
			UpdatedAt:   msg.CreatedAt,
		}
	}
	return messages
}

// convertBufferedSteps converts BufferedStep slice to store Resume slice
func (ast *Assistant) convertBufferedSteps(buffered []*agentcontext.BufferedStep) []*storetypes.Resume {
	if len(buffered) == 0 {
		return nil
	}

	steps := make([]*storetypes.Resume, len(buffered))
	for i, step := range buffered {
		steps[i] = &storetypes.Resume{
			ResumeID:      step.ResumeID,
			ChatID:        step.ChatID,
			RequestID:     step.RequestID,
			AssistantID:   step.AssistantID,
			StackID:       step.StackID,
			StackParentID: step.StackParentID,
			StackDepth:    step.StackDepth,
			Type:          step.Type,
			Status:        step.Status,
			Input:         step.Input,
			Output:        step.Output,
			SpaceSnapshot: step.SpaceSnapshot,
			Error:         step.Error,
			Sequence:      step.Sequence,
			Metadata:      step.Metadata,
			CreatedAt:     step.CreatedAt,
			UpdatedAt:     step.CreatedAt,
		}
	}
	return steps
}

// EnsureChat ensures a chat session exists, creates if not
func (ast *Assistant) EnsureChat(ctx *agentcontext.Context) error {
	if ctx.ChatID == "" {
		return nil // No chat ID, skip
	}

	// Skip if history is disabled
	if ctx.Stack != nil && ctx.Stack.Options != nil && ctx.Stack.Options.Skip != nil && ctx.Stack.Options.Skip.History {
		return nil // Skip.History is true, don't create chat session
	}

	chatStore := GetChatStore()
	if chatStore == nil {
		return nil // No store, skip
	}

	// Check if chat exists
	_, err := chatStore.GetChat(ctx.ChatID)
	if err == nil {
		return nil // Chat exists
	}

	// Create new chat with permission fields
	chat := &storetypes.Chat{
		ChatID:      ctx.ChatID,
		AssistantID: ast.ID,
		Status:      "active",
		Share:       "private",
		Sort:        0,
		Metadata:    ctx.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set last_connector from options (user selected connector)
	if ctx.Stack != nil && ctx.Stack.Options != nil && ctx.Stack.Options.Connector != "" {
		chat.LastConnector = ctx.Stack.Options.Connector
	}

	// Set last_workspace from metadata
	if ctx.Metadata != nil {
		if ws, ok := ctx.Metadata["workspace_id"].(string); ok && ws != "" {
			chat.LastWorkspace = ws
		}
	}

	// Set permission fields from authorized info
	if ctx.Authorized != nil {
		chat.CreatedBy = ctx.Authorized.UserID
		chat.UpdatedBy = ctx.Authorized.UserID
		chat.TeamID = ctx.Authorized.TeamID
		chat.TenantID = ctx.Authorized.TenantID
	}

	return chatStore.CreateChat(chat)
}

// updateChatWorkspace persists last_workspace early so API handlers can
// resolve the host node during the first stream (before FlushBuffer runs).
func (ast *Assistant) updateChatWorkspace(chatID, workspaceID string) {
	if chatID == "" || workspaceID == "" {
		return
	}
	chatStore := GetChatStore()
	if chatStore == nil {
		return
	}
	_ = chatStore.UpdateChat(chatID, map[string]interface{}{
		"last_workspace": workspaceID,
	})
}

// GetChatStore returns the chat store instance
// Returns nil if storage is not configured
func GetChatStore() storetypes.ChatStore {
	if storage == nil {
		return nil
	}
	return storage
}

// GetStore returns the full store instance (implements both ChatStore and AssistantStore)
// Returns nil if storage is not configured
func GetStore() storetypes.Store {
	if storage == nil {
		return nil
	}
	return storage
}

// =============================================================================
// Deprecated methods (kept for compatibility)
// =============================================================================

func (ast *Assistant) saveChat(ctx *agentcontext.Context, input []agentcontext.Message, opts *agentcontext.Options) error {
	_ = ctx
	_ = input
	_ = opts
	return nil
}
