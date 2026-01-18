package standard

import (
	"fmt"

	"github.com/yaoapp/gou/text"
	"github.com/yaoapp/yao/agent/assistant"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// AgentCaller provides unified interface for calling AI assistants
// It wraps the Yao Assistant framework and handles:
// - Getting assistant by ID
// - Single call with messages (streaming)
// - Multi-turn conversation with session state
// - Parsing responses (text, JSON, Next hook data)
type AgentCaller struct {
	// SkipOutput skips sending output to client (for internal calls)
	SkipOutput bool

	// SkipHistory skips saving to chat history (default: true for robot)
	// Set to false to enable multi-turn conversation with history
	SkipHistory bool

	// SkipSearch skips auto search
	SkipSearch bool

	// ChatID is used for multi-turn conversations to maintain session state
	// If empty, each call is independent (no history)
	ChatID string
}

// NewAgentCaller creates a new AgentCaller with default settings (single-call mode)
func NewAgentCaller() *AgentCaller {
	return &AgentCaller{
		SkipOutput:  true, // Robot executions don't send to UI
		SkipHistory: true, // Robot executions don't save to chat history
		SkipSearch:  true, // Robot executions don't trigger auto search
	}
}

// NewConversationCaller creates an AgentCaller for multi-turn conversations
// chatID is used to maintain session state across calls
// This is useful for:
// - P2 (Tasks): Iterative task refinement with user feedback
// - P3 (Run): Multi-step task execution with intermediate results
func NewConversationCaller(chatID string) *AgentCaller {
	return &AgentCaller{
		SkipOutput:  true,
		SkipHistory: false, // Enable history for multi-turn
		SkipSearch:  true,
		ChatID:      chatID,
	}
}

// CallResult holds the result of an agent call
type CallResult struct {
	// Content is the raw text content from LLM completion
	Content string

	// Next is the data returned from Next hook (if any)
	// This is typically a structured response from the assistant
	Next interface{}

	// Response is the full response object (for advanced use)
	Response *agentcontext.Response
}

// IsEmpty returns true if the result has no content
func (r *CallResult) IsEmpty() bool {
	return r.Content == "" && r.Next == nil
}

// GetText returns the text content, preferring Content over Next
func (r *CallResult) GetText() string {
	if r.Content != "" {
		return r.Content
	}
	// If Next is a string, return it
	if s, ok := r.Next.(string); ok {
		return s
	}
	// If Next has a "content" field, return it
	if m, ok := r.Next.(map[string]interface{}); ok {
		if content, ok := m["content"].(string); ok {
			return content
		}
		// Also check "data" field (common pattern in Next hook)
		if data, ok := m["data"].(map[string]interface{}); ok {
			if content, ok := data["content"].(string); ok {
				return content
			}
		}
	}
	return ""
}

// GetJSON attempts to parse the result as JSON
// It tries in order:
// 1. Next hook data (already structured)
// 2. Content parsed using gou/text.ExtractJSON (fault-tolerant)
// Returns the parsed data and any error
func (r *CallResult) GetJSON() (map[string]interface{}, error) {
	// Try Next hook data first
	if r.Next != nil {
		if m, ok := r.Next.(map[string]interface{}); ok {
			// Check for "data" wrapper (common in Next hook)
			if data, ok := m["data"].(map[string]interface{}); ok {
				return data, nil
			}
			return m, nil
		}
	}

	// Try parsing Content using gou/text (handles markdown blocks, JSON, YAML)
	if r.Content != "" {
		data := text.ExtractJSON(r.Content)
		if data != nil {
			if m, ok := data.(map[string]interface{}); ok {
				return m, nil
			}
		}
		return nil, fmt.Errorf("content is not a JSON object")
	}

	return nil, fmt.Errorf("no content to parse")
}

// GetJSONArray attempts to parse the result as JSON array
// Similar to GetJSON but for array responses
func (r *CallResult) GetJSONArray() ([]interface{}, error) {
	// Try Next hook data first
	if r.Next != nil {
		if arr, ok := r.Next.([]interface{}); ok {
			return arr, nil
		}
		if m, ok := r.Next.(map[string]interface{}); ok {
			// Check for "data" wrapper
			if data, ok := m["data"].([]interface{}); ok {
				return data, nil
			}
		}
	}

	// Try parsing Content using gou/text (handles markdown blocks, JSON, YAML)
	if r.Content != "" {
		data := text.ExtractJSON(r.Content)
		if data != nil {
			if arr, ok := data.([]interface{}); ok {
				return arr, nil
			}
		}
		return nil, fmt.Errorf("content is not a JSON array")
	}

	return nil, fmt.Errorf("no content to parse")
}

// Call calls an assistant with messages and returns the result
// This is the main entry point for agent calls
func (c *AgentCaller) Call(ctx *robottypes.Context, assistantID string, messages []agentcontext.Message) (*CallResult, error) {
	// Get assistant
	ast, err := assistant.Get(assistantID)
	if err != nil {
		return nil, fmt.Errorf("assistant not found: %s: %w", assistantID, err)
	}

	// Build options
	opts := &agentcontext.Options{
		Skip: &agentcontext.Skip{
			Output:  c.SkipOutput,
			History: c.SkipHistory,
			Search:  c.SkipSearch,
		},
	}

	// Convert robot context to agent context
	agentCtx := c.buildAgentContext(ctx)
	defer agentCtx.Release() // IMPORTANT: Release agent context to prevent resource leaks

	// Call assistant with streaming
	response, err := ast.Stream(agentCtx, messages, opts)
	if err != nil {
		return nil, fmt.Errorf("assistant call failed: %w", err)
	}

	// Build result
	result := &CallResult{
		Response: response,
	}

	// Extract Next hook data
	if response.Next != nil {
		result.Next = response.Next
	}

	// Extract Content from Completion
	if response.Completion != nil {
		if content, ok := response.Completion.Content.(string); ok {
			result.Content = content
		}
	}

	return result, nil
}

// CallWithMessages is a convenience method that builds messages from a single user input
func (c *AgentCaller) CallWithMessages(ctx *robottypes.Context, assistantID string, userContent string) (*CallResult, error) {
	messages := []agentcontext.Message{
		{
			Role:    agentcontext.RoleUser,
			Content: userContent,
		},
	}
	return c.Call(ctx, assistantID, messages)
}

// CallWithSystemAndUser calls with both system and user messages
func (c *AgentCaller) CallWithSystemAndUser(ctx *robottypes.Context, assistantID string, systemContent, userContent string) (*CallResult, error) {
	messages := []agentcontext.Message{
		{
			Role:    agentcontext.RoleSystem,
			Content: systemContent,
		},
		{
			Role:    agentcontext.RoleUser,
			Content: userContent,
		},
	}
	return c.Call(ctx, assistantID, messages)
}

// buildAgentContext converts robot context to agent context
func (c *AgentCaller) buildAgentContext(ctx *robottypes.Context) *agentcontext.Context {
	// Build authorized info for agent context
	var authorized *oauthtypes.AuthorizedInfo
	if ctx.Auth != nil {
		authorized = &oauthtypes.AuthorizedInfo{
			UserID: ctx.Auth.UserID,
			TeamID: ctx.Auth.TeamID,
		}
	}

	// Create a new agent context
	// Use ChatID for multi-turn conversations, empty for single calls
	agentCtx := agentcontext.New(ctx.Context, authorized, c.ChatID)

	// Set locale if available
	if ctx.Locale != "" {
		agentCtx.Locale = ctx.Locale
	}

	// Use noop logger to suppress LLM debug output for robot executions
	// Robot executions run in background and don't need console output
	if agentCtx.Logger != nil {
		agentCtx.Logger.Close()
	}
	agentCtx.Logger = agentcontext.Noop()

	return agentCtx
}

// ExtractCodeBlock extracts the first code block from content using gou/text
// Returns the CodeBlock with type, content, and parsed data (for JSON/YAML)
func ExtractCodeBlock(content string) *text.CodeBlock {
	return text.ExtractFirst(content)
}

// ExtractAllCodeBlocks extracts all code blocks from content using gou/text
func ExtractAllCodeBlocks(content string) []text.CodeBlock {
	return text.Extract(content)
}

// ============================================================================
// Conversation - Multi-turn dialogue support
// ============================================================================

// Conversation manages a multi-turn dialogue with an assistant
// Useful for:
// - P2 (Tasks): Iterative task planning with clarification
// - P3 (Run): Multi-step execution with intermediate validation
// - Complex reasoning that requires back-and-forth
type Conversation struct {
	caller      *AgentCaller
	assistantID string
	messages    []agentcontext.Message
	maxTurns    int
}

// TurnResult holds the result of a single conversation turn
type TurnResult struct {
	Turn     int                    // Turn number (1-based)
	Input    string                 // User input for this turn
	Result   *CallResult            // Agent response
	Messages []agentcontext.Message // Full message history after this turn
}

// NewConversation creates a new multi-turn conversation
// assistantID: the assistant to converse with
// chatID: session ID for maintaining state (use exec.ID for robot executions)
// maxTurns: maximum number of turns (0 = unlimited)
func NewConversation(assistantID, chatID string, maxTurns int) *Conversation {
	return &Conversation{
		caller:      NewConversationCaller(chatID),
		assistantID: assistantID,
		messages:    make([]agentcontext.Message, 0),
		maxTurns:    maxTurns,
	}
}

// WithCaller sets a custom AgentCaller for the conversation
// Useful for customizing SkipSearch or other options
func (c *Conversation) WithCaller(caller *AgentCaller) *Conversation {
	c.caller = caller
	return c
}

// WithSystemPrompt adds a system prompt at the beginning of the conversation
func (c *Conversation) WithSystemPrompt(systemPrompt string) *Conversation {
	if systemPrompt != "" {
		c.messages = append([]agentcontext.Message{{
			Role:    agentcontext.RoleSystem,
			Content: systemPrompt,
		}}, c.messages...)
	}
	return c
}

// WithHistory initializes the conversation with existing message history
// Note: Message structs are copied, but Content (interface{}) is a shallow copy
func (c *Conversation) WithHistory(messages []agentcontext.Message) *Conversation {
	c.messages = append(c.messages, messages...)
	return c
}

// Turn executes a single turn in the conversation
// userInput: the user's message for this turn
// Returns the turn result with agent response
func (c *Conversation) Turn(ctx *robottypes.Context, userInput string) (*TurnResult, error) {
	// Check max turns
	turnNum := c.TurnCount() + 1
	if c.maxTurns > 0 && turnNum > c.maxTurns {
		return nil, fmt.Errorf("max turns (%d) exceeded", c.maxTurns)
	}

	// Build messages with user input (don't modify history yet)
	userMsg := agentcontext.Message{
		Role:    agentcontext.RoleUser,
		Content: userInput,
	}
	// Create a new slice to avoid modifying c.messages if capacity allows append in-place
	messagesWithInput := make([]agentcontext.Message, len(c.messages)+1)
	copy(messagesWithInput, c.messages)
	messagesWithInput[len(c.messages)] = userMsg

	// Call assistant with full history
	result, err := c.caller.Call(ctx, c.assistantID, messagesWithInput)
	if err != nil {
		return nil, fmt.Errorf("turn %d failed: %w", turnNum, err)
	}

	// Only update history after successful call
	c.messages = append(c.messages, userMsg)

	// Add assistant response to history
	if result.Content != "" {
		c.messages = append(c.messages, agentcontext.Message{
			Role:    agentcontext.RoleAssistant,
			Content: result.Content,
		})
	}

	// Return a copy of messages to prevent external modification
	messagesCopy := make([]agentcontext.Message, len(c.messages))
	copy(messagesCopy, c.messages)

	return &TurnResult{
		Turn:     turnNum,
		Input:    userInput,
		Result:   result,
		Messages: messagesCopy,
	}, nil
}

// TurnCount returns the number of user turns so far
func (c *Conversation) TurnCount() int {
	count := 0
	for _, msg := range c.messages {
		if msg.Role == agentcontext.RoleUser {
			count++
		}
	}
	return count
}

// Messages returns a copy of the current message history
func (c *Conversation) Messages() []agentcontext.Message {
	messagesCopy := make([]agentcontext.Message, len(c.messages))
	copy(messagesCopy, c.messages)
	return messagesCopy
}

// LastResponse returns a copy of the last assistant response, or nil if none
func (c *Conversation) LastResponse() *agentcontext.Message {
	for i := len(c.messages) - 1; i >= 0; i-- {
		if c.messages[i].Role == agentcontext.RoleAssistant {
			// Return a copy to prevent external modification
			msg := c.messages[i]
			return &msg
		}
	}
	return nil
}

// Reset clears the conversation history (keeps system prompt if any)
func (c *Conversation) Reset() {
	// Keep system prompt if present
	var systemPrompt *agentcontext.Message
	if len(c.messages) > 0 && c.messages[0].Role == agentcontext.RoleSystem {
		systemPrompt = &c.messages[0]
	}

	c.messages = make([]agentcontext.Message, 0)
	if systemPrompt != nil {
		c.messages = append(c.messages, *systemPrompt)
	}
}

// RunUntil runs the conversation until a condition is met
// checkFn: called after each turn, returns (done, error)
// Returns all turn results
func (c *Conversation) RunUntil(
	ctx *robottypes.Context,
	inputFn func(turn int, lastResult *CallResult) (string, error),
	checkFn func(turn int, result *CallResult) (done bool, err error),
) ([]*TurnResult, error) {
	var results []*TurnResult

	for {
		turnNum := c.TurnCount() + 1

		// Check max turns
		if c.maxTurns > 0 && turnNum > c.maxTurns {
			return results, fmt.Errorf("max turns (%d) exceeded without completion", c.maxTurns)
		}

		// Get input for this turn
		var lastResult *CallResult
		if len(results) > 0 {
			lastResult = results[len(results)-1].Result
		}

		input, err := inputFn(turnNum, lastResult)
		if err != nil {
			return results, fmt.Errorf("input generation failed at turn %d: %w", turnNum, err)
		}

		// Execute turn
		turnResult, err := c.Turn(ctx, input)
		if err != nil {
			return results, err
		}
		results = append(results, turnResult)

		// Check completion condition
		done, err := checkFn(turnNum, turnResult.Result)
		if err != nil {
			return results, fmt.Errorf("check failed at turn %d: %w", turnNum, err)
		}
		if done {
			return results, nil
		}
	}
}
