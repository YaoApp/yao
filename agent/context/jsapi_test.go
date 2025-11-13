package context

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
	"rogchap.com/v8go"
)

// TestJsValue test the JsValue function
func TestJsValue(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "ChatID-123456",
		AssistantID: "AssistantID-1234",
		Sid:         "Sid-1234",
	}

	v8.RegisterFunction("testContextJsvalue", testContextJsvalueEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test(cxt) {
			return testContextJsvalue(cxt)
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	assert.Equal(t, "ChatID-123456", res)
	assert.Equal(t, 0, len(objects))
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

	chatID, err := ctx.Get("ChatID")
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
				sid := fmt.Sprintf("Sid-%d-%d", routineID, j)

				cxt := &Context{
					ChatID:      chatID,
					AssistantID: assistantID,
					Sid:         sid,
				}

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
	assert.Equal(t, 0, len(objects), "All objects should be cleaned up")
}

// TestJsValueRegistrationAndCleanup test the object registration and cleanup mechanism
func TestJsValueRegistrationAndCleanup(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Clear objects map before test
	objectsMutex.Lock()
	objects = map[string]*Context{}
	objectsMutex.Unlock()

	v8.RegisterFunction("testContextRegistration", testContextRegistrationEmbed)

	// Create multiple contexts and verify registration
	contextCount := 5
	for i := 0; i < contextCount; i++ {
		cxt := &Context{
			ChatID:      fmt.Sprintf("ChatID-%d", i),
			AssistantID: fmt.Sprintf("AssistantID-%d", i),
			Sid:         fmt.Sprintf("Sid-%d", i),
		}

		_, err := v8.Call(v8.CallOptions{}, `
			function test(cxt) {
				return testContextRegistration(cxt)
			}`, cxt)

		if err != nil {
			t.Fatalf("Call %d failed: %v", i, err)
		}
	}

	// All objects should be cleaned up after v8.Call completes
	assert.Equal(t, 0, len(objects), "All objects should be cleaned up after execution")
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

	// Verify the object has __id field
	id, err := ctx.Get("__id")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if !id.IsString() {
		return bridge.JsException(info.Context(), fmt.Errorf("__id should be a string"))
	}

	// Verify the object has __release function
	release, err := ctx.Get("__release")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if !release.IsFunction() {
		return bridge.JsException(info.Context(), fmt.Errorf("__release should be a function"))
	}

	// Verify the object is registered
	objectsMutex.Lock()
	idStr := id.String()
	_, exists := objects[idStr]
	objectsMutex.Unlock()

	if !exists {
		return bridge.JsException(info.Context(), fmt.Errorf("object %s not registered", idStr))
	}

	val, err := v8go.NewValue(info.Context().Isolate(), true)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	return val
}
