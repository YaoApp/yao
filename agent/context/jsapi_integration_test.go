//go:build integration

package context_test

import (
	stdContext "context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"rogchap.com/v8go"
)

func TestJsValue(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "ChatID-123456")
	cxt.AssistantID = "AssistantID-1234"

	v8.RegisterFunction("testContextJsvalue", testContextJsvalueEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			return testContextJsvalue(cxt)
		}`, cxt)
	require.NoError(t, err, "Call failed")
	assert.Equal(t, "ChatID-123456", res)
}

func testContextJsvalueEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testContextJsvalueFunction)
}

func testContextJsvalueFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
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

func TestJsValueConcurrent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	v8.RegisterFunction("testContextJsvalue", testContextJsvalueEmbed)

	concurrency := 10
	iterationsPerGoroutine := 5

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterationsPerGoroutine)
	results := make(chan string, concurrency*iterationsPerGoroutine)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				chatID := fmt.Sprintf("ChatID-%d-%d", routineID, j)
				assistantID := fmt.Sprintf("AssistantID-%d-%d", routineID, j)

				cxt := agentctx.New(stdContext.Background(), nil, chatID)
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

	wg.Wait()
	close(errors)
	close(results)

	for err := range errors {
		t.Error(err)
	}

	resultCount := 0
	for res := range results {
		assert.Contains(t, res, "ChatID-")
		resultCount++
	}

	expectedResults := concurrency * iterationsPerGoroutine
	assert.Equal(t, expectedResults, resultCount, "Should have %d results", expectedResults)
}

func TestJsValueRegistrationAndCleanup(t *testing.T) {
	testprepare.PrepareSandbox(t)

	v8.RegisterFunction("testContextRegistration", testContextRegistrationEmbed)

	contextCount := 5
	for i := 0; i < contextCount; i++ {
		cxt := agentctx.New(stdContext.Background(), nil, fmt.Sprintf("ChatID-%d", i))
		cxt.AssistantID = fmt.Sprintf("AssistantID-%d", i)

		_, err := v8.Call(v8.CallOptions{}, `
			function test(cxt) {
				return testContextRegistration(cxt)
			}`, cxt)

		require.NoError(t, err, "Call %d failed", i)
	}
}

func testContextRegistrationEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testContextRegistrationFunction)
}

func testContextRegistrationFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	ctx, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	release, err := ctx.Get("__release")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if !release.IsFunction() {
		return bridge.JsException(info.Context(), fmt.Errorf("__release should be a function"))
	}

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

func TestJsValueAllFields(t *testing.T) {
	testprepare.PrepareSandbox(t)

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

	cxt := agentctx.New(stdContext.Background(), authInfo, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Locale = "zh-cn"
	cxt.Theme = "dark"
	cxt.Client = agentctx.Client{
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
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, "test-chat-id", result["chat_id"], "chat_id mismatch")
	assert.Equal(t, "test-assistant-id", result["assistant_id"], "assistant_id mismatch")
	assert.Equal(t, "zh-cn", result["locale"], "locale mismatch")
	assert.Equal(t, "dark", result["theme"], "theme mismatch")
	assert.Equal(t, "api", result["referer"], "referer mismatch")
	assert.Equal(t, "cui-web", result["accept"], "accept mismatch")
	assert.Equal(t, "/dashboard/home", result["route"], "route mismatch")

	client, ok := result["client"].(map[string]interface{})
	assert.True(t, ok, "client should be an object")
	assert.Equal(t, "web", client["type"], "client.type mismatch")
	assert.Equal(t, "Mozilla/5.0", client["user_agent"], "client.user_agent mismatch")
	assert.Equal(t, "127.0.0.1", client["ip"], "client.ip mismatch")

	metadata, ok := result["metadata"].(map[string]interface{})
	assert.True(t, ok, "metadata should be an object")
	assert.Equal(t, "value1", metadata["key1"], "metadata.key1 mismatch")
	assert.Equal(t, float64(123), metadata["key2"], "metadata.key2 mismatch")
	assert.Equal(t, true, metadata["key3"], "metadata.key3 mismatch")

	authorized, ok := result["authorized"].(map[string]interface{})
	assert.True(t, ok, "authorized should be an object")
	assert.Equal(t, "test-user", authorized["sub"], "authorized.sub mismatch")
	assert.Equal(t, "test-client", authorized["client_id"], "authorized.client_id mismatch")
	assert.Equal(t, "user-123", authorized["user_id"], "authorized.user_id mismatch")
	assert.Equal(t, "team-456", authorized["team_id"], "authorized.team_id mismatch")
	assert.Equal(t, "tenant-789", authorized["tenant_id"], "authorized.tenant_id mismatch")

	constraints, ok := authorized["constraints"].(map[string]interface{})
	assert.True(t, ok, "authorized.constraints should be an object")
	assert.Equal(t, true, constraints["owner_only"], "constraints.owner_only mismatch")
	if creatorOnly, exists := constraints["creator_only"]; exists {
		assert.Equal(t, false, creatorOnly, "constraints.creator_only mismatch")
	}
	assert.Equal(t, true, constraints["team_only"], "constraints.team_only mismatch")

	extra, ok := constraints["extra"].(map[string]interface{})
	assert.True(t, ok, "constraints.extra should be an object")
	assert.Equal(t, "engineering", extra["department"], "constraints.extra.department mismatch")
	assert.Equal(t, "us-west", extra["region"], "constraints.extra.region mismatch")
}

func testAllFieldsEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testAllFieldsFunction)
}

func testAllFieldsFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	ctx, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	result := map[string]interface{}{}

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

	fields := []string{"chat_id", "assistant_id", "locale", "theme", "client", "referer", "accept", "route", "metadata", "authorized", "sid", "silent"}
	for _, f := range fields {
		if val, ok := getField(f); ok {
			result[f] = val
		}
	}

	jsVal, err := bridge.JsValue(info.Context(), result)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	return jsVal
}

func TestJsValueTrace(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Stack = &agentctx.Stack{
		TraceID: "test-trace-id",
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			const trace = cxt.trace
			
			if (!trace) {
				throw new Error("Trace returned null or undefined")
			}
			
			if (typeof trace.Add !== 'function') {
				throw new Error("trace.Add is not a function")
			}
			if (typeof trace.Info !== 'function') {
				throw new Error("trace.Info is not a function")
			}
			
			const node = trace.Add({ type: "test", content: "Test from context" }, { label: "Test Node" })
			trace.Info("Testing trace from context")
			node.Info("Node info message")
			node.Complete({ result: "success" })
			
			return {
				trace_id: trace.id,
				node_id: node.id,
				success: true
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, "test-trace-id", result["trace_id"], "trace_id should match")
	assert.NotEmpty(t, result["node_id"], "node_id should not be empty")
	assert.Equal(t, true, result["success"], "operation should succeed")
}

func TestJsValueAuthorizedAndMetadata(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authInfo := &types.AuthorizedInfo{
		UserID:   "user-123",
		TenantID: "tenant-456",
		ClientID: "client-789",
	}
	cxt := agentctx.New(stdContext.Background(), authInfo, "test-chat-id")
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
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	authorized, ok := result["authorized"].(map[string]interface{})
	assert.True(t, ok, "authorized should be an object")
	assert.Equal(t, "user-123", authorized["user_id"], "authorized.user_id mismatch")
	assert.Equal(t, "tenant-456", authorized["tenant_id"], "authorized.tenant_id mismatch")
	assert.Equal(t, "client-789", authorized["client_id"], "authorized.client_id mismatch")

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
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	ctx, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	result := map[string]interface{}{}

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

func TestJsValueAuthorizedNil(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Metadata = nil

	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
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
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, true, result["has_authorized"], "authorized property should exist")
	assert.Equal(t, "object", result["authorized_type"], "authorized should be an object")
	assert.Equal(t, true, result["metadata_is_object"], "authorized should be an object (not null)")

	assert.Equal(t, true, result["has_metadata"], "metadata property should exist")
	assert.Equal(t, "object", result["metadata_type"], "metadata should be an object")
	assert.Equal(t, true, result["metadata_is_object"], "metadata should be an object")
	assert.Equal(t, true, result["metadata_is_empty"], "metadata should be empty object when not set")
}
