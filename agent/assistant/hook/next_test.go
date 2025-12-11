package hook_test

import (
	stdContext "context"
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newTestContextForNext creates a Context for testing Next Hook with commonly used fields pre-populated.
// You can override any fields after creation as needed for specific test scenarios.
func newTestContextForNext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:    "test-user",
		ClientID:   "test-client-id",
		Scope:      "openid profile email",
		SessionID:  "test-session-id",
		UserID:     "test-user-123",
		TeamID:     "test-team-456",
		TenantID:   "test-tenant-789",
		RememberMe: true,
		Constraints: types.DataConstraints{
			OwnerOnly:   false,
			CreatorOnly: false,
			EditorOnly:  false,
			TeamOnly:    true,
			Extra: map[string]interface{}{
				"department": "engineering",
				"region":     "us-west",
				"project":    "yao",
			},
		},
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "TestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// TestNext tests the Next hook
func TestNext(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.next")
	if err != nil {
		t.Fatalf("Failed to get the tests.next assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("The tests.next assistant has no script")
	}

	// Use the helper function to create a test context
	ctx := newTestContextForNext("chat-test-next-hook", "tests.next")

	// Test scenario 1: Return null (should get nil response)
	t.Run("ReturnNull", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "return_null"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion",
			},
			Tools: nil,
			Error: "",
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook with null return: %s", err.Error())
		}
		if res != nil {
			t.Errorf("Expected nil response for null return, got: %v", res)
		}
	})

	// Test scenario 2: Return undefined (should get nil response)
	t.Run("ReturnUndefined", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "return_undefined"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion",
			},
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook with undefined return: %s", err.Error())
		}
		if res != nil {
			t.Errorf("Expected nil response for undefined return, got: %v", res)
		}
	})

	// Test scenario 3: Return empty object (should get empty NextHookResponse)
	t.Run("ReturnEmpty", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "return_empty"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion",
			},
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook with empty return: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response for empty object, got nil")
		}
		if res.Delegate != nil {
			t.Errorf("Expected nil Delegate, got: %v", res.Delegate)
		}
		if res.Data != nil {
			t.Errorf("Expected nil Data, got: %v", res.Data)
		}
	})

	// Test scenario 4: Return custom data
	t.Run("ReturnCustomData", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "return_custom_data"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion",
			},
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook with custom data: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify Data is present
		if res.Data == nil {
			t.Fatalf("Expected Data to be present, got nil")
		}

		// Data should be a map
		dataMap, ok := res.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected Data to be map[string]interface{}, got: %T", res.Data)
		}

		// Verify custom data fields
		if message, ok := dataMap["message"].(string); !ok || message != "Custom response from Next Hook" {
			t.Errorf("Expected custom message, got: %v", dataMap["message"])
		}
		if test, ok := dataMap["test"].(bool); !ok || !test {
			t.Errorf("Expected test=true, got: %v", dataMap["test"])
		}
		if _, ok := dataMap["timestamp"]; !ok {
			t.Errorf("Expected timestamp field")
		}

		// Verify Delegate is nil
		if res.Delegate != nil {
			t.Errorf("Expected nil Delegate, got: %v", res.Delegate)
		}
	})

	// Test scenario 5: Return data with metadata
	t.Run("ReturnDataWithMetadata", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "return_data_with_metadata"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion",
			},
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify Data
		if res.Data == nil {
			t.Fatalf("Expected Data to be present, got nil")
		}

		dataMap, ok := res.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected Data to be map[string]interface{}, got: %T", res.Data)
		}

		if result, ok := dataMap["result"].(string); !ok || result != "success" {
			t.Errorf("Expected result='success', got: %v", dataMap["result"])
		}

		// Verify Metadata
		if res.Metadata == nil {
			t.Fatalf("Expected Metadata to be present, got nil")
		}

		if hook, ok := res.Metadata["hook"].(string); !ok || hook != "next" {
			t.Errorf("Expected hook='next', got: %v", res.Metadata["hook"])
		}
		if processed, ok := res.Metadata["processed"].(bool); !ok || !processed {
			t.Errorf("Expected processed=true, got: %v", res.Metadata["processed"])
		}
	})

	// Test scenario 6: Return delegate
	t.Run("ReturnDelegate", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "return_delegate"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion",
			},
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook with delegate: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify Delegate is present
		if res.Delegate == nil {
			t.Fatalf("Expected Delegate to be present, got nil")
		}

		// Verify delegate fields
		if res.Delegate.AgentID != "tests.create" {
			t.Errorf("Expected AgentID='tests.create', got: %s", res.Delegate.AgentID)
		}

		if len(res.Delegate.Messages) != 1 {
			t.Errorf("Expected 1 message, got: %d", len(res.Delegate.Messages))
		} else {
			if res.Delegate.Messages[0].Role != context.RoleUser {
				t.Errorf("Expected user role, got: %s", res.Delegate.Messages[0].Role)
			}
			if content, ok := res.Delegate.Messages[0].Content.(string); !ok || content != "Hello from delegated agent" {
				t.Errorf("Expected specific content, got: %v", res.Delegate.Messages[0].Content)
			}
		}

		// Verify Data is nil (only delegate, no custom data)
		if res.Data != nil {
			t.Logf("Note: Data is present alongside Delegate: %v", res.Data)
		}
	})

	// Test scenario 7: Verify payload structure
	t.Run("VerifyPayload", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleSystem, Content: "System message"},
				{Role: context.RoleUser, Content: "verify_payload"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion content",
				Usage: &message.UsageInfo{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			},
			Tools: []context.ToolCallResponse{
				{
					ToolCallID: "call_123",
					Server:     "test-server",
					Tool:       "test-tool",
					Result:     map[string]interface{}{"success": true},
					Error:      "",
				},
			},
			Error: "",
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify Data contains validation results
		if res.Data == nil {
			t.Fatalf("Expected Data with validation results, got nil")
		}

		dataMap, ok := res.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected Data to be map[string]interface{}, got: %T", res.Data)
		}

		if validation, ok := dataMap["validation"].(string); !ok || validation != "success" {
			t.Errorf("Expected validation='success', got: %v", dataMap["validation"])
		}

		if checks, ok := dataMap["checks"].([]interface{}); !ok {
			t.Errorf("Expected checks array, got: %T", dataMap["checks"])
		} else {
			t.Logf("✓ Payload validation checks: %d items", len(checks))
			for i, check := range checks {
				t.Logf("  [%d] %v", i, check)
			}
		}
	})

	// Test scenario 8: Verify tools processing
	t.Run("VerifyTools", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "verify_tools"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion",
			},
			Tools: []context.ToolCallResponse{
				{
					ToolCallID: "call_1",
					Server:     "server1",
					Tool:       "tool1",
					Result:     map[string]interface{}{"value": 42},
					Error:      "",
				},
				{
					ToolCallID: "call_2",
					Server:     "server2",
					Tool:       "tool2",
					Result:     nil,
					Error:      "Tool execution failed",
				},
			},
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify Data
		if res.Data == nil {
			t.Fatalf("Expected Data, got nil")
		}

		dataMap, ok := res.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected Data to be map, got: %T", res.Data)
		}

		// Verify tool statistics
		if totalTools, ok := dataMap["total_tools"].(float64); !ok || int(totalTools) != 2 {
			t.Errorf("Expected total_tools=2, got: %v", dataMap["total_tools"])
		}
		if successful, ok := dataMap["successful"].(float64); !ok || int(successful) != 1 {
			t.Errorf("Expected successful=1, got: %v", dataMap["successful"])
		}
		if failed, ok := dataMap["failed"].(float64); !ok || int(failed) != 1 {
			t.Errorf("Expected failed=1, got: %v", dataMap["failed"])
		}

		t.Log("✓ Tools processing validated successfully")
	})

	// Test scenario 9: Handle error
	t.Run("HandleError", func(t *testing.T) {
		payload := &context.NextHookPayload{
			Messages: []context.Message{
				{Role: context.RoleUser, Content: "handle_error"},
			},
			Completion: &context.CompletionResponse{
				Content: "Test completion",
			},
			Error: "Tool execution failed: timeout",
		}

		res, _, err := agent.HookScript.Next(ctx, payload)
		if err != nil {
			t.Fatalf("Failed to execute Next hook: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify error handling
		if res.Data == nil {
			t.Fatalf("Expected Data, got nil")
		}

		dataMap, ok := res.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected Data to be map, got: %T", res.Data)
		}

		if errorMsg, ok := dataMap["error"].(string); !ok || errorMsg != "Tool execution failed: timeout" {
			t.Errorf("Expected error message, got: %v", dataMap["error"])
		}
		if recovered, ok := dataMap["recovered"].(bool); !ok || !recovered {
			t.Errorf("Expected recovered=true, got: %v", dataMap["recovered"])
		}

		t.Log("✓ Error handling validated successfully")
	})
}
