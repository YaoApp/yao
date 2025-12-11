package hook_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newRealWorldNextContext creates a Context for real world Next Hook testing
func newRealWorldNextContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:   "realworld-test-user",
		ClientID:  "realworld-test-client",
		Scope:     "openid profile",
		SessionID: "realworld-test-session",
		UserID:    "realworld-user-123",
		TeamID:    "realworld-team-456",
		TenantID:  "realworld-tenant-789",
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "RealWorldTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// TestRealWorldNextStandard tests standard response (nil return)
func TestRealWorldNextStandard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world Next Hook test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldNextContext("test-next-standard", "tests.realworld-next")

	// Simulate completion with scenario marker
	messages := []context.Message{
		{Role: context.RoleUser, Content: "scenario: standard"},
		{Role: context.RoleAssistant, Content: "I'll process your request using standard response."},
	}

	completion := &context.CompletionResponse{
		Content: "Processing complete. Standard response will be used.",
	}

	payload := &context.NextHookPayload{
		Messages:   messages,
		Completion: completion,
		Tools:      nil,
		Error:      "",
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	if err != nil {
		t.Fatalf("Next hook failed: %v", err)
	}

	// Should return nil for standard response
	assert.Nil(t, response, "Standard scenario should return nil")
	t.Log("✓ Standard response scenario passed")
}

// TestRealWorldNextCustomData tests custom data response
func TestRealWorldNextCustomData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world Next Hook test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldNextContext("test-next-custom", "tests.realworld-next")

	messages := []context.Message{
		{Role: context.RoleUser, Content: "scenario: custom_data"},
		{Role: context.RoleAssistant, Content: "Here's some information for you."},
	}

	completion := &context.CompletionResponse{
		Content: "This is the LLM completion that will be summarized.",
	}

	payload := &context.NextHookPayload{
		Messages:   messages,
		Completion: completion,
		Tools:      nil,
		Error:      "",
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	if err != nil {
		t.Fatalf("Next hook failed: %v", err)
	}

	assert.NotNil(t, response, "Custom data scenario should return response")
	assert.NotNil(t, response.Data, "Response should have Data")

	dataMap, ok := response.Data.(map[string]interface{})
	assert.True(t, ok, "Data should be a map")
	assert.Equal(t, "custom_response", dataMap["type"])
	assert.Contains(t, dataMap, "timestamp")

	t.Log("✓ Custom data response scenario passed")
}

// TestRealWorldNextDelegate tests agent delegation
func TestRealWorldNextDelegate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world Next Hook test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldNextContext("test-next-delegate", "tests.realworld-next")

	messages := []context.Message{
		{Role: context.RoleUser, Content: "scenario: delegate"},
	}

	completion := &context.CompletionResponse{
		Content: "I should delegate this request to another agent.",
	}

	payload := &context.NextHookPayload{
		Messages:   messages,
		Completion: completion,
		Tools:      nil,
		Error:      "",
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	if err != nil {
		t.Fatalf("Next hook failed: %v", err)
	}

	assert.NotNil(t, response, "Delegate scenario should return response")
	assert.NotNil(t, response.Delegate, "Response should have Delegate")
	assert.Equal(t, "tests.create", response.Delegate.AgentID)
	assert.NotEmpty(t, response.Delegate.Messages)

	t.Log("✓ Delegation scenario passed")
}

// TestRealWorldNextProcessTools tests tool result processing
func TestRealWorldNextProcessTools(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world Next Hook test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldNextContext("test-next-tools", "tests.realworld-next")

	messages := []context.Message{
		{Role: context.RoleUser, Content: "scenario: process_tools"},
	}

	completion := &context.CompletionResponse{
		Content: "Tool calls have been executed.",
	}

	// Simulate tool call results
	tools := []context.ToolCallResponse{
		{
			ToolCallID: "call_1",
			Server:     "test-server",
			Tool:       "test-tool-1",
			Result:     map[string]interface{}{"status": "success"},
			Error:      "",
		},
		{
			ToolCallID: "call_2",
			Server:     "test-server",
			Tool:       "test-tool-2",
			Result:     nil,
			Error:      "Tool execution failed",
		},
	}

	payload := &context.NextHookPayload{
		Messages:   messages,
		Completion: completion,
		Tools:      tools,
		Error:      "",
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	if err != nil {
		t.Fatalf("Next hook failed: %v", err)
	}

	assert.NotNil(t, response, "Process tools scenario should return response")
	assert.NotNil(t, response.Data, "Response should have Data")

	dataMap, ok := response.Data.(map[string]interface{})
	assert.True(t, ok, "Data should be a map")
	assert.Equal(t, "Tool execution summary", dataMap["message"])

	// Check summary
	summary, ok := dataMap["summary"].(map[string]interface{})
	assert.True(t, ok, "Should have summary")
	assert.Equal(t, float64(2), summary["total"])
	assert.Equal(t, float64(1), summary["successful"])
	assert.Equal(t, float64(1), summary["failed"])

	t.Log("✓ Process tools scenario passed")
}

// TestRealWorldNextErrorRecovery tests error handling and recovery
func TestRealWorldNextErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world Next Hook test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldNextContext("test-next-error", "tests.realworld-next")

	messages := []context.Message{
		{Role: context.RoleUser, Content: "scenario: error_recovery"},
	}

	completion := &context.CompletionResponse{
		Content: "An error occurred during processing.",
	}

	payload := &context.NextHookPayload{
		Messages:   messages,
		Completion: completion,
		Tools:      nil,
		Error:      "System error: Database connection timeout",
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	if err != nil {
		t.Fatalf("Next hook failed: %v", err)
	}

	assert.NotNil(t, response, "Error recovery scenario should return response")
	assert.NotNil(t, response.Data, "Response should have Data")

	dataMap, ok := response.Data.(map[string]interface{})
	assert.True(t, ok, "Data should be a map")
	assert.Equal(t, "Error was handled by Next Hook", dataMap["message"])
	assert.Contains(t, dataMap, "error")
	assert.Contains(t, dataMap, "recovery_action")

	t.Log("✓ Error recovery scenario passed")
}

// TestRealWorldNextConditional tests conditional logic based on completion
func TestRealWorldNextConditional(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world Next Hook test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldNextContext("test-next-conditional", "tests.realworld-next")

	t.Run("ConditionalSuccess", func(t *testing.T) {
		messages := []context.Message{
			{Role: context.RoleUser, Content: "scenario: conditional"},
		}

		completion := &context.CompletionResponse{
			Content: "The operation completed successfully. All tasks are done.",
		}

		payload := &context.NextHookPayload{
			Messages:   messages,
			Completion: completion,
			Tools:      nil,
			Error:      "",
		}

		response, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Next hook failed: %v", err)
		}

		assert.NotNil(t, response, "Conditional scenario should return response")
		assert.NotNil(t, response.Data, "Response should have Data")

		dataMap, ok := response.Data.(map[string]interface{})
		assert.True(t, ok, "Data should be a map")
		assert.Equal(t, "Conditional analysis complete", dataMap["message"])
		assert.Contains(t, dataMap, "action")
		assert.Contains(t, dataMap, "conditions")

		t.Log("✓ Conditional (success) scenario passed")
	})

	t.Run("ConditionalDelegate", func(t *testing.T) {
		messages := []context.Message{
			{Role: context.RoleUser, Content: "scenario: conditional"},
		}

		completion := &context.CompletionResponse{
			Content: "I should delegate this request to another service for better handling.",
		}

		payload := &context.NextHookPayload{
			Messages:   messages,
			Completion: completion,
			Tools:      nil,
			Error:      "",
		}

		response, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Next hook failed: %v", err)
		}

		assert.NotNil(t, response, "Conditional delegate should return response")
		assert.NotNil(t, response.Delegate, "Should delegate based on condition")
		assert.Equal(t, "tests.create", response.Delegate.AgentID)

		t.Log("✓ Conditional (delegate) scenario passed")
	})
}

// TestRealWorldNextDefault tests default behavior
func TestRealWorldNextDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real world Next Hook test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	if err != nil {
		t.Fatalf("Failed to get assistant: %v", err)
	}

	ctx := newRealWorldNextContext("test-next-default", "tests.realworld-next")

	messages := []context.Message{
		{Role: context.RoleUser, Content: "Just a normal request"},
	}

	completion := &context.CompletionResponse{
		Content: "Here's the response to your request.",
	}

	payload := &context.NextHookPayload{
		Messages:   messages,
		Completion: completion,
		Tools:      nil,
		Error:      "",
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	if err != nil {
		t.Fatalf("Next hook failed: %v", err)
	}

	// Default behavior should return nil
	assert.Nil(t, response, "Default scenario should return nil for standard response")

	t.Log("✓ Default scenario passed")
}
