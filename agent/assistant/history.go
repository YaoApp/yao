package assistant

import (
	"fmt"
	"reflect"

	agentcontext "github.com/yaoapp/yao/agent/context"
	storetypes "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/trace/types"
)

// =============================================================================
// Chat History Management
// =============================================================================

// HistoryResult represents the result of history processing
type HistoryResult struct {
	InputMessages []agentcontext.Message // Clean input messages (without overlap)
	FullMessages  []agentcontext.Message // Full messages (history + clean input)
}

// WithHistory merges the input messages with chat history and traces it
// Returns HistoryResult containing:
// - InputMessages: cleaned input (overlap removed)
// - FullMessages: history + clean input merged
func (ast *Assistant) WithHistory(ctx *agentcontext.Context, input []agentcontext.Message, agentNode types.Node, options ...*agentcontext.Options) (*HistoryResult, error) {

	// Get options
	var opts *agentcontext.Options
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}

	// SKIP: History (for internal calls like title/prompt etc.)
	if opts != nil && opts.Skip != nil && opts.Skip.History {
		result := &HistoryResult{
			InputMessages: input,
			FullMessages:  input,
		}
		ast.traceAgentHistory(ctx, agentNode, result.FullMessages)
		return result, nil
	}

	// Get MaxSize from store setting
	maxSize := 20 // default
	if storeSetting := GetStoreSetting(); storeSetting != nil && storeSetting.MaxSize > 0 {
		maxSize = storeSetting.MaxSize
	}

	// Load history from store
	historyMessages, err := ast.loadHistory(ctx)
	if err != nil {
		// Log warning but continue without history
		ctx.Logger.Warn("Failed to load history for chat=%s: %v", ctx.ChatID, err)
		result := &HistoryResult{
			InputMessages: input,
			FullMessages:  input,
		}
		ast.traceAgentHistory(ctx, agentNode, result.FullMessages)
		return result, nil
	}

	// If no history, return input as is
	if len(historyMessages) == 0 {
		ctx.Logger.HistoryLoad(0, maxSize)
		result := &HistoryResult{
			InputMessages: input,
			FullMessages:  input,
		}
		ast.traceAgentHistory(ctx, agentNode, result.FullMessages)
		return result, nil
	}

	// Log history loaded
	ctx.Logger.HistoryLoad(len(historyMessages), maxSize)

	// Find overlap between history and input
	// Some external clients may include history in their requests
	overlapIndex := ast.findOverlapIndex(historyMessages, input)

	// Remove overlap from input
	cleanInput := input
	if overlapIndex > 0 {
		cleanInput = input[overlapIndex:]
		ctx.Logger.HistoryOverlap(overlapIndex)
	}

	// Merge history with clean input
	fullMessages := make([]agentcontext.Message, 0, len(historyMessages)+len(cleanInput))
	fullMessages = append(fullMessages, historyMessages...)
	fullMessages = append(fullMessages, cleanInput...)

	result := &HistoryResult{
		InputMessages: cleanInput,
		FullMessages:  fullMessages,
	}

	// Log the chat history
	ast.traceAgentHistory(ctx, agentNode, result.FullMessages)

	return result, nil
}

// loadHistory loads chat history from the store
// Returns the most recent MaxSize messages, ordered by time (oldest first)
func (ast *Assistant) loadHistory(ctx *agentcontext.Context) ([]agentcontext.Message, error) {
	// Check if chat ID is available
	if ctx.ChatID == "" {
		return nil, nil
	}

	// Get chat store
	chatStore := GetChatStore()
	if chatStore == nil {
		return nil, nil
	}

	// Get store setting for MaxSize
	setting := GetStoreSetting()
	maxSize := 20 // default
	if setting != nil && setting.MaxSize > 0 {
		maxSize = setting.MaxSize
	}

	// Load messages from store with limit
	filter := storetypes.MessageFilter{
		Limit: maxSize,
	}

	storeMessages, err := chatStore.GetMessages(ctx.ChatID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	if len(storeMessages) == 0 {
		return nil, nil
	}

	// Convert store messages to context messages
	messages := make([]agentcontext.Message, 0, len(storeMessages))
	for _, msg := range storeMessages {
		// Only include user and assistant messages for LLM context
		// Skip internal types like loading, event, etc.
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}

		// Convert store message to context message
		ctxMsg := ast.convertStoreMessageToContext(msg)
		if ctxMsg != nil {
			messages = append(messages, *ctxMsg)
		}
	}

	return messages, nil
}

// convertStoreMessageToContext converts a store message to a context message
func (ast *Assistant) convertStoreMessageToContext(msg *storetypes.Message) *agentcontext.Message {
	if msg == nil {
		return nil
	}

	// Skip internal message types that should not be included in LLM context
	// These types are for UI/internal use only and can confuse the LLM
	// Note: "error" is kept so LLM can help troubleshoot issues
	switch msg.Type {
	case "tool_call", "loading", "action", "event":
		return nil
	}

	// Extract content from Props
	content := ast.extractContentFromProps(msg.Props, msg.Type)
	if content == nil {
		return nil
	}

	// Build context message
	ctxMsg := &agentcontext.Message{
		Role:    agentcontext.MessageRole(msg.Role),
		Content: content,
	}

	// Handle name field
	if msg.Props != nil {
		if name, ok := msg.Props["name"].(string); ok && name != "" {
			ctxMsg.Name = &name
		}
	}

	return ctxMsg
}

// extractContentFromProps extracts the content from message Props based on message type
func (ast *Assistant) extractContentFromProps(props map[string]interface{}, msgType string) interface{} {
	if props == nil {
		return nil
	}

	// For user input, content is stored directly in props["content"]
	if msgType == "user_input" {
		return props["content"]
	}

	// For text type messages
	if msgType == "text" {
		if text, ok := props["text"].(string); ok {
			return text
		}
		// Also try content field
		if content, ok := props["content"].(string); ok {
			return content
		}
	}

	// For other types, try to extract content or text
	if content, ok := props["content"]; ok {
		return content
	}
	if text, ok := props["text"]; ok {
		return text
	}

	return nil
}

// findOverlapIndex finds the index in input where history messages end
// Returns the number of input messages that overlap with history
func (ast *Assistant) findOverlapIndex(history, input []agentcontext.Message) int {
	if len(history) == 0 || len(input) == 0 {
		return 0
	}

	// We need to find the longest suffix of history that matches a prefix of input
	// Start from the end of history and try to match with the beginning of input

	maxOverlap := len(history)
	if maxOverlap > len(input) {
		maxOverlap = len(input)
	}

	// Try different overlap lengths, starting from the largest possible
	for overlapLen := maxOverlap; overlapLen > 0; overlapLen-- {
		// Check if the last 'overlapLen' messages of history match the first 'overlapLen' of input
		historyStart := len(history) - overlapLen
		matched := true

		for i := 0; i < overlapLen; i++ {
			if !ast.messagesMatch(history[historyStart+i], input[i]) {
				matched = false
				break
			}
		}

		if matched {
			return overlapLen
		}
	}

	return 0
}

// messagesMatch checks if two messages are equivalent
func (ast *Assistant) messagesMatch(a, b agentcontext.Message) bool {
	// Must have same role
	if a.Role != b.Role {
		return false
	}

	// Compare content
	return ast.contentMatches(a.Content, b.Content)
}

// contentMatches compares two content values for equality
func (ast *Assistant) contentMatches(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// If both are strings, compare directly
	aStr, aIsStr := a.(string)
	bStr, bIsStr := b.(string)
	if aIsStr && bIsStr {
		return aStr == bStr
	}

	// For complex content (arrays, etc.), use deep equal
	return reflect.DeepEqual(a, b)
}
