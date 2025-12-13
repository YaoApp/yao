package search

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCitationGenerator_Next(t *testing.T) {
	gen := NewCitationGenerator()

	// First ID should be ref_001
	id1 := gen.Next()
	assert.Equal(t, "ref_001", id1)

	// Second ID should be ref_002
	id2 := gen.Next()
	assert.Equal(t, "ref_002", id2)

	// Third ID should be ref_003
	id3 := gen.Next()
	assert.Equal(t, "ref_003", id3)
}

func TestCitationGenerator_Reset(t *testing.T) {
	gen := NewCitationGenerator()

	// Generate some IDs
	gen.Next()
	gen.Next()
	gen.Next()

	// Reset
	gen.Reset()

	// Next ID should be ref_001 again
	id := gen.Next()
	assert.Equal(t, "ref_001", id)
}

func TestCitationGenerator_Format(t *testing.T) {
	gen := NewCitationGenerator()

	// Generate 999 IDs to test padding
	for i := 0; i < 999; i++ {
		gen.Next()
	}

	// 1000th ID should be ref_1000 (no padding limit)
	id := gen.Next()
	assert.Equal(t, "ref_1000", id)
}

func TestCitationGenerator_Concurrent(t *testing.T) {
	gen := NewCitationGenerator()

	// Run 100 goroutines, each generating 10 IDs
	var wg sync.WaitGroup
	ids := make(chan string, 1000)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				ids <- gen.Next()
			}
		}()
	}

	wg.Wait()
	close(ids)

	// Collect all IDs
	idSet := make(map[string]bool)
	for id := range ids {
		idSet[id] = true
	}

	// All 1000 IDs should be unique
	assert.Equal(t, 1000, len(idSet))
}

func TestNewCitationGenerator(t *testing.T) {
	gen := NewCitationGenerator()
	assert.NotNil(t, gen)
}
