package assistant_test

import (
	stdContext "context"
	"testing"

	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newTestContext creates a Context for testing with commonly used fields pre-populated
func newTestContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client-id",
		UserID:    "test-user-123",
		TeamID:    "test-team-456",
		TenantID:  "test-tenant-789",
		SessionID: "test-session-id",
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
	ctx.Route = "/test/route"
	ctx.Metadata = map[string]interface{}{
		"test": "context_metadata",
	}
	return ctx
}

// TestBuildRequest tests the BuildRequest function
func TestBuildRequest(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.buildrequest")
	if err != nil {
		t.Fatalf("Failed to get tests.buildrequest assistant: %s", err.Error())
	}

	if agent.HookScript == nil {
		t.Fatalf("The tests.buildrequest assistant has no script")
	}

	ctx := newTestContext("chat-test-buildrequest", "tests.buildrequest")

	// Test 1: No override from hook - should use ast.Options and ctx values
	t.Run("NoOverride", func(t *testing.T) {
		inputMessages := []context.Message{{Role: "user", Content: "no_override"}}

		// Call Create hook
		createResponse, _, err := agent.HookScript.Create(ctx, inputMessages, &context.Options{})
		if err != nil {
			t.Fatalf("Failed to call Create hook: %s", err.Error())
		}

		// Build LLM request
		_, options, err := agent.BuildRequest(ctx, inputMessages, createResponse)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify options - should use ast.Options values
		if options.Temperature == nil {
			t.Error("Expected temperature from ast.Options, got nil")
		} else if *options.Temperature != 0.5 {
			t.Errorf("Expected temperature 0.5 from ast.Options, got: %f", *options.Temperature)
		}

		if options.MaxTokens == nil {
			t.Error("Expected max_tokens from ast.Options, got nil")
		} else if *options.MaxTokens != 1000 {
			t.Errorf("Expected max_tokens 1000 from ast.Options, got: %d", *options.MaxTokens)
		}

		if options.TopP == nil {
			t.Error("Expected top_p from ast.Options, got nil")
		} else if *options.TopP != 0.9 {
			t.Errorf("Expected top_p 0.9 from ast.Options, got: %f", *options.TopP)
		}

		// Verify ctx values
		if options.Route != "/test/route" {
			t.Errorf("Expected route '/test/route' from ctx, got: %s", options.Route)
		}

		if options.Metadata == nil {
			t.Error("Expected metadata from ctx, got nil")
		} else if options.Metadata["test"] != "context_metadata" {
			t.Errorf("Expected metadata from ctx, got: %v", options.Metadata)
		}

		t.Log("✓ No override: ast.Options and ctx values used correctly")
	})

	// Test 2: Override temperature - hook value should take priority
	t.Run("OverrideTemperature", func(t *testing.T) {
		inputMessages := []context.Message{{Role: "user", Content: "override_temperature"}}

		createResponse, _, err := agent.HookScript.Create(ctx, inputMessages, &context.Options{})
		if err != nil {
			t.Fatalf("Failed to call Create hook: %s", err.Error())
		}

		_, options, err := agent.BuildRequest(ctx, inputMessages, createResponse)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify temperature override
		if options.Temperature == nil {
			t.Error("Expected temperature, got nil")
		} else if *options.Temperature != 0.9 {
			t.Errorf("Expected temperature 0.9 from hook, got: %f", *options.Temperature)
		}

		// Other values should still come from ast.Options
		if options.MaxTokens == nil {
			t.Error("Expected max_tokens from ast.Options, got nil")
		} else if *options.MaxTokens != 1000 {
			t.Errorf("Expected max_tokens 1000 from ast.Options, got: %d", *options.MaxTokens)
		}

		t.Log("✓ Temperature override: hook value takes priority over ast.Options")
	})

	// Test 3: Override all - all hook values should take priority
	t.Run("OverrideAll", func(t *testing.T) {
		inputMessages := []context.Message{{Role: "user", Content: "override_all"}}

		createResponse, _, err := agent.HookScript.Create(ctx, inputMessages, &context.Options{})
		if err != nil {
			t.Fatalf("Failed to call Create hook: %s", err.Error())
		}

		_, options, err := agent.BuildRequest(ctx, inputMessages, createResponse)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify all overrides
		if options.Temperature == nil || *options.Temperature != 0.8 {
			t.Errorf("Expected temperature 0.8 from hook, got: %v", options.Temperature)
		}

		if options.MaxTokens == nil || *options.MaxTokens != 2000 {
			t.Errorf("Expected max_tokens 2000 from hook, got: %v", options.MaxTokens)
		}

		if options.MaxCompletionTokens == nil || *options.MaxCompletionTokens != 1800 {
			t.Errorf("Expected max_completion_tokens 1800 from hook, got: %v", options.MaxCompletionTokens)
		}

		if options.Audio == nil {
			t.Error("Expected audio from hook, got nil")
		} else {
			if options.Audio.Voice != "alloy" {
				t.Errorf("Expected voice 'alloy', got: %s", options.Audio.Voice)
			}
			if options.Audio.Format != "mp3" {
				t.Errorf("Expected format 'mp3', got: %s", options.Audio.Format)
			}
		}

		if options.Route != "/hook/route" {
			t.Errorf("Expected route '/hook/route' from hook, got: %s", options.Route)
		}

		if options.Metadata == nil {
			t.Error("Expected metadata from hook, got nil")
		} else {
			if options.Metadata["source"] != "hook" {
				t.Errorf("Expected metadata['source'] = 'hook', got: %v", options.Metadata["source"])
			}
		}

		t.Log("✓ Override all: all hook values take priority")
	})

	// Test 4: Override route and metadata - tests CUI context priority
	t.Run("OverrideRouteMetadata", func(t *testing.T) {
		inputMessages := []context.Message{{Role: "user", Content: "override_route_metadata"}}

		createResponse, _, err := agent.HookScript.Create(ctx, inputMessages, &context.Options{})
		if err != nil {
			t.Fatalf("Failed to call Create hook: %s", err.Error())
		}

		_, options, err := agent.BuildRequest(ctx, inputMessages, createResponse)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify route override
		if options.Route != "/custom/route" {
			t.Errorf("Expected route '/custom/route' from hook, got: %s", options.Route)
		}

		// Verify metadata merge (ctx metadata should be merged with hook metadata)
		if options.Metadata == nil {
			t.Error("Expected metadata, got nil")
		} else {
			// Hook metadata should be present
			if options.Metadata["custom"] != true {
				t.Errorf("Expected metadata['custom'] = true from hook, got: %v", options.Metadata["custom"])
			}
			if options.Metadata["hook_data"] != "test" {
				t.Errorf("Expected metadata['hook_data'] = 'test' from hook, got: %v", options.Metadata["hook_data"])
			}
			// Original ctx metadata should still be there (merged)
			if options.Metadata["test"] != "context_metadata" {
				t.Errorf("Expected original ctx metadata to be preserved, got: %v", options.Metadata)
			}
		}

		// Other values should still come from ast.Options
		if options.Temperature == nil || *options.Temperature != 0.5 {
			t.Errorf("Expected temperature 0.5 from ast.Options, got: %v", options.Temperature)
		}

		t.Log("✓ Route and metadata override: hook values take priority, metadata merged")
	})

	// Test 5: Nil createResponse - should use ast.Options and ctx values
	t.Run("NilCreateResponse", func(t *testing.T) {
		// Create a fresh context for this test
		freshCtx := newTestContext("chat-test-nil", "tests.buildrequest")
		inputMessages := []context.Message{{Role: "user", Content: "test message"}}

		_, options, err := agent.BuildRequest(freshCtx, inputMessages, nil)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Should use ast.Options values
		if options.Temperature == nil || *options.Temperature != 0.5 {
			t.Errorf("Expected temperature 0.5 from ast.Options, got: %v", options.Temperature)
		}

		// Should use ctx values
		if options.Route != "/test/route" {
			t.Errorf("Expected route '/test/route' from ctx, got: %s", options.Route)
		}

		t.Log("✓ Nil createResponse: ast.Options and ctx values used")
	})

	// Test 6: ResponseFormat with *context.ResponseFormat
	t.Run("ResponseFormatStruct", func(t *testing.T) {
		freshCtx := newTestContext("chat-test-response-format", "tests.buildrequest")
		inputMessages := []context.Message{{Role: "user", Content: "test message"}}

		// Create a test agent with response_format in Options
		testAgent := *agent
		strict := true
		testAgent.Options = map[string]interface{}{
			"temperature": 0.7,
			"response_format": &context.ResponseFormat{
				Type: context.ResponseFormatJSONSchema,
				JSONSchema: &context.JSONSchema{
					Name:        "test_schema",
					Description: "Test schema description",
					Schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type": "string",
							},
						},
					},
					Strict: &strict,
				},
			},
		}

		_, options, err := testAgent.BuildRequest(freshCtx, inputMessages, nil)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify ResponseFormat
		if options.ResponseFormat == nil {
			t.Fatal("Expected ResponseFormat, got nil")
		}

		if options.ResponseFormat.Type != context.ResponseFormatJSONSchema {
			t.Errorf("Expected type 'json_schema', got: %s", options.ResponseFormat.Type)
		}

		if options.ResponseFormat.JSONSchema == nil {
			t.Fatal("Expected JSONSchema, got nil")
		}

		if options.ResponseFormat.JSONSchema.Name != "test_schema" {
			t.Errorf("Expected schema name 'test_schema', got: %s", options.ResponseFormat.JSONSchema.Name)
		}

		if options.ResponseFormat.JSONSchema.Description != "Test schema description" {
			t.Errorf("Expected schema description 'Test schema description', got: %s", options.ResponseFormat.JSONSchema.Description)
		}

		if options.ResponseFormat.JSONSchema.Strict == nil || *options.ResponseFormat.JSONSchema.Strict != true {
			t.Errorf("Expected strict = true, got: %v", options.ResponseFormat.JSONSchema.Strict)
		}

		t.Log("✓ ResponseFormat with *context.ResponseFormat struct works correctly")
	})

	// Test 7: ResponseFormat with legacy map[string]interface{}
	t.Run("ResponseFormatLegacyMap", func(t *testing.T) {
		freshCtx := newTestContext("chat-test-response-format-map", "tests.buildrequest")
		inputMessages := []context.Message{{Role: "user", Content: "test message"}}

		// Create a test agent with legacy map format
		testAgent := *agent
		testAgent.Options = map[string]interface{}{
			"temperature": 0.7,
			"response_format": map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"name":        "legacy_schema",
					"description": "Legacy schema format",
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"email": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"strict": true,
				},
			},
		}

		_, options, err := testAgent.BuildRequest(freshCtx, inputMessages, nil)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify ResponseFormat was converted from map
		if options.ResponseFormat == nil {
			t.Fatal("Expected ResponseFormat, got nil")
		}

		if options.ResponseFormat.Type != context.ResponseFormatJSONSchema {
			t.Errorf("Expected type 'json_schema', got: %s", options.ResponseFormat.Type)
		}

		if options.ResponseFormat.JSONSchema == nil {
			t.Fatal("Expected JSONSchema, got nil")
		}

		if options.ResponseFormat.JSONSchema.Name != "legacy_schema" {
			t.Errorf("Expected schema name 'legacy_schema', got: %s", options.ResponseFormat.JSONSchema.Name)
		}

		if options.ResponseFormat.JSONSchema.Description != "Legacy schema format" {
			t.Errorf("Expected schema description 'Legacy schema format', got: %s", options.ResponseFormat.JSONSchema.Description)
		}

		t.Log("✓ ResponseFormat with legacy map[string]interface{} format works correctly")
	})

	// Test 8: ResponseFormat with simple type (text or json_object)
	t.Run("ResponseFormatSimpleType", func(t *testing.T) {
		freshCtx := newTestContext("chat-test-response-format-simple", "tests.buildrequest")
		inputMessages := []context.Message{{Role: "user", Content: "test message"}}

		// Create a test agent with simple response_format
		testAgent := *agent
		testAgent.Options = map[string]interface{}{
			"temperature": 0.7,
			"response_format": map[string]interface{}{
				"type": "json_object",
			},
		}

		_, options, err := testAgent.BuildRequest(freshCtx, inputMessages, nil)
		if err != nil {
			t.Fatalf("Failed to build LLM request: %s", err.Error())
		}

		// Verify ResponseFormat
		if options.ResponseFormat == nil {
			t.Fatal("Expected ResponseFormat, got nil")
		}

		if options.ResponseFormat.Type != context.ResponseFormatJSON {
			t.Errorf("Expected type 'json_object', got: %s", options.ResponseFormat.Type)
		}

		if options.ResponseFormat.JSONSchema != nil {
			t.Errorf("Expected JSONSchema to be nil for simple type, got: %v", options.ResponseFormat.JSONSchema)
		}

		t.Log("✓ ResponseFormat with simple type (json_object) works correctly")
	})
}
