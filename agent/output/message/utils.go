package message

import (
	"fmt"
	"sync/atomic"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// IDGenerator generates unique IDs within a context (e.g., one conversation stream)
// Each Context should have its own IDGenerator to ensure IDs are unique within that context
type IDGenerator struct {
	chunkCounter   uint64
	messageCounter uint64
	blockCounter   uint64
	threadCounter  uint64
}

// NewIDGenerator creates a new ID generator for a context
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

// GenerateChunkID generates a unique chunk ID with prefix C
// Format: C1, C2, C3...
func (g *IDGenerator) GenerateChunkID() string {
	id := atomic.AddUint64(&g.chunkCounter, 1)
	return fmt.Sprintf("C%d", id)
}

// GenerateMessageID generates a unique message ID with prefix M
// Format: M1, M2, M3...
func (g *IDGenerator) GenerateMessageID() string {
	id := atomic.AddUint64(&g.messageCounter, 1)
	return fmt.Sprintf("M%d", id)
}

// GenerateBlockID generates a unique block ID with prefix B
// Format: B1, B2, B3...
func (g *IDGenerator) GenerateBlockID() string {
	id := atomic.AddUint64(&g.blockCounter, 1)
	return fmt.Sprintf("B%d", id)
}

// GenerateThreadID generates a unique thread ID with prefix T
// Format: T1, T2, T3...
func (g *IDGenerator) GenerateThreadID() string {
	id := atomic.AddUint64(&g.threadCounter, 1)
	return fmt.Sprintf("T%d", id)
}

// Reset resets all counters (useful for testing)
func (g *IDGenerator) Reset() {
	atomic.StoreUint64(&g.chunkCounter, 0)
	atomic.StoreUint64(&g.messageCounter, 0)
	atomic.StoreUint64(&g.blockCounter, 0)
	atomic.StoreUint64(&g.threadCounter, 0)
}

// GetCounters returns current counter values (for debugging/testing)
func (g *IDGenerator) GetCounters() (chunk, message, block, thread uint64) {
	return atomic.LoadUint64(&g.chunkCounter),
		atomic.LoadUint64(&g.messageCounter),
		atomic.LoadUint64(&g.blockCounter),
		atomic.LoadUint64(&g.threadCounter)
}

// GenerateNanoID generates a unique ID using nanoid
// Returns a 21-character URL-safe string
// This is a static function that doesn't depend on the generator's counter
func GenerateNanoID() string {
	id, err := gonanoid.New()
	if err != nil {
		// Fallback to timestamp-based ID if nanoid fails
		return fmt.Sprintf("id_%d", atomic.AddUint64(new(uint64), 1))
	}
	return id
}

// GenerateCustomID generates a custom ID with prefix and nanoid
// Format: prefix_nanoid (e.g., "msg_V1StGXR8_Z5jdHi6B-myT")
// This is a static function that doesn't depend on the generator's counter
func GenerateCustomID(prefix string) string {
	id, err := gonanoid.New()
	if err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%s_%d", prefix, atomic.AddUint64(new(uint64), 1))
	}
	return fmt.Sprintf("%s_%s", prefix, id)
}
