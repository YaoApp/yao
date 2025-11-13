package hook_test

import (
	defaultContext "context"
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestCreate test the create hook
func TestCreate(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.create")
	if err != nil {
		t.Fatalf("Failed to get the tests.create assistant: %s", err.Error())
	}

	if agent.Script == nil {
		t.Fatalf("The tests.create assistant has no script")
	}

	ctx := &context.Context{
		Context:     defaultContext.Background(),
		ChatID:      "chat-test-create-hook",
		AssistantID: "tests.create",
		Sid:         "test-session-create-hook",
	}

	// Test scenario 1: Return null (should get nil response)
	t.Run("ReturnNull", func(t *testing.T) {
		res, err := agent.Script.Create(ctx, []context.Message{{Role: "user", Content: "return_null"}})
		if err != nil {
			t.Fatalf("Failed to create with null return: %s", err.Error())
		}
		if res != nil {
			t.Errorf("Expected nil response for null return, got: %v", res)
		}
	})

	// Test scenario 2: Return undefined (should get nil response)
	t.Run("ReturnUndefined", func(t *testing.T) {
		res, err := agent.Script.Create(ctx, []context.Message{{Role: "user", Content: "return_undefined"}})
		if err != nil {
			t.Fatalf("Failed to create with undefined return: %s", err.Error())
		}
		if res != nil {
			t.Errorf("Expected nil response for undefined return, got: %v", res)
		}
	})

	// Test scenario 3: Return empty object (should get empty HookCreateResponse)
	t.Run("ReturnEmpty", func(t *testing.T) {
		res, err := agent.Script.Create(ctx, []context.Message{{Role: "user", Content: "return_empty"}})
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
		res, err := agent.Script.Create(ctx, []context.Message{{Role: "user", Content: "return_full"}})
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

		// Verify metadata
		if res.Metadata == nil {
			t.Error("Expected metadata, got nil")
		} else {
			if res.Metadata["test"] != "full_response" {
				t.Errorf("Expected metadata['test'] = 'full_response', got: %s", res.Metadata["test"])
			}
			if res.Metadata["user_id"] != "test_user_123" {
				t.Errorf("Expected metadata['user_id'] = 'test_user_123', got: %s", res.Metadata["user_id"])
			}
		}
	})

	// Test scenario 5: Return partial response
	t.Run("ReturnPartial", func(t *testing.T) {
		res, err := agent.Script.Create(ctx, []context.Message{{Role: "user", Content: "return_partial"}})
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
		res, err := agent.Script.Create(ctx, []context.Message{{Role: "user", Content: "return_process"}})
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

		// Verify metadata
		if res.Metadata == nil {
			t.Error("Expected metadata, got nil")
		} else {
			if res.Metadata["test"] != "process_call" {
				t.Errorf("Expected metadata['test'] = 'process_call', got: %s", res.Metadata["test"])
			}
			// roles_count should be present
			if _, ok := res.Metadata["roles_count"]; !ok {
				t.Error("Expected metadata['roles_count'] to be present")
			}
		}
	})

	// Test scenario 7: Default response
	t.Run("ReturnDefault", func(t *testing.T) {
		testContent := "Hello, how are you?"
		res, err := agent.Script.Create(ctx, []context.Message{{Role: "user", Content: testContent}})
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
}
