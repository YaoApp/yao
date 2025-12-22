package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/memory"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

func TestMemoryUserNamespace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set values in user namespace
				ctx.memory.user.Set("name", "John");
				ctx.memory.user.Set("age", 30);
				ctx.memory.user.Set("active", true);
				
				// Get values back
				const name = ctx.memory.user.Get("name");
				const age = ctx.memory.user.Get("age");
				const active = ctx.memory.user.Get("active");
				
				// Verify
				if (name !== "John") throw new Error("Name mismatch");
				if (age !== 30) throw new Error("Age mismatch");
				if (active !== true) throw new Error("Active mismatch");
				
				return { 
					success: true,
					name: name,
					age: age,
					active: active
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "John", result["name"])
	assert.Equal(t, float64(30), result["age"])
	assert.Equal(t, true, result["active"])
}

func TestMemoryTeamNamespace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set team-wide settings
				ctx.memory.team.Set("settings", { theme: "dark", language: "en" });
				
				// Get back
				const settings = ctx.memory.team.Get("settings");
				
				if (settings.theme !== "dark") throw new Error("Theme mismatch");
				if (settings.language !== "en") throw new Error("Language mismatch");
				
				return { 
					success: true,
					settings: settings
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
}

func TestMemoryChatNamespace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set chat context
				ctx.memory.chat.Set("topic", "AI Discussion");
				ctx.memory.chat.Set("participants", ["Alice", "Bob"]);
				
				// Get back
				const topic = ctx.memory.chat.Get("topic");
				const participants = ctx.memory.chat.Get("participants");
				
				if (topic !== "AI Discussion") throw new Error("Topic mismatch");
				if (participants.length !== 2) throw new Error("Participants mismatch");
				
				return { 
					success: true,
					topic: topic,
					participants: participants
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "AI Discussion", result["topic"])
}

func TestMemoryContextNamespace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set temporary context data
				ctx.memory.context.Set("temp_result", { step: 1, data: "processing" });
				
				// Get back
				const result = ctx.memory.context.Get("temp_result");
				
				if (result.step !== 1) throw new Error("Step mismatch");
				if (result.data !== "processing") throw new Error("Data mismatch");
				
				return { 
					success: true,
					result: result
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
}

func TestMemoryHasAndDel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set a value
				ctx.memory.user.Set("key", "value");
				
				// Check Has
				const hasBefore = ctx.memory.user.Has("key");
				if (!hasBefore) throw new Error("Should have key before delete");
				
				// Delete
				ctx.memory.user.Del("key");
				
				// Check Has again
				const hasAfter = ctx.memory.user.Has("key");
				if (hasAfter) throw new Error("Should not have key after delete");
				
				return { 
					success: true,
					hasBefore: hasBefore,
					hasAfter: hasAfter
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
	assert.True(t, result["hasBefore"].(bool))
	assert.False(t, result["hasAfter"].(bool))
}

func TestMemoryIncrDecr(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Incr on non-existent key
				const v1 = ctx.memory.user.Incr("counter");
				if (v1 !== 1) throw new Error("First incr should be 1, got " + v1);
				
				// Incr with delta
				const v2 = ctx.memory.user.Incr("counter", 5);
				if (v2 !== 6) throw new Error("Second incr should be 6, got " + v2);
				
				// Decr
				const v3 = ctx.memory.user.Decr("counter", 2);
				if (v3 !== 4) throw new Error("Decr should be 4, got " + v3);
				
				return { 
					success: true,
					v1: v1,
					v2: v2,
					v3: v3
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
	assert.Equal(t, float64(1), result["v1"])
	assert.Equal(t, float64(6), result["v2"])
	assert.Equal(t, float64(4), result["v3"])
}

func TestMemoryKeysAndLen(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Use unique IDs to avoid data pollution from other tests
	mem, err := memory.New(nil, "user-keys-len", "team-keys-len", "chat-keys-len", "ctx-keys-len")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set multiple values
				ctx.memory.user.Set("a", 1);
				ctx.memory.user.Set("b", 2);
				ctx.memory.user.Set("c", 3);
				
				// Get keys
				const keys = ctx.memory.user.Keys();
				if (keys.length !== 3) throw new Error("Should have 3 keys, got " + keys.length);
				
				// Get len
				const len = ctx.memory.user.Len();
				if (len !== 3) throw new Error("Len should be 3, got " + len);
				
				return { 
					success: true,
					keys: keys,
					len: len
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}
	assert.Equal(t, float64(3), result["len"])
}

func TestMemoryClear(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set values
				ctx.memory.user.Set("a", 1);
				ctx.memory.user.Set("b", 2);
				
				// Clear
				ctx.memory.user.Clear();
				
				// Check len
				const len = ctx.memory.user.Len();
				if (len !== 0) throw new Error("Len should be 0 after clear, got " + len);
				
				return { 
					success: true,
					len: len
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
	assert.Equal(t, float64(0), result["len"])
}

func TestMemoryGetDel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	ctx := &context.Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Locale:      "en",
		Context:     stdContext.Background(),
		Memory:      mem,
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set a one-time value
				ctx.memory.user.Set("token", "secret123");
				
				// GetDel
				const value = ctx.memory.user.GetDel("token");
				if (value !== "secret123") throw new Error("Value mismatch");
				
				// Should be deleted
				const after = ctx.memory.user.Get("token");
				if (after !== null) throw new Error("Should be null after GetDel");
				
				return { 
					success: true,
					value: value,
					after: after
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "secret123", result["value"])
	assert.Nil(t, result["after"])
}

func TestMemoryIsolation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create two different memory instances
	mem1, err := memory.New(nil, "user1", "", "", "")
	require.NoError(t, err)

	mem2, err := memory.New(nil, "user2", "", "", "")
	require.NoError(t, err)

	ctx1 := &context.Context{
		ChatID:  "chat1",
		Context: stdContext.Background(),
		Memory:  mem1,
	}

	ctx2 := &context.Context{
		ChatID:  "chat2",
		Context: stdContext.Background(),
		Memory:  mem2,
	}

	// Set value in user1
	res1, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			ctx.memory.user.Set("key", "user1_value");
			return ctx.memory.user.Get("key");
		}`, ctx1)
	require.NoError(t, err)
	assert.Equal(t, "user1_value", res1)

	// Set value in user2
	res2, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			ctx.memory.user.Set("key", "user2_value");
			return ctx.memory.user.Get("key");
		}`, ctx2)
	require.NoError(t, err)
	assert.Equal(t, "user2_value", res2)

	// Verify user1 still has its own value
	res3, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			return ctx.memory.user.Get("key");
		}`, ctx1)
	require.NoError(t, err)
	assert.Equal(t, "user1_value", res3)
}

func TestMemoryNoMemory(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := &context.Context{
		ChatID:  "test-chat-id",
		Context: stdContext.Background(),
		Memory:  nil, // No memory
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const hasMemory = ctx.memory !== undefined && ctx.memory !== null;
				return { 
					success: true,
					hasMemory: hasMemory
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
	assert.False(t, result["hasMemory"].(bool))
}

func TestMemoryWithAuthorizedInfo(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Use context.New to create context with authorized info
	authorized := &types.AuthorizedInfo{
		UserID: "user123",
		TeamID: "team456",
	}

	ctx := context.New(stdContext.Background(), authorized, "chat789")
	defer ctx.Release()

	// Verify memory was created with correct IDs
	require.NotNil(t, ctx.Memory)
	require.NotNil(t, ctx.Memory.User)
	require.NotNil(t, ctx.Memory.Team)
	require.NotNil(t, ctx.Memory.Chat)
	require.NotNil(t, ctx.Memory.Context)

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Set values in different namespaces
				ctx.memory.user.Set("pref", "dark");
				ctx.memory.team.Set("setting", "shared");
				ctx.memory.chat.Set("topic", "test");
				ctx.memory.context.Set("temp", "data");
				
				return { 
					success: true,
					user: ctx.memory.user.Get("pref"),
					team: ctx.memory.team.Get("setting"),
					chat: ctx.memory.chat.Get("topic"),
					context: ctx.memory.context.Get("temp")
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "dark", result["user"])
	assert.Equal(t, "shared", result["team"])
	assert.Equal(t, "test", result["chat"])
	assert.Equal(t, "data", result["context"])
}
