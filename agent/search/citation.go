package search

import (
	"sync/atomic"
)

// CitationGenerator generates unique citation IDs (1-based integers)
// Thread-safe for concurrent use within a single request
type CitationGenerator struct {
	counter uint64
}

// NewCitationGenerator creates a new citation generator
func NewCitationGenerator() *CitationGenerator {
	return &CitationGenerator{}
}

// Next generates the next citation ID (1, 2, 3, ...)
func (g *CitationGenerator) Next() string {
	n := atomic.AddUint64(&g.counter, 1)
	return uint64ToString(n)
}

// NextInt generates the next citation ID as integer
func (g *CitationGenerator) NextInt() int {
	return int(atomic.AddUint64(&g.counter, 1))
}

// Current returns the current counter value without incrementing
func (g *CitationGenerator) Current() int {
	return int(atomic.LoadUint64(&g.counter))
}

// Reset resets the counter (for testing)
func (g *CitationGenerator) Reset() {
	atomic.StoreUint64(&g.counter, 0)
}

// uint64ToString converts uint64 to string without fmt package
func uint64ToString(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte // max uint64 is 20 digits
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
