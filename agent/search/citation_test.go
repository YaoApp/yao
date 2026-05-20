//go:build unit

package search_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search"
)

func TestCitationGenerator_Next(t *testing.T) {
	gen := search.NewCitationGenerator()

	id1 := gen.Next()
	assert.Equal(t, "1", id1)

	id2 := gen.Next()
	assert.Equal(t, "2", id2)

	id3 := gen.Next()
	assert.Equal(t, "3", id3)
}

func TestCitationGenerator_NextInt(t *testing.T) {
	gen := search.NewCitationGenerator()

	id1 := gen.NextInt()
	assert.Equal(t, 1, id1)

	id2 := gen.NextInt()
	assert.Equal(t, 2, id2)
}

func TestCitationGenerator_Current(t *testing.T) {
	gen := search.NewCitationGenerator()

	assert.Equal(t, 0, gen.Current())

	gen.Next()
	assert.Equal(t, 1, gen.Current())

	assert.Equal(t, 1, gen.Current())
}

func TestCitationGenerator_Reset(t *testing.T) {
	gen := search.NewCitationGenerator()

	gen.Next()
	gen.Next()
	gen.Next()

	gen.Reset()

	id := gen.Next()
	assert.Equal(t, "1", id)
}

func TestCitationGenerator_LargeNumbers(t *testing.T) {
	gen := search.NewCitationGenerator()

	for i := 0; i < 999; i++ {
		gen.Next()
	}

	id := gen.Next()
	assert.Equal(t, "1000", id)
}

func TestCitationGenerator_Concurrent(t *testing.T) {
	gen := search.NewCitationGenerator()

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

	idSet := make(map[string]bool)
	for id := range ids {
		idSet[id] = true
	}

	assert.Equal(t, 1000, len(idSet))
}

func TestNewCitationGenerator(t *testing.T) {
	gen := search.NewCitationGenerator()
	assert.NotNil(t, gen)
}

func TestUint64ToString(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{999, "999"},
		{1000, "1000"},
		{18446744073709551615, "18446744073709551615"}, // max uint64
	}

	for _, tt := range tests {
		result := search.Uint64ToStringForTest(tt.input)
		assert.Equal(t, tt.expected, result, "uint64ToString(%d)", tt.input)
	}
}
