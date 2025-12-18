package assistant_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newAgentNextTestContext creates a test context
func newAgentNextTestContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.ID = chatID
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Client = context.Client{
		Type: "web",
		IP:   "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.IDGenerator = message.NewIDGenerator() // Initialize ID generator
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// TestAgentNextStandard tests agent with Next Hook returning nil (standard response)
func TestAgentNextStandard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	assert.NoError(t, err)

	ctx := newAgentNextTestContext("test-standard", "tests.realworld-next")
	messages := []context.Message{
		{Role: context.RoleUser, Content: "scenario: standard - Hello"},
	}

	response, err := agent.Stream(ctx, messages)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	assert.NotNil(t, response.Completion)
	assert.Nil(t, response.Next)

	// Verify response structure
	assert.Equal(t, "tests.realworld-next", response.AssistantID)
	assert.NotEmpty(t, response.ContextID)
	assert.NotEmpty(t, response.RequestID)
	assert.NotEmpty(t, response.TraceID)
	assert.NotEmpty(t, response.ChatID)

	// Verify completion has content
	assert.NotNil(t, response.Completion.Content)

	t.Log("✓ Standard response test passed")
}

// TestAgentNextCustomData tests agent with Next Hook returning custom data
func TestAgentNextCustomData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	assert.NoError(t, err)

	ctx := newAgentNextTestContext("test-custom", "tests.realworld-next")
	messages := []context.Message{
		{Role: context.RoleUser, Content: "scenario: custom_data - Give me info"},
	}

	response, err := agent.Stream(ctx, messages)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	assert.NotNil(t, response.Completion)
	assert.NotNil(t, response.Next)

	// Verify response structure
	assert.Equal(t, "tests.realworld-next", response.AssistantID)
	assert.NotEmpty(t, response.ContextID)
	assert.NotEmpty(t, response.RequestID)
	assert.NotEmpty(t, response.TraceID)

	// Verify custom data structure (from scenarioCustomData)
	// response.Next contains the "data" field value from NextHookResponse
	nextData, ok := response.Next.(map[string]interface{})
	assert.True(t, ok, "Next should be a map")
	assert.Equal(t, "custom_response", nextData["type"])
	assert.Equal(t, "This is a custom response from Next Hook", nextData["message"])
	assert.NotEmpty(t, nextData["timestamp"])
	assert.NotNil(t, nextData["message_count"])

	t.Log("✓ Custom data test passed")
}

// TestAgentNextDelegate tests agent with Next Hook delegating to another agent
func TestAgentNextDelegate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	assert.NoError(t, err)

	ctx := newAgentNextTestContext("test-delegate", "tests.realworld-next")
	messages := []context.Message{
		{Role: context.RoleUser, Content: "scenario: delegate - Forward this"},
	}

	response, err := agent.Stream(ctx, messages)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Verify response structure
	assert.NotEmpty(t, response.AssistantID)
	assert.NotEmpty(t, response.ContextID)
	assert.NotEmpty(t, response.RequestID)
	assert.NotEmpty(t, response.TraceID)

	// Verify completion (delegated agent should have returned completion)
	assert.NotNil(t, response.Completion)
	assert.NotNil(t, response.Completion.Content)

	// Next should be from the delegated agent
	// If delegated agent also has Next hook, it will be present
	t.Logf("✓ Delegation test passed (delegated to: %s)", response.AssistantID)
}

// TestAgentNextConditional tests agent with conditional logic in Next Hook
func TestAgentNextConditional(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.realworld-next")
	assert.NoError(t, err)

	ctx := newAgentNextTestContext("test-conditional", "tests.realworld-next")
	messages := []context.Message{
		// Use conditional_success sub-scenario for deterministic behavior
		// This avoids test flakiness caused by LLM response unpredictability
		{Role: context.RoleUser, Content: "scenario: conditional_success - Task completed"},
	}

	response, err := agent.Stream(ctx, messages)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	assert.NotNil(t, response.Next)

	// Verify response structure
	assert.Equal(t, "tests.realworld-next", response.AssistantID)
	assert.NotEmpty(t, response.ContextID)
	assert.NotEmpty(t, response.RequestID)
	assert.NotEmpty(t, response.TraceID)

	// Verify conditional response structure (from scenarioConditional)
	// response.Next contains the "data" field value from NextHookResponse
	nextData, ok := response.Next.(map[string]interface{})
	assert.True(t, ok, "Next should be a map")
	assert.Equal(t, "Conditional analysis complete", nextData["message"])
	assert.Contains(t, nextData, "action")
	assert.Contains(t, nextData, "reason")
	assert.Contains(t, nextData, "conditions")

	// Verify action is one of the expected values
	action, ok := nextData["action"].(string)
	assert.True(t, ok)
	assert.Contains(t, []string{"continue", "flag_for_review", "confirm_success", "summarize", "delegate"}, action)

	t.Log("✓ Conditional logic test passed")
}

// TestAgentWithoutNextHook tests agent without Next Hook
func TestAgentWithoutNextHook(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	assert.NoError(t, err)

	ctx := newAgentNextTestContext("test-no-next", "tests.create")
	messages := []context.Message{
		{Role: context.RoleUser, Content: "Hello"},
	}

	response, err := agent.Stream(ctx, messages)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	assert.Nil(t, response.Next)

	// Verify response structure
	assert.Equal(t, "tests.create", response.AssistantID)
	assert.NotEmpty(t, response.ContextID)
	assert.NotEmpty(t, response.RequestID)
	assert.NotEmpty(t, response.TraceID)
	assert.NotEmpty(t, response.ChatID)

	// Verify completion
	assert.NotNil(t, response.Completion)
	assert.NotNil(t, response.Completion.Content)

	t.Log("✓ No Next Hook test passed")
}
