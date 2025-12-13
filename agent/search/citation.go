package search

import (
	"fmt"
	"sync/atomic"
)

// CitationGenerator generates unique citation IDs
type CitationGenerator struct {
	counter uint64
}

// NewCitationGenerator creates a new citation generator
func NewCitationGenerator() *CitationGenerator {
	return &CitationGenerator{}
}

// Next generates the next citation ID
func (g *CitationGenerator) Next() string {
	n := atomic.AddUint64(&g.counter, 1)
	return fmt.Sprintf("ref_%03d", n)
}

// Reset resets the counter (for testing)
func (g *CitationGenerator) Reset() {
	atomic.StoreUint64(&g.counter, 0)
}
