package hook_test

import (
	stdContext "context"
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newTestContext creates a Context for testing with commonly used fields pre-populated.
// You can override any fields after creation as needed for specific test scenarios.
func newTestContext(chatID, assistantID string) *context.Context {
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

// TestCreate test the create hook
func TestCreate(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get the tests.create assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("The tests.create assistant has no script")
	}

	// Use the helper function to create a test context
	ctx := newTestContext("chat-test-create-hook", "tests.create")

	// Test scenario 1: Return null (should get nil response)
	t.Run("ReturnNull", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "return_null"}})
		if err != nil {
			t.Fatalf("Failed to create with null return: %s", err.Error())
		}
		if res != nil {
			t.Errorf("Expected nil response for null return, got: %v", res)
		}
	})

	// Test scenario 2: Return undefined (should get nil response)
	t.Run("ReturnUndefined", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "return_undefined"}})
		if err != nil {
			t.Fatalf("Failed to create with undefined return: %s", err.Error())
		}
		if res != nil {
			t.Errorf("Expected nil response for undefined return, got: %v", res)
		}
	})

	// Test scenario 3: Return empty object (should get empty HookCreateResponse)
	t.Run("ReturnEmpty", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "return_empty"}})
		if err != nil {
			t.Fatalf("Failed to create with empty return: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response for empty object, got nil")
		}
		if len(res.Messages) != 0 {
			t.Errorf("Expected empty messages, got: %d messages", len(res.Messages))
		}
	})

	// Test scenario 4: Return full response with all fields
	t.Run("ReturnFull", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "return_full"}})
		if err != nil {
			t.Fatalf("Failed to create with full return: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify messages
		if len(res.Messages) != 2 {
			t.Errorf("Expected 2 messages, got: %d", len(res.Messages))
		} else {
			if res.Messages[0].Role != context.RoleSystem {
				t.Errorf("Expected system role for first message, got: %s", res.Messages[0].Role)
			}
			if res.Messages[1].Role != context.RoleUser {
				t.Errorf("Expected user role for second message, got: %s", res.Messages[1].Role)
			}
		}

		// Verify audio config
		if res.Audio == nil {
			t.Error("Expected audio config, got nil")
		} else {
			if res.Audio.Voice != "alloy" {
				t.Errorf("Expected voice 'alloy', got: %s", res.Audio.Voice)
			}
			if res.Audio.Format != "mp3" {
				t.Errorf("Expected format 'mp3', got: %s", res.Audio.Format)
			}
		}

		// Verify temperature
		if res.Temperature == nil {
			t.Error("Expected temperature, got nil")
		} else if *res.Temperature != 0.7 {
			t.Errorf("Expected temperature 0.7, got: %f", *res.Temperature)
		}

		// Verify max_tokens
		if res.MaxTokens == nil {
			t.Error("Expected max_tokens, got nil")
		} else if *res.MaxTokens != 2000 {
			t.Errorf("Expected max_tokens 2000, got: %d", *res.MaxTokens)
		}

		// Verify max_completion_tokens
		if res.MaxCompletionTokens == nil {
			t.Error("Expected max_completion_tokens, got nil")
		} else if *res.MaxCompletionTokens != 1500 {
			t.Errorf("Expected max_completion_tokens 1500, got: %d", *res.MaxCompletionTokens)
		}
	})

	// Test scenario 5: Return partial response
	t.Run("ReturnPartial", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "return_partial"}})
		if err != nil {
			t.Fatalf("Failed to create with partial return: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify messages
		if len(res.Messages) != 1 {
			t.Errorf("Expected 1 message, got: %d", len(res.Messages))
		}

		// Verify temperature
		if res.Temperature == nil {
			t.Error("Expected temperature, got nil")
		} else if *res.Temperature != 0.5 {
			t.Errorf("Expected temperature 0.5, got: %f", *res.Temperature)
		}

		// Verify optional fields are nil
		if res.Audio != nil {
			t.Errorf("Expected audio to be nil, got: %v", res.Audio)
		}
		if res.MaxTokens != nil {
			t.Errorf("Expected max_tokens to be nil, got: %d", *res.MaxTokens)
		}
	})

	// Test scenario 6: Process call - calls models.__yao.role.Get and adds to messages
	t.Run("ReturnProcess", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "return_process"}})
		if err != nil {
			t.Fatalf("Failed to create with process return: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify messages - should have at least 1 (system message)
		if len(res.Messages) < 1 {
			t.Errorf("Expected at least 1 message, got: %d", len(res.Messages))
		} else {
			// First message should be system role
			if res.Messages[0].Role != context.RoleSystem {
				t.Errorf("Expected system role for first message, got: %s", res.Messages[0].Role)
			}
			// Check system message content
			if content, ok := res.Messages[0].Content.(string); ok {
				if content != "Here are the available roles in the system:" {
					t.Errorf("Unexpected system message content: %s", content)
				}
			}
		}
	})

	// Test scenario 7: Default response
	t.Run("ReturnDefault", func(t *testing.T) {
		testContent := "Hello, how are you?"
		res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: testContent}})
		if err != nil {
			t.Fatalf("Failed to create with default return: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify messages
		if len(res.Messages) != 1 {
			t.Errorf("Expected 1 message, got: %d", len(res.Messages))
		} else {
			if res.Messages[0].Role != context.RoleUser {
				t.Errorf("Expected user role, got: %s", res.Messages[0].Role)
			}
			if content, ok := res.Messages[0].Content.(string); ok {
				if content != testContent {
					t.Errorf("Expected content '%s', got: '%s'", testContent, content)
				}
			} else {
				t.Errorf("Expected string content, got: %T", res.Messages[0].Content)
			}
		}
	})

	// Test scenario 8: Verify context fields - validates all context fields in JavaScript
	t.Run("VerifyContext", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "verify_context"}})
		if err != nil {
			t.Fatalf("Failed to create with verify_context: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify we have messages
		if len(res.Messages) < 1 {
			t.Fatalf("Expected at least 1 message, got: %d", len(res.Messages))
		}

		// First message should be system role with success/failure indicator
		if res.Messages[0].Role != context.RoleSystem {
			t.Errorf("Expected system role for first message, got: %s", res.Messages[0].Role)
		}

		// Check the validation result
		content, ok := res.Messages[0].Content.(string)
		if !ok {
			t.Fatalf("Expected string content for system message, got: %T", res.Messages[0].Content)
		}

		// The content should be "success:all_fields_validated"
		if content != "success:all_fields_validated" {
			t.Errorf("Context validation failed: %s", content)

			// Print detailed validation results if available
			if len(res.Messages) > 1 {
				if details, ok := res.Messages[1].Content.(string); ok {
					t.Logf("Validation details:\n%s", details)
				}
			}
		} else {
			t.Log("✓ All context fields validated successfully in JavaScript")

			// Optionally print validation details
			if len(res.Messages) > 1 {
				if details, ok := res.Messages[1].Content.(string); ok {
					t.Logf("Validation details:\n%s", details)
				}
			}
		}
	})

	// Test scenario 9: Adjust context fields - tests that context fields can be modified by the hook
	t.Run("AdjustContext", func(t *testing.T) {
		// Create a fresh context for this test
		adjustCtx := newTestContext("chat-test-adjust", "tests.create")

		// Call the hook which should adjust context fields
		res, _, err := agent.HookScript.Create(adjustCtx, []context.Message{{Role: "user", Content: "adjust_context"}})
		if err != nil {
			t.Fatalf("Failed to create with adjust_context: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify the response contains adjusted fields
		// Note: AssistantID cannot be overridden by hooks, removed from HookCreateResponse
		if res.Connector != "adjusted-connector" {
			t.Errorf("Expected adjusted connector 'adjusted-connector', got: %s", res.Connector)
		}
		if res.Locale != "zh-cn" {
			t.Errorf("Expected adjusted locale 'zh-cn', got: %s", res.Locale)
		}
		if res.Theme != "dark" {
			t.Errorf("Expected adjusted theme 'dark', got: %s", res.Theme)
		}
		if res.Route != "/adjusted/route" {
			t.Errorf("Expected adjusted route '/adjusted/route', got: %s", res.Route)
		}

		// Verify metadata
		if res.Metadata == nil {
			t.Fatalf("Expected metadata, got nil")
		}
		if adjusted, ok := res.Metadata["adjusted"].(bool); !ok || !adjusted {
			t.Errorf("Expected metadata['adjusted'] = true, got: %v", res.Metadata["adjusted"])
		}

		// Verify context fields were actually updated
		// Note: AssistantID is immutable and cannot be overridden
		// Note: Connector is now in Options, not in Context
		if adjustCtx.Locale != "zh-cn" {
			t.Errorf("Context locale not updated. Expected 'zh-cn', got: %s", adjustCtx.Locale)
		}
		if adjustCtx.Theme != "dark" {
			t.Errorf("Context theme not updated. Expected 'dark', got: %s", adjustCtx.Theme)
		}
		if adjustCtx.Route != "/adjusted/route" {
			t.Errorf("Context route not updated. Expected '/adjusted/route', got: %s", adjustCtx.Route)
		}
		if adjustCtx.Metadata["adjusted"] != true {
			t.Errorf("Context metadata not updated. Expected metadata['adjusted'] = true, got: %v", adjustCtx.Metadata["adjusted"])
		}

		t.Log("✓ Context fields successfully adjusted by hook")
	})

	// Test scenario 10: Adjust uses configuration - tests that uses can be modified by the hook
	t.Run("AdjustUses", func(t *testing.T) {
		// Create a fresh context for this test
		usesCtx := newTestContext("chat-test-uses", "tests.create")

		// Call the hook which should adjust uses configuration
		res, _, err := agent.HookScript.Create(usesCtx, []context.Message{{Role: "user", Content: "adjust_uses"}})
		if err != nil {
			t.Fatalf("Failed to create with adjust_uses: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify the response contains uses configuration
		if res.Uses == nil {
			t.Fatalf("Expected uses configuration, got nil")
		}

		// Verify each uses field
		if res.Uses.Vision != "mcp:vision-server" {
			t.Errorf("Expected vision 'mcp:vision-server', got: %s", res.Uses.Vision)
		}
		if res.Uses.Audio != "mcp:audio-server" {
			t.Errorf("Expected audio 'mcp:audio-server', got: %s", res.Uses.Audio)
		}
		if res.Uses.Search != "agent" {
			t.Errorf("Expected search 'agent', got: %s", res.Uses.Search)
		}
		if res.Uses.Fetch != "mcp:fetch-server" {
			t.Errorf("Expected fetch 'mcp:fetch-server', got: %s", res.Uses.Fetch)
		}

		// Verify metadata
		if res.Metadata == nil {
			t.Fatalf("Expected metadata, got nil")
		}
		if usesAdjusted, ok := res.Metadata["uses_adjusted"].(bool); !ok || !usesAdjusted {
			t.Errorf("Expected metadata['uses_adjusted'] = true, got: %v", res.Metadata["uses_adjusted"])
		}

		// Now test that BuildRequest properly applies the uses configuration
		inputMessages := []context.Message{{Role: "user", Content: "test uses"}}
		_, options, err := agent.BuildRequest(usesCtx, inputMessages, res)
		if err != nil {
			t.Fatalf("Failed to build request: %s", err.Error())
		}

		// Verify that options.Uses has the values from createResponse
		if options.Uses == nil {
			t.Fatalf("Expected options.Uses to be set, got nil")
		}
		if options.Uses.Vision != "mcp:vision-server" {
			t.Errorf("Expected options.Uses.Vision 'mcp:vision-server', got: %s", options.Uses.Vision)
		}
		if options.Uses.Audio != "mcp:audio-server" {
			t.Errorf("Expected options.Uses.Audio 'mcp:audio-server', got: %s", options.Uses.Audio)
		}
		if options.Uses.Search != "agent" {
			t.Errorf("Expected options.Uses.Search 'agent', got: %s", options.Uses.Search)
		}
		if options.Uses.Fetch != "mcp:fetch-server" {
			t.Errorf("Expected options.Uses.Fetch 'mcp:fetch-server', got: %s", options.Uses.Fetch)
		}

		t.Log("✓ Uses configuration successfully adjusted by hook and applied to options")
	})

	// Test scenario 11: Adjust uses configuration with force_uses flag
	t.Run("AdjustUsesForce", func(t *testing.T) {
		// Create a fresh context for this test
		usesCtx := newTestContext("chat-test-uses-force", "tests.create")

		// Call the hook which should adjust uses configuration and set force_uses
		res, _, err := agent.HookScript.Create(usesCtx, []context.Message{{Role: "user", Content: "adjust_uses_force"}})
		if err != nil {
			t.Fatalf("Failed to create with adjust_uses_force: %s", err.Error())
		}
		if res == nil {
			t.Fatalf("Expected non-nil response, got nil")
		}

		// Verify the response contains uses configuration
		if res.Uses == nil {
			t.Fatalf("Expected uses configuration, got nil")
		}

		// Verify uses fields
		if res.Uses.Vision != "tests.vision-helper" {
			t.Errorf("Expected vision 'tests.vision-helper', got: %s", res.Uses.Vision)
		}
		if res.Uses.Audio != "mcp:audio-server" {
			t.Errorf("Expected audio 'mcp:audio-server', got: %s", res.Uses.Audio)
		}

		// Verify force_uses flag
		if res.ForceUses == nil {
			t.Fatalf("Expected force_uses to be set, got nil")
		}
		if !*res.ForceUses {
			t.Errorf("Expected force_uses to be true, got: %v", *res.ForceUses)
		}

		// Verify metadata
		if res.Metadata == nil {
			t.Fatalf("Expected metadata, got nil")
		}
		if usesForced, ok := res.Metadata["uses_forced"].(bool); !ok || !usesForced {
			t.Errorf("Expected metadata['uses_forced'] = true, got: %v", res.Metadata["uses_forced"])
		}

		// Now test that BuildRequest properly applies the force_uses flag
		inputMessages := []context.Message{{Role: "user", Content: "test force uses"}}
		_, options, err := agent.BuildRequest(usesCtx, inputMessages, res)
		if err != nil {
			t.Fatalf("Failed to build request: %s", err.Error())
		}

		// Verify that options.ForceUses is true
		if !options.ForceUses {
			t.Errorf("Expected options.ForceUses to be true, got: %v", options.ForceUses)
		}

		t.Log("✓ Uses configuration with force_uses flag successfully adjusted by hook and applied to options")
	})
}
