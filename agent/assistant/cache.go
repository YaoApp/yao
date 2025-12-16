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
		item := element.Value.(*cacheItem)

		// Unregister scripts before removing from cache
		if item.value != nil && len(item.value.Scripts) > 0 {
			item.value.UnregisterScripts()
		}

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

// All returns all assistants in the cache
func (c *Cache) All() []*Assistant {
	c.mu.RLock()
	defer c.mu.RUnlock()

	assistants := make([]*Assistant, 0, c.list.Len())
	for element := c.list.Front(); element != nil; element = element.Next() {
		item := element.Value.(*cacheItem)
		assistants = append(assistants, item.value)
	}
	return assistants
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Unregister all scripts before clearing cache
	for element := c.list.Front(); element != nil; element = element.Next() {
		item := element.Value.(*cacheItem)
		if item.value != nil && len(item.value.Scripts) > 0 {
			item.value.UnregisterScripts()
		}
	}

	c.list.Init()
	c.items = make(map[string]*list.Element)
}

// ClearExcept removes items from the cache except those matching the keep function
// keep function returns true for items that should be preserved
func (c *Cache) ClearExcept(keep func(id string) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Collect items to remove
	var toRemove []*list.Element
	for element := c.list.Front(); element != nil; element = element.Next() {
		item := element.Value.(*cacheItem)
		if !keep(item.key) {
			toRemove = append(toRemove, element)
		}
	}

	// Remove collected items
	for _, element := range toRemove {
		item := element.Value.(*cacheItem)

		// Unregister scripts before removing
		if item.value != nil && len(item.value.Scripts) > 0 {
			item.value.UnregisterScripts()
		}

		c.list.Remove(element)
		delete(c.items, item.key)
	}
}

// removeOldest removes the least recently used item from the cache
func (c *Cache) removeOldest() {
	if element := c.list.Back(); element != nil {
		item := element.Value.(*cacheItem)

		// Unregister scripts before removing from cache
		if item.value != nil && len(item.value.Scripts) > 0 {
			item.value.UnregisterScripts()
		}

		c.list.Remove(element)
		delete(c.items, item.key)
	}
}
