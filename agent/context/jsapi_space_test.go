package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/plan"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestSpaceSetAndGet tests ctx.space.Set and ctx.space.Get
func TestSpaceSetAndGet(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set various types of values
				ctx.space.Set("string_key", "hello world");
				ctx.space.Set("number_key", 42);
				ctx.space.Set("boolean_key", true);
				ctx.space.Set("object_key", { name: "test", value: 123 });
				ctx.space.Set("array_key", [1, 2, 3, 4, 5]);
				
				// Get values back
				const str = ctx.space.Get("string_key");
				const num = ctx.space.Get("number_key");
				const bool = ctx.space.Get("boolean_key");
				const obj = ctx.space.Get("object_key");
				const arr = ctx.space.Get("array_key");
				
				// Verify values
				if (str !== "hello world") throw new Error("String mismatch");
				if (num !== 42) throw new Error("Number mismatch");
				if (bool !== true) throw new Error("Boolean mismatch");
				if (obj.name !== "test" || obj.value !== 123) throw new Error("Object mismatch");
				if (arr.length !== 5 || arr[0] !== 1) throw new Error("Array mismatch");
				
				return { 
					success: true,
					str: str,
					num: num,
					bool: bool,
					obj: obj,
					arr: arr
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, true, result["success"], "Space Set/Get should succeed")
	assert.Equal(t, "hello world", result["str"], "String should match")
	assert.Equal(t, float64(42), result["num"], "Number should match")
	assert.Equal(t, true, result["bool"], "Boolean should match")
}

// TestSpaceGetNonExistentKey tests getting a non-existent key
func TestSpaceGetNonExistentKey(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Get non-existent key should return null/undefined
				const value = ctx.space.Get("non_existent_key");
				
				return { 
					success: true,
					value: value,
					is_null: value === null,
					is_undefined: value === undefined
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["success"], "Get non-existent key should succeed")
	// JavaScript null is returned as nil in Go
	assert.True(t, result["is_null"].(bool) || result["is_undefined"].(bool), "Non-existent key should return null or undefined")
}

// TestSpaceDelete tests ctx.space.Delete
func TestSpaceDelete(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set a value
				ctx.space.Set("delete_me", "temporary value");
				
				// Verify it exists
				const before = ctx.space.Get("delete_me");
				if (before !== "temporary value") throw new Error("Value not set correctly");
				
				// Delete it
				ctx.space.Delete("delete_me");
				
				// Verify it's gone
				const after = ctx.space.Get("delete_me");
				
				return { 
					success: true,
					before: before,
					after: after,
					is_deleted: after === null || after === undefined
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, true, result["success"], "Space Delete should succeed")
	assert.Equal(t, "temporary value", result["before"], "Value should exist before delete")
	assert.Equal(t, true, result["is_deleted"], "Value should be deleted")
}

// TestSpaceDeleteNonExistentKey tests deleting a non-existent key
func TestSpaceDeleteNonExistentKey(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Delete non-existent key should not throw error
				ctx.space.Delete("non_existent_key");
				
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["success"], "Delete non-existent key should not throw error")
}

// TestSpaceWithNamespace tests using Space with namespace prefixes (like agent IDs)
func TestSpaceWithNamespace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Simulate namespace pattern used in voucher assistant
				const agentID = "workers.voucher";
				
				// Set files_info with namespace
				const filesInfo = [
					{
						file_id: "abc123",
						filename: "test.png",
						content_type: "image/png",
						file_type: "image",
						source: "uploader"
					}
				];
				ctx.space.Set(agentID + ":files_info", filesInfo);
				
				// Set current_file with namespace
				const currentFile = {
					file_id: "abc123",
					filename: "test.png",
					content_type: "image/png"
				};
				ctx.space.Set(agentID + ":current_file", currentFile);
				
				// Read back with namespace
				const retrievedFiles = ctx.space.Get(agentID + ":files_info");
				const retrievedCurrent = ctx.space.Get(agentID + ":current_file");
				
				// Verify
				if (!Array.isArray(retrievedFiles)) throw new Error("files_info should be array");
				if (retrievedFiles.length !== 1) throw new Error("files_info length mismatch");
				if (retrievedFiles[0].file_id !== "abc123") throw new Error("file_id mismatch");
				if (retrievedCurrent.filename !== "test.png") throw new Error("filename mismatch");
				
				return { 
					success: true,
					files_count: retrievedFiles.length,
					file_id: retrievedFiles[0].file_id,
					current_filename: retrievedCurrent.filename
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, true, result["success"], "Namespace operations should succeed")
	assert.Equal(t, float64(1), result["files_count"], "Should have 1 file")
	assert.Equal(t, "abc123", result["file_id"], "File ID should match")
	assert.Equal(t, "test.png", result["current_filename"], "Filename should match")
}

// TestSpaceComplexData tests Space with complex nested data structures
func TestSpaceComplexData(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Complex nested structure
				const complexData = {
					metadata: {
						assistant_id: "tests.vision-helper",
						has_files_info: true,
						files_count: 2
					},
					files_info: [
						{
							file_id: "file1",
							filename: "image1.png",
							content_type: "image/png",
							metadata: {
								size: 1024,
								created: Date.now()
							}
						},
						{
							file_id: "file2",
							filename: "image2.jpg",
							content_type: "image/jpeg",
							metadata: {
								size: 2048,
								created: Date.now()
							}
						}
					],
					tags: ["vision", "test", "multi-file"]
				};
				
				ctx.space.Set("complex_data", complexData);
				
				// Retrieve and verify
				const retrieved = ctx.space.Get("complex_data");
				
				if (!retrieved) throw new Error("Data not retrieved");
				if (!retrieved.metadata) throw new Error("Metadata missing");
				if (retrieved.metadata.files_count !== 2) throw new Error("Files count mismatch");
				if (!Array.isArray(retrieved.files_info)) throw new Error("files_info not array");
				if (retrieved.files_info.length !== 2) throw new Error("files_info length mismatch");
				if (!Array.isArray(retrieved.tags)) throw new Error("tags not array");
				if (retrieved.tags[0] !== "vision") throw new Error("tags mismatch");
				
				return { 
					success: true,
					files_count: retrieved.files_info.length,
					first_file_id: retrieved.files_info[0].file_id,
					second_filename: retrieved.files_info[1].filename,
					tags: retrieved.tags
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, true, result["success"], "Complex data operations should succeed")
	assert.Equal(t, float64(2), result["files_count"], "Should have 2 files")
	assert.Equal(t, "file1", result["first_file_id"], "First file ID should match")
	assert.Equal(t, "image2.jpg", result["second_filename"], "Second filename should match")
}

// TestSpaceOverwrite tests overwriting existing values
func TestSpaceOverwrite(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set initial value
				ctx.space.Set("counter", 1);
				const first = ctx.space.Get("counter");
				
				// Overwrite with new value
				ctx.space.Set("counter", 2);
				const second = ctx.space.Get("counter");
				
				// Overwrite again
				ctx.space.Set("counter", 3);
				const third = ctx.space.Get("counter");
				
				return { 
					success: true,
					first: first,
					second: second,
					third: third
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, true, result["success"], "Overwrite operations should succeed")
	assert.Equal(t, float64(1), result["first"], "First value should be 1")
	assert.Equal(t, float64(2), result["second"], "Second value should be 2")
	assert.Equal(t, float64(3), result["third"], "Third value should be 3")
}

// TestSpaceNoSpace tests behavior when Space is nil
func TestSpaceNoSpace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       nil, // No Space
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// ctx.space should be undefined when Space is nil
				const hasSpace = ctx.space !== undefined && ctx.space !== null;
				
				return { 
					success: true,
					has_space: hasSpace
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["success"], "Should handle nil Space gracefully")
	assert.Equal(t, false, result["has_space"], "Should not have space when Space is nil")
}

// TestSpaceErrorHandling tests error handling in Space methods
func TestSpaceErrorHandling(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	// Test Set without key
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set without proper arguments should throw
				ctx.space.Set();
				return { success: false, error: "Should have thrown" };
			} catch (error) {
				return { success: true, caught_error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["success"], "Should catch Set error")

	// Test Get without key
	res, err = v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Get without key should throw
				ctx.space.Get();
				return { success: false, error: "Should have thrown" };
			} catch (error) {
				return { success: true, caught_error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok = res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["success"], "Should catch Get error")

	// Test Delete without key
	res, err = v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Delete without key should throw
				ctx.space.Delete();
				return { success: false, error: "Should have thrown" };
			} catch (error) {
				return { success: true, caught_error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok = res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["success"], "Should catch Delete error")
}

// TestSpaceGetDel tests ctx.space.GetDel method
func TestSpaceGetDel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set a one-time use value
				ctx.space.Set("one_time_token", "secret_token_12345");
				
				// Verify it exists before GetDel
				const before = ctx.space.Get("one_time_token");
				if (before !== "secret_token_12345") throw new Error("Value not set");
				
				// Use GetDel - should get value and delete automatically
				const value = ctx.space.GetDel("one_time_token");
				
				// Verify value was retrieved
				if (value !== "secret_token_12345") throw new Error("GetDel returned wrong value");
				
				// Verify key was deleted
				const after = ctx.space.Get("one_time_token");
				if (after !== null && after !== undefined) throw new Error("Key should be deleted after GetDel");
				
				return { 
					success: true,
					value: value,
					is_deleted: after === null || after === undefined
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, true, result["success"], "GetDel should succeed")
	assert.Equal(t, "secret_token_12345", result["value"], "GetDel should return correct value")
	assert.Equal(t, true, result["is_deleted"], "Key should be deleted after GetDel")
}

// TestSpaceGetDelNonExistentKey tests GetDel on non-existent key
func TestSpaceGetDelNonExistentKey(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// GetDel on non-existent key should return null/undefined
				const value = ctx.space.GetDel("non_existent_key");
				
				return { 
					success: true,
					value: value,
					is_null: value === null || value === undefined
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["success"], "GetDel on non-existent key should not throw")
	assert.Equal(t, true, result["is_null"], "GetDel on non-existent key should return null")
}

// TestSpaceGetDelComplexData tests GetDel with complex data structures
func TestSpaceGetDelComplexData(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set complex file info data (like voucher assistant use case)
				const filesInfo = [
					{
						file_id: "file123",
						filename: "invoice.pdf",
						content_type: "application/pdf",
						file_type: "pdf",
						source: "uploader",
						uploader_name: "__yao.attachment"
					},
					{
						file_id: "file456",
						filename: "receipt.png",
						content_type: "image/png",
						file_type: "image",
						source: "uploader",
						uploader_name: "__yao.attachment"
					}
				];
				
				ctx.space.Set("workers.voucher:files_info", filesInfo);
				
				// Use GetDel to retrieve and clean up
				const retrieved = ctx.space.GetDel("workers.voucher:files_info");
				
				// Verify data integrity
				if (!Array.isArray(retrieved)) throw new Error("Should be array");
				if (retrieved.length !== 2) throw new Error("Length mismatch");
				if (retrieved[0].file_id !== "file123") throw new Error("First file_id mismatch");
				if (retrieved[1].filename !== "receipt.png") throw new Error("Second filename mismatch");
				
				// Verify it's deleted
				const after = ctx.space.Get("workers.voucher:files_info");
				if (after !== null && after !== undefined) throw new Error("Should be deleted");
				
				return { 
					success: true,
					files_count: retrieved.length,
					first_file_id: retrieved[0].file_id,
					is_deleted: after === null || after === undefined
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, true, result["success"], "GetDel with complex data should succeed")
	assert.Equal(t, float64(2), result["files_count"], "Should have 2 files")
	assert.Equal(t, "file123", result["first_file_id"], "File ID should match")
	assert.Equal(t, true, result["is_deleted"], "Should be deleted after GetDel")
}

// TestSpaceGetDelMultipleCalls tests that GetDel only works once
func TestSpaceGetDelMultipleCalls(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Space:       plan.NewMemorySharedSpace(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set a value
				ctx.space.Set("single_use", "use_me_once");
				
				// First GetDel should work
				const first = ctx.space.GetDel("single_use");
				if (first !== "use_me_once") throw new Error("First GetDel failed");
				
				// Second GetDel should return null (already deleted)
				const second = ctx.space.GetDel("single_use");
				if (second !== null && second !== undefined) throw new Error("Second GetDel should return null");
				
				// Third GetDel should also return null
				const third = ctx.space.GetDel("single_use");
				if (third !== null && third !== undefined) throw new Error("Third GetDel should return null");
				
				return { 
					success: true,
					first: first,
					second_is_null: second === null || second === undefined,
					third_is_null: third === null || third === undefined
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, true, result["success"], "Multiple GetDel calls should work correctly")
	assert.Equal(t, "use_me_once", result["first"], "First GetDel should return value")
	assert.Equal(t, true, result["second_is_null"], "Second GetDel should return null")
	assert.Equal(t, true, result["third_is_null"], "Third GetDel should return null")
}
