package assistant

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/kb"
	kbapi "github.com/yaoapp/yao/kb/api"
)

// kbCollectionCreating tracks collections currently being created to avoid duplicate creation
var kbCollectionCreating sync.Map

// InitializeConversation prepares KB collection for the conversation (synchronous)
func (ast *Assistant) InitializeConversation(ctx *agentcontext.Context, options ...*agentcontext.Options) error {

	var opts *agentcontext.Options
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	} else {
		opts = &agentcontext.Options{}
	}

	// SKIP: History (for internal calls like title/prompt etc.)
	if opts.Skip != nil && opts.Skip.History {
		return nil
	}

	// Check if authorized info is available
	if ctx.Authorized == nil {
		ctx.Logger.Warn("no authorized info, skipping KB collection preparation")
		return nil
	}

	// Prepare kb collection
	err := ast.prepareKBCollection(ctx, opts)
	if err != nil {
		// Log but don't fail the chat
		ctx.Logger.Warn("failed to prepare KB collection: %v", err)
	}

	return nil
}

// InitializeConversationAsync prepares KB collection asynchronously
func (ast *Assistant) InitializeConversationAsync(ctx *agentcontext.Context, options ...*agentcontext.Options) {
	go ast.InitializeConversation(ctx, options...)
}

// prepareKBCollection prepares kb collection (internal method)
func (ast *Assistant) prepareKBCollection(ctx *agentcontext.Context, opts *agentcontext.Options) error {

	// Get global KB setting
	kbSetting := GetGlobalKBSetting()
	if kbSetting == nil || kbSetting.Chat == nil {
		return nil // No KB configuration for chat, skip
	}

	// Check if KB API is initialized
	if kb.API == nil {
		return fmt.Errorf("KB API not initialized")
	}

	// Check if authorized info is available
	if ctx.Authorized == nil {
		return fmt.Errorf("authorized information not available")
	}

	chatKB := kbSetting.Chat

	// Debug: log locale information
	ctx.Logger.Debug("prepareKBCollection: locale=%s", ctx.Locale)

	// Get KB collection ID for this chat session
	// Same team + user always produces the same ID (idempotent)
	collectionID := GetChatKBID(ctx.Authorized.TeamID, ctx.Authorized.UserID)

	// Check if this collection is currently being created by another goroutine
	if _, isCreating := kbCollectionCreating.LoadOrStore(collectionID, true); isCreating {
		ctx.Logger.Debug("KB collection %s is already being created, skipping", collectionID)
		return nil
	}
	// Ensure cleanup even if panic occurs
	defer kbCollectionCreating.Delete(collectionID)

	// Check if collection already exists
	existsResult, err := kb.API.CollectionExists(ctx.Context, collectionID)
	if err != nil {
		// If check fails, log and continue to create (let create handle conflicts)
		ctx.Logger.Warn("failed to check collection existence: %v, will attempt to create", err)
	} else if existsResult != nil && existsResult.Exists {
		// Collection exists, no need to create
		ctx.Logger.Debug("KB collection already exists: %s", collectionID)
		return nil
	}

	// Create new collection for this chat session
	createParams := &kbapi.CreateCollectionParams{
		ID:                  collectionID,
		EmbeddingProviderID: chatKB.EmbeddingProviderID,
		EmbeddingOptionID:   chatKB.EmbeddingOptionID,
		Locale:              chatKB.Locale,
		Config:              chatKB.Config,
		Metadata:            mergeChatMetadata(chatKB.Metadata, ctx),
		AuthScope:           ctx.Authorized.WithCreateScope(make(map[string]interface{})),
	}

	_, err = kb.API.CreateCollection(ctx.Context, createParams)
	if err != nil {
		return fmt.Errorf("failed to create KB collection: %w", err)
	}

	ctx.Logger.Info("Created KB collection: %s for team=%s, user=%s",
		collectionID, ctx.Authorized.TeamID, ctx.Authorized.UserID)

	_ = opts
	return nil
}

// GetChatKBID returns the KB collection ID for a chat session
// Same team + user always returns the same ID (deterministic)
// Format: chat_{team}_{user} or chat_user_{user} if no team
func GetChatKBID(teamID, userID string) string {
	// Sanitize IDs: replace invalid chars with underscores
	cleanTeamID := sanitizeCollectionID(teamID)
	cleanUserID := sanitizeCollectionID(userID)

	if cleanTeamID != "" {
		return fmt.Sprintf("chat_%s_%s", cleanTeamID, cleanUserID)
	}
	return fmt.Sprintf("chat_user_%s", cleanUserID)
}

// sanitizeCollectionID replaces invalid characters with underscores
// Collection IDs only allow: a-z, A-Z, 0-9, and underscore
func sanitizeCollectionID(id string) string {
	if id == "" {
		return ""
	}

	// Replace any character that is not alphanumeric or underscore with underscore
	result := make([]byte, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result[i] = c
		} else {
			result[i] = '_'
		}
	}
	return string(result)
}

// mergeChatMetadata merges default metadata with chat context information
func mergeChatMetadata(defaultMetadata map[string]interface{}, ctx *agentcontext.Context) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Copy default metadata
	for k, v := range defaultMetadata {
		metadata[k] = v
	}

	// Add chat-specific metadata (only for internal tracking, not displayed)
	metadata["chat_id"] = ctx.ChatID
	metadata["team_id"] = ctx.Authorized.TeamID
	metadata["user_id"] = ctx.Authorized.UserID

	// Get locale from context, default to zh-CN if not set
	locale := ctx.Locale
	if locale == "" {
		locale = "zh-CN"
	}
	locale = strings.ToLower(locale)

	// Use i18n for name and description (fixed, not showing user/team IDs)
	if _, exists := metadata["name"]; !exists {
		metadata["name"] = i18n.T(locale, "kb.chat.name")
	}
	if _, exists := metadata["description"]; !exists {
		metadata["description"] = i18n.T(locale, "kb.chat.description")
	}

	return metadata
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
}

// convertBufferedMessages converts BufferedMessage slice to store Message slice
func (ast *Assistant) convertBufferedMessages(buffered []*agentcontext.BufferedMessage) []*storetypes.Message {
	if len(buffered) == 0 {
		return nil
	}

	messages := make([]*storetypes.Message, len(buffered))
	for i, msg := range buffered {
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
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set last_connector from options (user selected connector)
	if ctx.Stack != nil && ctx.Stack.Options != nil && ctx.Stack.Options.Connector != "" {
		chat.LastConnector = ctx.Stack.Options.Connector
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
