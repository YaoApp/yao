package assistant

import (
	"container/list"
	"sync"
)

// Cache represents a thread-safe LRU cache for Assistant objects
type Cache struct {
	capacity int
	mu       sync.RWMutex
	list     *list.List
	items    map[string]*list.Element
}

// cacheItem represents an item in the cache
type cacheItem struct {
	key   string
	value *Assistant
}

// NewCache creates a new LRU cache with the given capacity
func NewCache(capacity int) *Cache {
	return &Cache{
		capacity: capacity,
		list:     list.New(),
		items:    make(map[string]*list.Element),
	}
}

// Get retrieves an Assistant from the cache by its ID
func (c *Cache) Get(id string) (*Assistant, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[id]; exists {
		c.list.MoveToFront(element)
		return element.Value.(*cacheItem).value, true
	}
	return nil, false
}

// Put adds or updates an Assistant in the cache
func (c *Cache) Put(assistant *Assistant) {
	if assistant == nil || assistant.ID == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// If item exists, update it and move to front
	if element, exists := c.items[assistant.ID]; exists {
		c.list.MoveToFront(element)
		element.Value.(*cacheItem).value = assistant
		return
	}

	// If cache is at capacity, remove oldest item before adding new one
	if c.list.Len() >= c.capacity {
		c.removeOldest()
	}

	// Add new item
	element := c.list.PushFront(&cacheItem{
		key:   assistant.ID,
		value: assistant,
	})
	c.items[assistant.ID] = element
}

// Remove removes an Assistant from the cache
func (c *Cache) Remove(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[id]; exists {
		c.list.Remove(element)
		delete(c.items, id)
	}
}

// Len returns the current number of items in the cache
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.list.Len()
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.list.Init()
	c.items = make(map[string]*list.Element)
}

// removeOldest removes the least recently used item from the cache
func (c *Cache) removeOldest() {
	if element := c.list.Back(); element != nil {
		c.list.Remove(element)
		delete(c.items, element.Value.(*cacheItem).key)
	}
}
