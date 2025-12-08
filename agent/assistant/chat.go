package assistant

import (
	"fmt"
	"strings"
	"sync"

	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/kb"
	kbapi "github.com/yaoapp/yao/kb/api"
	"github.com/yaoapp/yao/trace/types"
)

// kbCollectionCreating tracks collections currently being created to avoid duplicate creation
var kbCollectionCreating sync.Map

// WithHistory merges the input messages with chat history and traces it
// This method can be overridden or extended to implement actual history loading
func (ast *Assistant) WithHistory(ctx *agentcontext.Context, input []agentcontext.Message, agentNode types.Node, options ...*agentcontext.Options) ([]agentcontext.Message, error) {

	// TODO: Implement actual history loading logic here
	// For now, just simulate a check and return the input messages as is

	// Simulate error check (this is where actual history loading would happen)
	// if some_condition {
	//     ast.traceAgentFail(agentNode, err)
	//     return nil, err
	// }

	fullMessages := input

	// Log the chat history
	ast.traceAgentHistory(ctx, agentNode, fullMessages)

	return fullMessages, nil
}

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
		fmt.Printf(">>> Warning: no authorized info, skipping KB collection preparation\n")
		return nil
	}

	// Prepare kb collection
	err := ast.prepareKBCollection(ctx, opts)
	if err != nil {
		// Log but don't fail the chat
		fmt.Printf(">>> Warning: failed to prepare KB collection: %v\n", err)
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
	fmt.Printf(">>> prepareKBCollection: locale=%s\n", ctx.Locale)

	// Get KB collection ID for this chat session
	// Same team + user always produces the same ID (idempotent)
	collectionID := GetChatKBID(ctx.Authorized.TeamID, ctx.Authorized.UserID)

	// Check if this collection is currently being created by another goroutine
	if _, isCreating := kbCollectionCreating.LoadOrStore(collectionID, true); isCreating {
		fmt.Printf(">>> KB collection %s is already being created, skipping\n", collectionID)
		return nil
	}
	// Ensure cleanup even if panic occurs
	defer kbCollectionCreating.Delete(collectionID)

	// Check if collection already exists
	existsResult, err := kb.API.CollectionExists(ctx.Context, collectionID)
	if err != nil {
		// If check fails, log and continue to create (let create handle conflicts)
		fmt.Printf(">>> Warning: failed to check collection existence: %v, will attempt to create\n", err)
	} else if existsResult != nil && existsResult.Exists {
		// Collection exists, no need to create
		fmt.Printf(">>> KB collection already exists: %s\n", collectionID)
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

	fmt.Printf(">>> Created KB collection: %s for team=%s, user=%s\n",
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

	fmt.Printf(">>> mergeChatMetadata: locale=%s, name=%v, description=%v\n", locale, metadata["name"], metadata["description"]) // Debug log

	return metadata
}

func (ast *Assistant) saveChat(ctx *agentcontext.Context, input []agentcontext.Message, opts *agentcontext.Options) error {
	_ = ctx
	_ = input
	_ = opts
	return nil
}
