package context_test

import (
	stdContext "context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
	"rogchap.com/v8go"
)

// TestJsValue test the JsValue function
func TestJsValue(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := context.New(stdContext.Background(), nil, "ChatID-123456")
	cxt.AssistantID = "AssistantID-1234"

	v8.RegisterFunction("testContextJsvalue", testContextJsvalueEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			return testContextJsvalue(cxt)
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	assert.Equal(t, "ChatID-123456", res)
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

func testContextJsvalueEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testContextJsvalueFunction)
}

func testContextJsvalueFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	var args = info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	ctx, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	chatID, err := ctx.Get("chat_id")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	return chatID
}

// TestJsValueConcurrent test the JsValue function with concurrent requests
func TestJsValueConcurrent(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	v8.RegisterFunction("testContextJsvalue", testContextJsvalueEmbed)

	// Number of concurrent goroutines
	concurrency := 10
	iterationsPerGoroutine := 5

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterationsPerGoroutine)
	results := make(chan string, concurrency*iterationsPerGoroutine)

	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				chatID := fmt.Sprintf("ChatID-%d-%d", routineID, j)
				assistantID := fmt.Sprintf("AssistantID-%d-%d", routineID, j)

				cxt := context.New(stdContext.Background(), nil, chatID)
				cxt.AssistantID = assistantID

				res, err := v8.Call(v8.CallOptions{}, `
					function test(cxt) {
						return testContextJsvalue(cxt)
					}`, cxt)

				if err != nil {
					errors <- fmt.Errorf("routine %d iteration %d failed: %v", routineID, j, err)
					return
				}

				results <- res.(string)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)
	close(results)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all results
	resultCount := 0
	for res := range results {
		assert.Contains(t, res, "ChatID-")
		resultCount++
	}

	// Verify the correct number of results
	expectedResults := concurrency * iterationsPerGoroutine
	assert.Equal(t, expectedResults, resultCount, "Should have %d results", expectedResults)

	// Verify all objects are cleaned up after GC
	// Note: objects should be released when v8 values are garbage collected
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

// TestJsValueRegistrationAndCleanup test the object registration and cleanup mechanism
func TestJsValueRegistrationAndCleanup(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	v8.RegisterFunction("testContextRegistration", testContextRegistrationEmbed)

	// Create multiple contexts and verify registration
	contextCount := 5
	for i := 0; i < contextCount; i++ {
		cxt := context.New(stdContext.Background(), nil, fmt.Sprintf("ChatID-%d", i))
		cxt.AssistantID = fmt.Sprintf("AssistantID-%d", i)

		_, err := v8.Call(v8.CallOptions{}, `
			function test(cxt) {
				return testContextRegistration(cxt)
			}`, cxt)

		if err != nil {
			t.Fatalf("Call %d failed: %v", i, err)
		}
	}

	// All objects should be cleaned up after v8.Call completes
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

func testContextRegistrationEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testContextRegistrationFunction)
}

func testContextRegistrationFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	var args = info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	ctx, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Verify the object has __release function
	release, err := ctx.Get("__release")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if !release.IsFunction() {
		return bridge.JsException(info.Context(), fmt.Errorf("__release should be a function"))
	}

	// Verify the object has internal field (goValueID is stored in internal field, not accessible from JS)
	if ctx.InternalFieldCount() == 0 {
		return bridge.JsException(info.Context(), fmt.Errorf("object should have internal field"))
	}

	goValueID := ctx.GetInternalField(0)
	if goValueID == nil || !goValueID.IsString() {
		return bridge.JsException(info.Context(), fmt.Errorf("internal field should contain goValueID string"))
	}

	val, err := v8go.NewValue(info.Context().Isolate(), true)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	return val
}

// TestJsValueAllFields test that all Context fields are properly exported to JavaScript
func TestJsValueAllFields(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	authInfo := &types.AuthorizedInfo{
		Subject:  "test-user",
		ClientID: "test-client",
		UserID:   "user-123",
		TeamID:   "team-456",
		TenantID: "tenant-789",
		Constraints: types.DataConstraints{
			OwnerOnly:   true,
			CreatorOnly: false,
			TeamOnly:    true,
			Extra: map[string]interface{}{
				"department": "engineering",
				"region":     "us-west",
			},
		},
	}

	cxt := context.New(stdContext.Background(), authInfo, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Locale = "zh-cn"
	cxt.Theme = "dark"
	cxt.Client = context.Client{
		Type:      "web",
		UserAgent: "Mozilla/5.0",
		IP:        "127.0.0.1",
	}
	cxt.Referer = "api"
	cxt.Accept = "cui-web"
	cxt.Route = "/dashboard/home"
	cxt.Metadata = map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	v8.RegisterFunction("testAllFields", testAllFieldsEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			return testAllFields(cxt)
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	// Verify all fields
	assert.Equal(t, "test-chat-id", result["chat_id"], "chat_id mismatch")
	assert.Equal(t, "test-assistant-id", result["assistant_id"], "assistant_id mismatch")
	assert.Equal(t, "zh-cn", result["locale"], "locale mismatch")
	assert.Equal(t, "dark", result["theme"], "theme mismatch")
	assert.Equal(t, "api", result["referer"], "referer mismatch")
	assert.Equal(t, "cui-web", result["accept"], "accept mismatch")
	assert.Equal(t, "/dashboard/home", result["route"], "route mismatch")

	// Verify client object
	client, ok := result["client"].(map[string]interface{})
	assert.True(t, ok, "client should be an object")
	assert.Equal(t, "web", client["type"], "client.type mismatch")
	assert.Equal(t, "Mozilla/5.0", client["user_agent"], "client.user_agent mismatch")
	assert.Equal(t, "127.0.0.1", client["ip"], "client.ip mismatch")

	// Verify metadata object
	metadata, ok := result["metadata"].(map[string]interface{})
	assert.True(t, ok, "metadata should be an object")
	assert.Equal(t, "value1", metadata["key1"], "metadata.key1 mismatch")
	assert.Equal(t, float64(123), metadata["key2"], "metadata.key2 mismatch")
	assert.Equal(t, true, metadata["key3"], "metadata.key3 mismatch")

	// Verify authorized object
	authorized, ok := result["authorized"].(map[string]interface{})
	assert.True(t, ok, "authorized should be an object")
	assert.Equal(t, "test-user", authorized["sub"], "authorized.sub mismatch")
	assert.Equal(t, "test-client", authorized["client_id"], "authorized.client_id mismatch")
	assert.Equal(t, "user-123", authorized["user_id"], "authorized.user_id mismatch")
	assert.Equal(t, "team-456", authorized["team_id"], "authorized.team_id mismatch")
	assert.Equal(t, "tenant-789", authorized["tenant_id"], "authorized.tenant_id mismatch")

	// Verify authorized.constraints object
	constraints, ok := authorized["constraints"].(map[string]interface{})
	assert.True(t, ok, "authorized.constraints should be an object")
	assert.Equal(t, true, constraints["owner_only"], "constraints.owner_only mismatch")
	// creator_only is false, and with omitempty it may not be present
	if creatorOnly, exists := constraints["creator_only"]; exists {
		assert.Equal(t, false, creatorOnly, "constraints.creator_only mismatch")
	}
	assert.Equal(t, true, constraints["team_only"], "constraints.team_only mismatch")

	// Verify constraints.extra object
	extra, ok := constraints["extra"].(map[string]interface{})
	assert.True(t, ok, "constraints.extra should be an object")
	assert.Equal(t, "engineering", extra["department"], "constraints.extra.department mismatch")
	assert.Equal(t, "us-west", extra["region"], "constraints.extra.region mismatch")

	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

func testAllFieldsEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testAllFieldsFunction)
}

func testAllFieldsFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	var args = info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	ctx, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Extract all fields and return as a map
	result := map[string]interface{}{}

	// Helper function to get field value
	getField := func(name string) (interface{}, bool) {
		val, err := ctx.Get(name)
		if err != nil || val.IsUndefined() {
			return nil, false
		}
		goVal, err := bridge.GoValue(val, info.Context())
		if err != nil {
			return nil, false
		}
		return goVal, true
	}

	if val, ok := getField("chat_id"); ok {
		result["chat_id"] = val
	}
	if val, ok := getField("assistant_id"); ok {
		result["assistant_id"] = val
	}
	if val, ok := getField("locale"); ok {
		result["locale"] = val
	}
	if val, ok := getField("theme"); ok {
		result["theme"] = val
	}
	if val, ok := getField("client"); ok {
		result["client"] = val
	}
	if val, ok := getField("referer"); ok {
		result["referer"] = val
	}
	if val, ok := getField("accept"); ok {
		result["accept"] = val
	}
	if val, ok := getField("route"); ok {
		result["route"] = val
	}
	if val, ok := getField("metadata"); ok {
		result["metadata"] = val
	}
	if val, ok := getField("authorized"); ok {
		result["authorized"] = val
	}

	// Check for deprecated fields - they should NOT exist
	if val, ok := getField("sid"); ok {
		result["sid"] = val
	}
	if val, ok := getField("silent"); ok {
		result["silent"] = val
	}

	jsVal, err := bridge.JsValue(info.Context(), result)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	return jsVal
}

// TestJsValueTrace test the Trace method on Context
func TestJsValueTrace(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := context.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Stack = &context.Stack{
		TraceID: "test-trace-id",
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			// Get trace from context (property, not method call)
			const trace = cxt.trace
			
			// Verify trace object exists
			if (!trace) {
				throw new Error("Trace returned null or undefined")
			}
			
			// Verify trace has expected methods
			if (typeof trace.Add !== 'function') {
				throw new Error("trace.Add is not a function")
			}
			if (typeof trace.Info !== 'function') {
				throw new Error("trace.Info is not a function")
			}
			
			// Actually use the trace - add a node
			const node = trace.Add({ type: "test", content: "Test from context" }, { label: "Test Node" })
			
			// Log some info
			trace.Info("Testing trace from context")
			node.Info("Node info message")
			
			// Complete the node
			node.Complete({ result: "success" })
			
			// Return verification info
			return {
				trace_id: trace.id,
				node_id: node.id,
				success: true
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	// Verify trace was accessible and operations succeeded
	assert.Equal(t, "test-trace-id", result["trace_id"], "trace_id should match")
	assert.NotEmpty(t, result["node_id"], "node_id should not be empty")
	assert.Equal(t, true, result["success"], "operation should succeed")
}

// TestJsValueAuthorizedAndMetadata test the authorized and metadata fields
func TestJsValueAuthorizedAndMetadata(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	authInfo := &types.AuthorizedInfo{
		UserID:   "user-123",
		TenantID: "tenant-456",
		ClientID: "client-789",
	}
	cxt := context.New(stdContext.Background(), authInfo, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Metadata = map[string]interface{}{
		"request_id": "req-001",
		"source":     "api",
		"version":    "1.0.0",
	}

	v8.RegisterFunction("testAuthorizedMetadata", testAuthorizedMetadataEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			return testAuthorizedMetadata(cxt)
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	// Verify authorized object
	authorized, ok := result["authorized"].(map[string]interface{})
	assert.True(t, ok, "authorized should be an object")
	assert.Equal(t, "user-123", authorized["user_id"], "authorized.user_id mismatch")
	assert.Equal(t, "tenant-456", authorized["tenant_id"], "authorized.tenant_id mismatch")
	assert.Equal(t, "client-789", authorized["client_id"], "authorized.client_id mismatch")

	// Verify metadata object
	metadata, ok := result["metadata"].(map[string]interface{})
	assert.True(t, ok, "metadata should be an object")
	assert.Equal(t, "req-001", metadata["request_id"], "metadata.request_id mismatch")
	assert.Equal(t, "api", metadata["source"], "metadata.source mismatch")
	assert.Equal(t, "1.0.0", metadata["version"], "metadata.version mismatch")
}

func testAuthorizedMetadataEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testAuthorizedMetadataFunction)
}

func testAuthorizedMetadataFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	var args = info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	ctx, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Extract authorized and metadata fields
	result := map[string]interface{}{}

	// Get authorized
	authorizedVal, err := ctx.Get("authorized")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	if !authorizedVal.IsUndefined() && !authorizedVal.IsNull() {
		authorized, err := bridge.GoValue(authorizedVal, info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}
		result["authorized"] = authorized
	}

	// Get metadata
	metadataVal, err := ctx.Get("metadata")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	if !metadataVal.IsUndefined() && !metadataVal.IsNull() {
		metadata, err := bridge.GoValue(metadataVal, info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}
		result["metadata"] = metadata
	}

	jsVal, err := bridge.JsValue(info.Context(), result)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	return jsVal
}

// TestJsValueAuthorizedNil test when authorized is nil
func TestJsValueAuthorizedNil(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := context.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Metadata = nil // Explicitly nil (should be empty object)

	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			// Debug: check the actual values
			const authorized = cxt.authorized;
			const metadata = cxt.metadata;
			
			return {
				authorized_type: typeof authorized,
				authorized_is_null: authorized === null,
				authorized_is_undefined: authorized === undefined,
				metadata_type: typeof metadata,
				metadata_is_object: typeof metadata === 'object' && metadata !== null,
				metadata_is_empty: metadata && Object.keys(metadata).length === 0,
				has_authorized: 'authorized' in cxt,
				has_metadata: 'metadata' in cxt
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	// Verify authorized exists and is an empty object when nil
	assert.Equal(t, true, result["has_authorized"], "authorized property should exist")
	assert.Equal(t, "object", result["authorized_type"], "authorized should be an object")
	assert.Equal(t, true, result["metadata_is_object"], "authorized should be an object (not null)")

	// Verify metadata is an empty object when not set
	assert.Equal(t, true, result["has_metadata"], "metadata property should exist")
	assert.Equal(t, "object", result["metadata_type"], "metadata should be an object")
	assert.Equal(t, true, result["metadata_is_object"], "metadata should be an object")
	assert.Equal(t, true, result["metadata_is_empty"], "metadata should be empty object when not set")
}
