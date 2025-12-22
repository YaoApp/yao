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

	// Verify isolation
	val1, ok := mem1.GetUser().Get("key")
	assert.True(t, ok)
	assert.Equal(t, "user1_value", val1)

	val2, ok := mem2.GetUser().Get("key")
	assert.True(t, ok)
	assert.Equal(t, "user2_value", val2)
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
