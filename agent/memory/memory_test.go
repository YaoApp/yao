package memory_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/memory"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestMemoryNew(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create memory with default stores
	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)
	require.NotNil(t, mem)

	// Verify all namespaces are initialized
	assert.NotNil(t, mem.User)
	assert.NotNil(t, mem.Team)
	assert.NotNil(t, mem.Chat)
	assert.NotNil(t, mem.Context)

	// Verify IDs
	assert.Equal(t, "user1", mem.UserID)
	assert.Equal(t, "team1", mem.TeamID)
	assert.Equal(t, "chat1", mem.ChatID)
	assert.Equal(t, "ctx1", mem.ContextID)
}

func TestMemoryPartialIDs(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create memory with only user and chat
	mem, err := memory.New(nil, "user1", "", "chat1", "")
	require.NoError(t, err)
	require.NotNil(t, mem)

	// Only user and chat namespaces should be initialized
	assert.NotNil(t, mem.User)
	assert.Nil(t, mem.Team)
	assert.NotNil(t, mem.Chat)
	assert.Nil(t, mem.Context)
}

func TestNamespaceBasicOperations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	// Test User namespace
	t.Run("User namespace", func(t *testing.T) {
		ns := mem.GetUser()
		require.NotNil(t, ns)

		// Set and Get
		err := ns.Set("name", "John", 0)
		require.NoError(t, err)

		val, ok := ns.Get("name")
		assert.True(t, ok)
		assert.Equal(t, "John", val)

		// Has
		assert.True(t, ns.Has("name"))
		assert.False(t, ns.Has("nonexistent"))

		// Del
		err = ns.Del("name")
		require.NoError(t, err)
		assert.False(t, ns.Has("name"))
	})

	// Test Team namespace
	t.Run("Team namespace", func(t *testing.T) {
		ns := mem.GetTeam()
		require.NotNil(t, ns)

		err := ns.Set("setting", "value", 0)
		require.NoError(t, err)

		val, ok := ns.Get("setting")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	// Test Chat namespace
	t.Run("Chat namespace", func(t *testing.T) {
		ns := mem.GetChat()
		require.NotNil(t, ns)

		err := ns.Set("topic", "AI", 0)
		require.NoError(t, err)

		val, ok := ns.Get("topic")
		assert.True(t, ok)
		assert.Equal(t, "AI", val)
	})

	// Test Context namespace
	t.Run("Context namespace", func(t *testing.T) {
		ns := mem.GetContext()
		require.NotNil(t, ns)

		err := ns.Set("temp", "data", 0)
		require.NoError(t, err)

		val, ok := ns.Get("temp")
		assert.True(t, ok)
		assert.Equal(t, "data", val)
	})
}

func TestNamespaceIsolation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	t.Run("User isolation", func(t *testing.T) {
		// Create two memory instances with different user IDs
		mem1, err := memory.New(nil, "user1", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "user2", "", "", "")
		require.NoError(t, err)

		// Set value in user1's namespace
		err = mem1.GetUser().Set("key", "user1_value", 0)
		require.NoError(t, err)

		// Set value in user2's namespace
		err = mem2.GetUser().Set("key", "user2_value", 0)
		require.NoError(t, err)

		// Verify isolation - each user sees their own value
		val1, ok := mem1.GetUser().Get("key")
		assert.True(t, ok)
		assert.Equal(t, "user1_value", val1)

		val2, ok := mem2.GetUser().Get("key")
		assert.True(t, ok)
		assert.Equal(t, "user2_value", val2)

		// Delete from user1 should not affect user2
		err = mem1.GetUser().Del("key")
		require.NoError(t, err)

		_, ok = mem1.GetUser().Get("key")
		assert.False(t, ok, "user1's key should be deleted")

		val2, ok = mem2.GetUser().Get("key")
		assert.True(t, ok, "user2's key should still exist")
		assert.Equal(t, "user2_value", val2)

		// Clear user1 should not affect user2
		mem1.GetUser().Clear()
		val2, ok = mem2.GetUser().Get("key")
		assert.True(t, ok, "user2's key should still exist after user1 clear")
		assert.Equal(t, "user2_value", val2)
	})

	t.Run("Team isolation", func(t *testing.T) {
		memA, err := memory.New(nil, "", "teamA", "", "")
		require.NoError(t, err)

		memB, err := memory.New(nil, "", "teamB", "", "")
		require.NoError(t, err)

		// Set same key in different teams
		memA.GetTeam().Set("config", "teamA_config", 0)
		memB.GetTeam().Set("config", "teamB_config", 0)

		// Verify isolation
		valA, ok := memA.GetTeam().Get("config")
		assert.True(t, ok)
		assert.Equal(t, "teamA_config", valA)

		valB, ok := memB.GetTeam().Get("config")
		assert.True(t, ok)
		assert.Equal(t, "teamB_config", valB)
	})

	t.Run("Chat isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "", "", "chat1", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "", "", "chat2", "")
		require.NoError(t, err)

		// Set same key in different chats
		mem1.GetChat().Set("topic", "chat1_topic", 0)
		mem2.GetChat().Set("topic", "chat2_topic", 0)

		// Verify isolation
		val1, ok := mem1.GetChat().Get("topic")
		assert.True(t, ok)
		assert.Equal(t, "chat1_topic", val1)

		val2, ok := mem2.GetChat().Get("topic")
		assert.True(t, ok)
		assert.Equal(t, "chat2_topic", val2)
	})

	t.Run("Context isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "", "", "", "ctx1")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "", "", "", "ctx2")
		require.NoError(t, err)

		// Set same key in different contexts
		mem1.GetContext().Set("temp", "ctx1_temp", 0)
		mem2.GetContext().Set("temp", "ctx2_temp", 0)

		// Verify isolation
		val1, ok := mem1.GetContext().Get("temp")
		assert.True(t, ok)
		assert.Equal(t, "ctx1_temp", val1)

		val2, ok := mem2.GetContext().Get("temp")
		assert.True(t, ok)
		assert.Equal(t, "ctx2_temp", val2)
	})

	t.Run("Keys and Len isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "userA", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "userB", "", "", "")
		require.NoError(t, err)

		// Clear first
		mem1.GetUser().Clear()
		mem2.GetUser().Clear()

		// Set keys in userA
		mem1.GetUser().Set("a", 1, 0)
		mem1.GetUser().Set("b", 2, 0)
		mem1.GetUser().Set("c", 3, 0)

		// Set keys in userB
		mem2.GetUser().Set("x", 10, 0)
		mem2.GetUser().Set("y", 20, 0)

		// Verify Keys isolation
		keys1 := mem1.GetUser().Keys()
		assert.Equal(t, 3, len(keys1), "userA should have 3 keys")

		keys2 := mem2.GetUser().Keys()
		assert.Equal(t, 2, len(keys2), "userB should have 2 keys")

		// Verify Len isolation
		assert.Equal(t, 3, mem1.GetUser().Len(), "userA Len should be 3")
		assert.Equal(t, 2, mem2.GetUser().Len(), "userB Len should be 2")

		// Keys should not contain prefix
		for _, k := range keys1 {
			assert.NotContains(t, k, "user:", "Key should not contain prefix")
		}
	})

	t.Run("Incr/Decr isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "userX", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "userY", "", "", "")
		require.NoError(t, err)

		// Incr counter in userX
		val1, err := mem1.GetUser().Incr("counter", 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), val1)

		// Incr counter in userY
		val2, err := mem2.GetUser().Incr("counter", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), val2)

		// Incr again - should be independent
		val1, err = mem1.GetUser().Incr("counter", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(11), val1)

		val2, err = mem2.GetUser().Incr("counter", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(6), val2)
	})

	t.Run("List operations isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "listUser1", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "listUser2", "", "", "")
		require.NoError(t, err)

		// Push to user1's list
		err = mem1.GetUser().Push("items", "a", "b", "c")
		require.NoError(t, err)

		// Push to user2's list
		err = mem2.GetUser().Push("items", "x", "y")
		require.NoError(t, err)

		// Verify isolation
		assert.Equal(t, 3, mem1.GetUser().ArrayLen("items"))
		assert.Equal(t, 2, mem2.GetUser().ArrayLen("items"))

		all1, _ := mem1.GetUser().ArrayAll("items")
		all2, _ := mem2.GetUser().ArrayAll("items")

		assert.Equal(t, 3, len(all1))
		assert.Equal(t, 2, len(all2))

		// Pop from user1 should not affect user2
		mem1.GetUser().Pop("items", 1)
		assert.Equal(t, 2, mem1.GetUser().ArrayLen("items"))
		assert.Equal(t, 2, mem2.GetUser().ArrayLen("items"))
	})

	t.Run("Del pattern isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "patternUser1", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "patternUser2", "", "", "")
		require.NoError(t, err)

		// Set keys with pattern in both users
		mem1.GetUser().Set("file:1", "data1", 0)
		mem1.GetUser().Set("file:2", "data2", 0)
		mem1.GetUser().Set("other", "other1", 0)

		mem2.GetUser().Set("file:1", "data1", 0)
		mem2.GetUser().Set("file:2", "data2", 0)
		mem2.GetUser().Set("other", "other2", 0)

		// Delete pattern from user1
		err = mem1.GetUser().Del("file:*")
		require.NoError(t, err)

		// user1's file:* keys should be deleted
		assert.False(t, mem1.GetUser().Has("file:1"))
		assert.False(t, mem1.GetUser().Has("file:2"))
		assert.True(t, mem1.GetUser().Has("other"))

		// user2's keys should be unaffected
		assert.True(t, mem2.GetUser().Has("file:1"))
		assert.True(t, mem2.GetUser().Has("file:2"))
		assert.True(t, mem2.GetUser().Has("other"))
	})
}

func TestNamespaceIncrDecr(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()

	// Incr on non-existent key
	val, err := ns.Incr("counter", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Incr again
	val, err = ns.Incr("counter", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(6), val)

	// Decr
	val, err = ns.Decr("counter", 2)
	require.NoError(t, err)
	assert.Equal(t, int64(4), val)
}

func TestNamespaceListOperations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()

	// Push values
	err = ns.Push("list", "a", "b", "c")
	require.NoError(t, err)

	// ArrayLen
	assert.Equal(t, 3, ns.ArrayLen("list"))

	// ArrayAll
	all, err := ns.ArrayAll("list")
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// Pop from end
	val, err := ns.Pop("list", 1)
	require.NoError(t, err)
	assert.Equal(t, "c", val)

	// ArrayLen after pop
	assert.Equal(t, 2, ns.ArrayLen("list"))
}

func TestNamespaceSetOperations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()

	// AddToSet
	err = ns.AddToSet("tags", "go", "rust", "go") // "go" should only appear once
	require.NoError(t, err)

	all, err := ns.ArrayAll("tags")
	require.NoError(t, err)
	assert.Len(t, all, 2) // Only "go" and "rust"
}

func TestNamespaceTTL(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "", "", "", "ctx1")
	require.NoError(t, err)

	ns := mem.GetContext()

	// Set with short TTL
	err = ns.Set("temp", "value", 100*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	val, ok := ns.Get("temp")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, ok = ns.Get("temp")
	assert.False(t, ok)
}

func TestMemoryClear(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	// Set values in all namespaces
	mem.GetUser().Set("key", "user_value", 0)
	mem.GetTeam().Set("key", "team_value", 0)
	mem.GetChat().Set("key", "chat_value", 0)
	mem.GetContext().Set("key", "ctx_value", 0)

	// Clear all
	mem.Clear()

	// All should be empty
	_, ok := mem.GetUser().Get("key")
	assert.False(t, ok)
	_, ok = mem.GetTeam().Get("key")
	assert.False(t, ok)
	_, ok = mem.GetChat().Get("key")
	assert.False(t, ok)
	_, ok = mem.GetContext().Get("key")
	assert.False(t, ok)
}

func TestMemoryStats(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	// Set some values
	mem.GetUser().Set("k1", "v1", 0)
	mem.GetUser().Set("k2", "v2", 0)
	mem.GetTeam().Set("k1", "v1", 0)

	stats := mem.GetStats()
	require.NotNil(t, stats)

	assert.Equal(t, 2, stats.User.KeyCount)
	assert.Equal(t, 1, stats.Team.KeyCount)
	assert.Equal(t, 0, stats.Chat.KeyCount)
	assert.Equal(t, 0, stats.Context.KeyCount)
}

func TestManager(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mgr := memory.NewManagerWithDefaults()
	defer mgr.Close()

	// Get memory instance
	mem1, err := mgr.Memory("user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)
	require.NotNil(t, mem1)

	// Set a value
	err = mem1.GetUser().Set("key", "value", 0)
	require.NoError(t, err)

	// Get same memory instance again
	mem2, err := mgr.Memory("user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	// Should be the same instance (cached)
	val, ok := mem2.GetUser().Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

func TestGetSpace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	// Test GetSpace
	assert.NotNil(t, mem.GetSpace(memory.SpaceUser))
	assert.NotNil(t, mem.GetSpace(memory.SpaceTeam))
	assert.NotNil(t, mem.GetSpace(memory.SpaceChat))
	assert.NotNil(t, mem.GetSpace(memory.SpaceContext))

	// Invalid space
	assert.Nil(t, mem.GetSpace(memory.Space("invalid")))
}

func TestNamespaceGetMultiSetMulti(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()

	// SetMulti
	ns.SetMulti(map[string]interface{}{
		"a": 1,
		"b": 2,
		"c": 3,
	}, 0)

	// GetMulti
	result := ns.GetMulti([]string{"a", "b", "c"})
	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 2, result["b"])
	assert.Equal(t, 3, result["c"])

	// DelMulti
	ns.DelMulti([]string{"a", "b"})
	assert.False(t, ns.Has("a"))
	assert.False(t, ns.Has("b"))
	assert.True(t, ns.Has("c"))
}

func TestNamespaceGetDel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	mem, err := memory.New(nil, "user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()

	// Set a value
	err = ns.Set("key", "value", 0)
	require.NoError(t, err)

	// GetDel
	val, ok := ns.GetDel("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// Should be deleted
	_, ok = ns.Get("key")
	assert.False(t, ok)
}
