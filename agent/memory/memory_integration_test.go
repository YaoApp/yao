//go:build integration

package memory_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/memory"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMemoryNew(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)
	require.NotNil(t, mem)

	assert.NotNil(t, mem.User)
	assert.NotNil(t, mem.Team)
	assert.NotNil(t, mem.Chat)
	assert.NotNil(t, mem.Context)

	assert.Equal(t, "user1", mem.UserID)
	assert.Equal(t, "team1", mem.TeamID)
	assert.Equal(t, "chat1", mem.ChatID)
	assert.Equal(t, "ctx1", mem.ContextID)
}

func TestMemoryPartialIDs(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "user1", "", "chat1", "")
	require.NoError(t, err)
	require.NotNil(t, mem)

	assert.NotNil(t, mem.User)
	assert.Nil(t, mem.Team)
	assert.NotNil(t, mem.Chat)
	assert.Nil(t, mem.Context)
}

func TestNamespaceBasicOperations(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "user1", "team1", "chat1", "ctx1")
	require.NoError(t, err)

	t.Run("User namespace", func(t *testing.T) {
		ns := mem.GetUser()
		require.NotNil(t, ns)

		err := ns.Set("name", "John", 0)
		require.NoError(t, err)

		val, ok := ns.Get("name")
		assert.True(t, ok)
		assert.Equal(t, "John", val)

		assert.True(t, ns.Has("name"))
		assert.False(t, ns.Has("nonexistent"))

		err = ns.Del("name")
		require.NoError(t, err)
		assert.False(t, ns.Has("name"))
	})

	t.Run("Team namespace", func(t *testing.T) {
		ns := mem.GetTeam()
		require.NotNil(t, ns)

		err := ns.Set("setting", "value", 0)
		require.NoError(t, err)

		val, ok := ns.Get("setting")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("Chat namespace", func(t *testing.T) {
		ns := mem.GetChat()
		require.NotNil(t, ns)

		err := ns.Set("topic", "AI", 0)
		require.NoError(t, err)

		val, ok := ns.Get("topic")
		assert.True(t, ok)
		assert.Equal(t, "AI", val)
	})

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
	testprepare.PrepareSandbox(t)

	t.Run("User isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "iso-user1", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "iso-user2", "", "", "")
		require.NoError(t, err)

		err = mem1.GetUser().Set("key", "user1_value", 0)
		require.NoError(t, err)

		err = mem2.GetUser().Set("key", "user2_value", 0)
		require.NoError(t, err)

		val1, ok := mem1.GetUser().Get("key")
		assert.True(t, ok)
		assert.Equal(t, "user1_value", val1)

		val2, ok := mem2.GetUser().Get("key")
		assert.True(t, ok)
		assert.Equal(t, "user2_value", val2)

		err = mem1.GetUser().Del("key")
		require.NoError(t, err)

		_, ok = mem1.GetUser().Get("key")
		assert.False(t, ok)

		val2, ok = mem2.GetUser().Get("key")
		assert.True(t, ok)
		assert.Equal(t, "user2_value", val2)

		mem1.GetUser().Clear()
		val2, ok = mem2.GetUser().Get("key")
		assert.True(t, ok)
		assert.Equal(t, "user2_value", val2)
	})

	t.Run("Team isolation", func(t *testing.T) {
		memA, err := memory.New(nil, "", "iso-teamA", "", "")
		require.NoError(t, err)

		memB, err := memory.New(nil, "", "iso-teamB", "", "")
		require.NoError(t, err)

		memA.GetTeam().Set("config", "teamA_config", 0)
		memB.GetTeam().Set("config", "teamB_config", 0)

		valA, ok := memA.GetTeam().Get("config")
		assert.True(t, ok)
		assert.Equal(t, "teamA_config", valA)

		valB, ok := memB.GetTeam().Get("config")
		assert.True(t, ok)
		assert.Equal(t, "teamB_config", valB)
	})

	t.Run("Chat isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "", "", "iso-chat1", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "", "", "iso-chat2", "")
		require.NoError(t, err)

		mem1.GetChat().Set("topic", "chat1_topic", 0)
		mem2.GetChat().Set("topic", "chat2_topic", 0)

		val1, ok := mem1.GetChat().Get("topic")
		assert.True(t, ok)
		assert.Equal(t, "chat1_topic", val1)

		val2, ok := mem2.GetChat().Get("topic")
		assert.True(t, ok)
		assert.Equal(t, "chat2_topic", val2)
	})

	t.Run("Context isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "", "", "", "iso-ctx1")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "", "", "", "iso-ctx2")
		require.NoError(t, err)

		mem1.GetContext().Set("temp", "ctx1_temp", 0)
		mem2.GetContext().Set("temp", "ctx2_temp", 0)

		val1, ok := mem1.GetContext().Get("temp")
		assert.True(t, ok)
		assert.Equal(t, "ctx1_temp", val1)

		val2, ok := mem2.GetContext().Get("temp")
		assert.True(t, ok)
		assert.Equal(t, "ctx2_temp", val2)
	})

	t.Run("Keys and Len isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "iso-keysA", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "iso-keysB", "", "", "")
		require.NoError(t, err)

		mem1.GetUser().Clear()
		mem2.GetUser().Clear()

		mem1.GetUser().Set("a", 1, 0)
		mem1.GetUser().Set("b", 2, 0)
		mem1.GetUser().Set("c", 3, 0)

		mem2.GetUser().Set("x", 10, 0)
		mem2.GetUser().Set("y", 20, 0)

		keys1 := mem1.GetUser().Keys()
		assert.Equal(t, 3, len(keys1))

		keys2 := mem2.GetUser().Keys()
		assert.Equal(t, 2, len(keys2))

		assert.Equal(t, 3, mem1.GetUser().Len())
		assert.Equal(t, 2, mem2.GetUser().Len())

		for _, k := range keys1 {
			assert.NotContains(t, k, "user:")
		}
	})

	t.Run("Incr/Decr isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "iso-incrX", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "iso-incrY", "", "", "")
		require.NoError(t, err)

		mem1.GetUser().Del("counter")
		mem2.GetUser().Del("counter")

		val1, err := mem1.GetUser().Incr("counter", 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), val1)

		val2, err := mem2.GetUser().Incr("counter", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), val2)

		val1, err = mem1.GetUser().Incr("counter", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(11), val1)

		val2, err = mem2.GetUser().Incr("counter", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(6), val2)
	})

	t.Run("List ops isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "iso-list1", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "iso-list2", "", "", "")
		require.NoError(t, err)

		mem1.GetUser().Del("items")
		mem2.GetUser().Del("items")

		err = mem1.GetUser().Push("items", "a", "b", "c")
		require.NoError(t, err)

		err = mem2.GetUser().Push("items", "x", "y")
		require.NoError(t, err)

		assert.Equal(t, 3, mem1.GetUser().ArrayLen("items"))
		assert.Equal(t, 2, mem2.GetUser().ArrayLen("items"))

		all1, _ := mem1.GetUser().ArrayAll("items")
		all2, _ := mem2.GetUser().ArrayAll("items")
		assert.Equal(t, 3, len(all1))
		assert.Equal(t, 2, len(all2))

		mem1.GetUser().Pop("items", 1)
		assert.Equal(t, 2, mem1.GetUser().ArrayLen("items"))
		assert.Equal(t, 2, mem2.GetUser().ArrayLen("items"))
	})

	t.Run("Del pattern isolation", func(t *testing.T) {
		mem1, err := memory.New(nil, "iso-pat1", "", "", "")
		require.NoError(t, err)

		mem2, err := memory.New(nil, "iso-pat2", "", "", "")
		require.NoError(t, err)

		mem1.GetUser().Set("file:1", "data1", 0)
		mem1.GetUser().Set("file:2", "data2", 0)
		mem1.GetUser().Set("other", "other1", 0)

		mem2.GetUser().Set("file:1", "data1", 0)
		mem2.GetUser().Set("file:2", "data2", 0)
		mem2.GetUser().Set("other", "other2", 0)

		err = mem1.GetUser().Del("file:*")
		require.NoError(t, err)

		assert.False(t, mem1.GetUser().Has("file:1"))
		assert.False(t, mem1.GetUser().Has("file:2"))
		assert.True(t, mem1.GetUser().Has("other"))

		assert.True(t, mem2.GetUser().Has("file:1"))
		assert.True(t, mem2.GetUser().Has("file:2"))
		assert.True(t, mem2.GetUser().Has("other"))
	})
}

func TestNamespaceIncrDecr(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "incr-user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()
	ns.Del("counter")

	val, err := ns.Incr("counter", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = ns.Incr("counter", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(6), val)

	val, err = ns.Decr("counter", 2)
	require.NoError(t, err)
	assert.Equal(t, int64(4), val)
}

func TestNamespaceListOperations(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "list-user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()
	ns.Del("list")

	err = ns.Push("list", "a", "b", "c")
	require.NoError(t, err)

	assert.Equal(t, 3, ns.ArrayLen("list"))

	all, err := ns.ArrayAll("list")
	require.NoError(t, err)
	assert.Len(t, all, 3)

	val, err := ns.Pop("list", 1)
	require.NoError(t, err)
	assert.Equal(t, "c", val)

	assert.Equal(t, 2, ns.ArrayLen("list"))
}

func TestNamespaceSetOperations(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "set-user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()

	err = ns.AddToSet("tags", "go", "rust", "go")
	require.NoError(t, err)

	all, err := ns.ArrayAll("tags")
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestNamespaceTTL(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "", "", "", "ttl-ctx1")
	require.NoError(t, err)

	ns := mem.GetContext()

	err = ns.Set("temp", "value", 100*time.Millisecond)
	require.NoError(t, err)

	val, ok := ns.Get("temp")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	time.Sleep(150 * time.Millisecond)

	_, ok = ns.Get("temp")
	assert.False(t, ok)
}

func TestMemoryClear(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "clear-user1", "clear-team1", "clear-chat1", "clear-ctx1")
	require.NoError(t, err)

	mem.GetUser().Set("key", "user_value", 0)
	mem.GetTeam().Set("key", "team_value", 0)
	mem.GetChat().Set("key", "chat_value", 0)
	mem.GetContext().Set("key", "ctx_value", 0)

	mem.Clear()

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
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "stats-user1", "stats-team1", "stats-chat1", "stats-ctx1")
	require.NoError(t, err)

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
	testprepare.PrepareSandbox(t)

	mgr := memory.NewManagerWithDefaults()
	defer mgr.Close()

	mem1, err := mgr.Memory("mgr-user1", "mgr-team1", "mgr-chat1", "mgr-ctx1")
	require.NoError(t, err)
	require.NotNil(t, mem1)

	err = mem1.GetUser().Set("key", "value", 0)
	require.NoError(t, err)

	mem2, err := mgr.Memory("mgr-user1", "mgr-team1", "mgr-chat1", "mgr-ctx1")
	require.NoError(t, err)

	val, ok := mem2.GetUser().Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

func TestGetSpace(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "space-user1", "space-team1", "space-chat1", "space-ctx1")
	require.NoError(t, err)

	assert.NotNil(t, mem.GetSpace(memory.SpaceUser))
	assert.NotNil(t, mem.GetSpace(memory.SpaceTeam))
	assert.NotNil(t, mem.GetSpace(memory.SpaceChat))
	assert.NotNil(t, mem.GetSpace(memory.SpaceContext))

	assert.Nil(t, mem.GetSpace(memory.Space("invalid")))
}

func TestNamespaceGetMultiSetMulti(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "multi-user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()

	ns.SetMulti(map[string]interface{}{
		"a": 1,
		"b": 2,
		"c": 3,
	}, 0)

	result := ns.GetMulti([]string{"a", "b", "c"})
	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 2, result["b"])
	assert.Equal(t, 3, result["c"])

	ns.DelMulti([]string{"a", "b"})
	assert.False(t, ns.Has("a"))
	assert.False(t, ns.Has("b"))
	assert.True(t, ns.Has("c"))
}

func TestNamespaceGetDel(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "getdel-user1", "", "", "")
	require.NoError(t, err)

	ns := mem.GetUser()

	err = ns.Set("key", "value", 0)
	require.NoError(t, err)

	val, ok := ns.GetDel("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	_, ok = ns.Get("key")
	assert.False(t, ok)
}

func TestMemoryFork(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mem, err := memory.New(nil, "fork-user1", "fork-team1", "fork-chat1", "fork-ctx1")
	require.NoError(t, err)

	mem.GetUser().Set("ukey", "uval", 0)
	mem.GetTeam().Set("tkey", "tval", 0)
	mem.GetChat().Set("ckey", "cval", 0)
	mem.GetContext().Set("xkey", "xval", 0)

	forked, err := mem.Fork("new-ctx")
	require.NoError(t, err)
	require.NotNil(t, forked)

	assert.Equal(t, "fork-user1", forked.UserID)
	assert.Equal(t, "fork-team1", forked.TeamID)
	assert.Equal(t, "fork-chat1", forked.ChatID)
	assert.Equal(t, "new-ctx", forked.ContextID)

	val, ok := forked.GetUser().Get("ukey")
	assert.True(t, ok)
	assert.Equal(t, "uval", val)

	val, ok = forked.GetTeam().Get("tkey")
	assert.True(t, ok)
	assert.Equal(t, "tval", val)

	val, ok = forked.GetChat().Get("ckey")
	assert.True(t, ok)
	assert.Equal(t, "cval", val)

	_, ok = forked.GetContext().Get("xkey")
	assert.False(t, ok)
}
