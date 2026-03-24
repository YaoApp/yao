package weixin

import (
	"sync"
	"time"
)

const ticketTTL = 20 * time.Hour

type ticketEntry struct {
	ticket    string
	expiresAt time.Time
}

type typingTicketCache struct {
	mu    sync.RWMutex
	items map[string]*ticketEntry
}

func newTypingTicketCache() *typingTicketCache {
	return &typingTicketCache{
		items: make(map[string]*ticketEntry),
	}
}

func (c *typingTicketCache) Get(userID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[userID]
	if !ok || time.Now().After(entry.expiresAt) {
		return ""
	}
	return entry.ticket
}

func (c *typingTicketCache) Set(userID, ticket string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[userID] = &ticketEntry{
		ticket:    ticket,
		expiresAt: time.Now().Add(ticketTTL),
	}
}
