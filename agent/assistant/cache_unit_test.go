//go:build unit

package assistant_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
)

func TestCacheBasic(t *testing.T) {
	cache := assistant.NewCache(3)
	assert.Equal(t, 0, cache.Len())

	a1 := &assistant.Assistant{}
	a1.ID = "test-1"
	a2 := &assistant.Assistant{}
	a2.ID = "test-2"

	cache.Put(a1)
	cache.Put(a2)
	assert.Equal(t, 2, cache.Len())

	got, ok := cache.Get("test-1")
	require.True(t, ok)
	assert.Equal(t, "test-1", got.ID)

	got2, ok := cache.Get("test-2")
	require.True(t, ok)
	assert.Equal(t, "test-2", got2.ID)

	_, ok = cache.Get("nonexistent")
	assert.False(t, ok)
}

func TestCacheLRU(t *testing.T) {
	cache := assistant.NewCache(2)

	a1 := &assistant.Assistant{}
	a1.ID = "a1"
	a2 := &assistant.Assistant{}
	a2.ID = "a2"
	a3 := &assistant.Assistant{}
	a3.ID = "a3"

	cache.Put(a1)
	cache.Put(a2)

	cache.Get("a1")

	cache.Put(a3)

	_, ok := cache.Get("a2")
	assert.False(t, ok, "a2 should be evicted (LRU)")

	_, ok = cache.Get("a1")
	assert.True(t, ok, "a1 should still be in cache")

	_, ok = cache.Get("a3")
	assert.True(t, ok, "a3 should be in cache")
}

func TestCacheRemove(t *testing.T) {
	cache := assistant.NewCache(3)

	a1 := &assistant.Assistant{}
	a1.ID = "r1"
	cache.Put(a1)
	assert.Equal(t, 1, cache.Len())

	cache.Remove("r1")
	assert.Equal(t, 0, cache.Len())

	_, ok := cache.Get("r1")
	assert.False(t, ok)

	cache.Remove("nonexistent")
	assert.Equal(t, 0, cache.Len())
}

func TestCacheClear(t *testing.T) {
	cache := assistant.NewCache(5)

	for i := 0; i < 3; i++ {
		a := &assistant.Assistant{}
		a.ID = string(rune('a' + i))
		cache.Put(a)
	}
	assert.Equal(t, 3, cache.Len())

	cache.Clear()
	assert.Equal(t, 0, cache.Len())
}

func TestCacheLRUEviction(t *testing.T) {
	cache := assistant.NewCache(2)

	a1 := &assistant.Assistant{}
	a1.ID = "e1"
	a2 := &assistant.Assistant{}
	a2.ID = "e2"
	a3 := &assistant.Assistant{}
	a3.ID = "e3"

	cache.Put(a1)
	cache.Put(a2)
	assert.Equal(t, 2, cache.Len())

	cache.Put(a3)
	assert.Equal(t, 2, cache.Len())

	_, ok := cache.Get("e1")
	assert.False(t, ok, "e1 should be evicted")

	_, ok = cache.Get("e2")
	assert.True(t, ok)
	_, ok = cache.Get("e3")
	assert.True(t, ok)
}

func TestCacheConcurrent(t *testing.T) {
	cache := assistant.NewCache(10)
	var wg sync.WaitGroup
	workers := 5
	iterations := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				a := &assistant.Assistant{}
				a.ID = string(rune('A'+workerID)) + "-" + string(rune('0'+j%10))
				cache.Put(a)
			}
		}(i)
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				id := string(rune('A'+workerID)) + "-" + string(rune('0'+j%10))
				cache.Get(id)
			}
		}(i)
	}

	wg.Wait()
	assert.True(t, cache.Len() >= 0)
	assert.True(t, cache.Len() <= 10)
}

func TestCacheNilInput(t *testing.T) {
	cache := assistant.NewCache(2)

	cache.Put(nil)
	assert.Equal(t, 0, cache.Len())

	empty := &assistant.Assistant{}
	cache.Put(empty)
	assert.Equal(t, 0, cache.Len())
}

func TestCacheAll(t *testing.T) {
	cache := assistant.NewCache(5)

	ids := []string{"all-1", "all-2", "all-3"}
	for _, id := range ids {
		a := &assistant.Assistant{}
		a.ID = id
		cache.Put(a)
	}

	all := cache.All()
	assert.Equal(t, 3, len(all))

	found := make(map[string]bool)
	for _, a := range all {
		found[a.ID] = true
	}
	for _, id := range ids {
		assert.True(t, found[id], "should contain %s", id)
	}
}

func TestCacheUpdate(t *testing.T) {
	cache := assistant.NewCache(3)

	a1 := &assistant.Assistant{}
	a1.ID = "u1"
	a1.Name = "original"
	cache.Put(a1)

	a1Updated := &assistant.Assistant{}
	a1Updated.ID = "u1"
	a1Updated.Name = "updated"
	cache.Put(a1Updated)

	assert.Equal(t, 1, cache.Len())
	got, ok := cache.Get("u1")
	require.True(t, ok)
	assert.Equal(t, "updated", got.Name)
}

func TestCacheClearExcept(t *testing.T) {
	cache := assistant.NewCache(5)

	for _, id := range []string{"keep-1", "remove-1", "keep-2", "remove-2"} {
		a := &assistant.Assistant{}
		a.ID = id
		cache.Put(a)
	}
	assert.Equal(t, 4, cache.Len())

	cache.ClearExcept(func(id string) bool {
		return id == "keep-1" || id == "keep-2"
	})

	assert.Equal(t, 2, cache.Len())
	_, ok := cache.Get("keep-1")
	assert.True(t, ok)
	_, ok = cache.Get("keep-2")
	assert.True(t, ok)
	_, ok = cache.Get("remove-1")
	assert.False(t, ok)
}
