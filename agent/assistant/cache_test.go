package assistant_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/testutils"
)

func TestCacheBasic(t *testing.T) {
	cache := assistant.NewCache(2)

	// Test empty cache
	assert.Equal(t, 0, cache.Len(), "Expected empty cache")

	// Create test assistants
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast1, err := assistant.Get("tests.mcpload")
	assert.NoError(t, err)

	ast2, err := assistant.Get("tests.create")
	assert.NoError(t, err)

	// Test adding items
	cache.Put(ast1)
	cache.Put(ast2)

	assert.Equal(t, 2, cache.Len(), "Expected cache length 2")

	// Test getting items
	cached1, exists := cache.Get("tests.mcpload")
	assert.True(t, exists, "Should find tests.mcpload")
	assert.Equal(t, "tests.mcpload", cached1.ID)

	cached2, exists := cache.Get("tests.create")
	assert.True(t, exists, "Should find tests.create")
	assert.Equal(t, "tests.create", cached2.ID)
}

func TestCacheLRU(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	cache := assistant.NewCache(2)

	ast1, _ := assistant.Get("tests.mcpload")
	ast2, _ := assistant.Get("tests.create")
	ast3, _ := assistant.Get("tests.next")

	// Add first two items
	cache.Put(ast1)
	cache.Put(ast2)

	// Access ast1 to make it most recently used
	cache.Get("tests.mcpload")

	// Add third item, should evict ast2
	cache.Put(ast3)

	// Check ast2 was evicted
	_, exists := cache.Get("tests.create")
	assert.False(t, exists, "tests.create should have been evicted")

	// Check ast1 and ast3 are still present
	_, exists = cache.Get("tests.mcpload")
	assert.True(t, exists, "tests.mcpload should still be in cache")

	_, exists = cache.Get("tests.next")
	assert.True(t, exists, "tests.next should be in cache")
}

func TestCacheRemove(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	cache := assistant.NewCache(2)

	ast1, _ := assistant.Get("tests.mcpload")
	cache.Put(ast1)

	// Verify scripts are registered
	_, exists := process.Handlers["agents.tests.mcpload.tools"]
	assert.True(t, exists, "Handler should be registered before removal")

	// Test remove existing item
	cache.Remove("tests.mcpload")
	assert.Equal(t, 0, cache.Len(), "Cache should be empty after removing item")

	// Verify scripts are unregistered
	_, exists = process.Handlers["agents.tests.mcpload.tools"]
	assert.False(t, exists, "Handler should be unregistered after removal")

	// Test remove non-existing item (should not panic)
	cache.Remove("nonexistent")
	assert.Equal(t, 0, cache.Len(), "Cache length should not change")
}

func TestCacheClear(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	cache := assistant.NewCache(3)

	ast1, _ := assistant.Get("tests.mcpload")
	ast2, _ := assistant.Get("tests.create")
	ast3, _ := assistant.Get("tests.next")

	cache.Put(ast1)
	cache.Put(ast2)
	cache.Put(ast3)

	assert.Equal(t, 3, cache.Len(), "Cache should have 3 items")

	// Verify scripts are registered
	_, exists := process.Handlers["agents.tests.mcpload.tools"]
	assert.True(t, exists, "Handler should be registered before clear")

	// Clear cache
	cache.Clear()
	assert.Equal(t, 0, cache.Len(), "Cache should be empty after clear")

	// Verify all scripts are unregistered
	_, exists = process.Handlers["agents.tests.mcpload.tools"]
	assert.False(t, exists, "Handler should be unregistered after clear")
}

func TestCacheLRUEviction(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	cache := assistant.NewCache(2)

	ast1, _ := assistant.Get("tests.mcpload")
	ast2, _ := assistant.Get("tests.create")
	ast3, _ := assistant.Get("tests.next")

	cache.Put(ast1)
	cache.Put(ast2)

	// Verify both are registered
	_, exists1 := process.Handlers["agents.tests.mcpload.tools"]
	assert.True(t, exists1, "Handler 1 should be registered")

	// Add third item to trigger LRU eviction of oldest (ast1)
	cache.Put(ast3)

	// Verify ast1's handler was unregistered due to eviction
	_, exists := process.Handlers["agents.tests.mcpload.tools"]
	assert.False(t, exists, "Handler should be unregistered after LRU eviction")

	// Verify ast2 and ast3 are still in cache
	_, exists = cache.Get("tests.create")
	assert.True(t, exists, "tests.create should still be in cache")

	_, exists = cache.Get("tests.next")
	assert.True(t, exists, "tests.next should be in cache")
}

func TestCacheConcurrent(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	cache := assistant.NewCache(10)
	var wg sync.WaitGroup
	workers := 5
	iterations := 20

	// Load some assistants for concurrent testing
	assistants := []string{
		"tests.mcpload",
		"tests.create",
		"tests.next",
	}

	// Concurrent writes
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				astID := assistants[j%len(assistants)]
				ast, _ := assistant.Get(astID)
				if ast != nil {
					cache.Put(ast)
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				astID := assistants[j%len(assistants)]
				cache.Get(astID)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is in valid state
	assert.True(t, cache.Len() >= 0, "Cache should have valid length")
	assert.True(t, cache.Len() <= 10, "Cache should not exceed capacity")
}

func TestCacheNilInput(t *testing.T) {
	cache := assistant.NewCache(2)

	// Test putting nil assistant
	cache.Put(nil)
	assert.Equal(t, 0, cache.Len(), "Cache should not store nil assistant")

	// Test putting assistant with empty ID
	emptyAST := &assistant.Assistant{}
	cache.Put(emptyAST)
	assert.Equal(t, 0, cache.Len(), "Cache should not store assistant with empty ID")
}

func TestCacheAll(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	cache := assistant.NewCache(5)

	ast1, _ := assistant.Get("tests.mcpload")
	ast2, _ := assistant.Get("tests.create")
	ast3, _ := assistant.Get("tests.next")

	cache.Put(ast1)
	cache.Put(ast2)
	cache.Put(ast3)

	all := cache.All()
	assert.Equal(t, 3, len(all), "All() should return 3 assistants")

	// Verify all expected assistants are present
	ids := make(map[string]bool)
	for _, ast := range all {
		ids[ast.ID] = true
	}

	assert.True(t, ids["tests.mcpload"], "Should contain tests.mcpload")
	assert.True(t, ids["tests.create"], "Should contain tests.create")
	assert.True(t, ids["tests.next"], "Should contain tests.next")
}
