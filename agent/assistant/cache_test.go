package assistant

import (
	"sync"
	"testing"
)

func TestCache_Basic(t *testing.T) {
	cache := NewCache(2)

	// Test empty cache
	if cache.Len() != 0 {
		t.Errorf("Expected empty cache, got length %d", cache.Len())
	}

	// Test adding items
	assistant1 := &Assistant{ID: "1", Name: "Test1"}
	assistant2 := &Assistant{ID: "2", Name: "Test2"}

	cache.Put(assistant1)
	cache.Put(assistant2)

	if cache.Len() != 2 {
		t.Errorf("Expected cache length 2, got %d", cache.Len())
	}

	// Test getting items
	if a, exists := cache.Get("1"); !exists || a.ID != "1" {
		t.Error("Failed to get assistant1")
	}

	if a, exists := cache.Get("2"); !exists || a.ID != "2" {
		t.Error("Failed to get assistant2")
	}
}

func TestCache_LRU(t *testing.T) {
	cache := NewCache(2)

	assistant1 := &Assistant{ID: "1", Name: "Test1"}
	assistant2 := &Assistant{ID: "2", Name: "Test2"}
	assistant3 := &Assistant{ID: "3", Name: "Test3"}

	// Add first two items
	cache.Put(assistant1)
	cache.Put(assistant2)

	// Access assistant1 to make it most recently used
	cache.Get("1")

	// Add third item, should evict assistant2
	cache.Put(assistant3)

	// Check assistant2 was evicted
	if _, exists := cache.Get("2"); exists {
		t.Error("Assistant2 should have been evicted")
	}

	// Check assistant1 and assistant3 are still present
	if _, exists := cache.Get("1"); !exists {
		t.Error("Assistant1 should still be in cache")
	}
	if _, exists := cache.Get("3"); !exists {
		t.Error("Assistant3 should be in cache")
	}
}

func TestCache_Remove(t *testing.T) {
	cache := NewCache(2)

	assistant1 := &Assistant{ID: "1", Name: "Test1"}
	cache.Put(assistant1)

	// Test remove existing item
	cache.Remove("1")
	if cache.Len() != 0 {
		t.Error("Cache should be empty after removing item")
	}

	// Test remove non-existing item
	cache.Remove("nonexistent")
	if cache.Len() != 0 {
		t.Error("Cache length should not change when removing non-existent item")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(2)

	assistant1 := &Assistant{ID: "1", Name: "Test1"}
	assistant2 := &Assistant{ID: "2", Name: "Test2"}

	cache.Put(assistant1)
	cache.Put(assistant2)

	cache.Clear()
	if cache.Len() != 0 {
		t.Error("Cache should be empty after clear")
	}
}

func TestCache_Concurrent(t *testing.T) {
	cache := NewCache(100)
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	// Concurrent writes
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				assistant := &Assistant{
					ID:   string(rune('A' + workerID)),
					Name: "Test",
				}
				cache.Put(assistant)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cache.Get(string(rune('A' + workerID)))
			}
		}(i)
	}

	wg.Wait()
}

func TestCache_NilInput(t *testing.T) {
	cache := NewCache(2)

	// Test putting nil assistant
	cache.Put(nil)
	if cache.Len() != 0 {
		t.Error("Cache should not store nil assistant")
	}

	// Test putting assistant with empty ID
	cache.Put(&Assistant{ID: "", Name: "Test"})
	if cache.Len() != 0 {
		t.Error("Cache should not store assistant with empty ID")
	}
}
