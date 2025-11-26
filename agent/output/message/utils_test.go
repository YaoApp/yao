package message

import (
	"sync"
	"testing"
)

func TestIDGenerator(t *testing.T) {
	gen := NewIDGenerator()

	t.Run("GenerateChunkID", func(t *testing.T) {
		id1 := gen.GenerateChunkID()
		id2 := gen.GenerateChunkID()
		id3 := gen.GenerateChunkID()

		if id1 != "C1" {
			t.Errorf("Expected C1, got %s", id1)
		}
		if id2 != "C2" {
			t.Errorf("Expected C2, got %s", id2)
		}
		if id3 != "C3" {
			t.Errorf("Expected C3, got %s", id3)
		}
	})

	t.Run("GenerateMessageID", func(t *testing.T) {
		gen := NewIDGenerator()
		id1 := gen.GenerateMessageID()
		id2 := gen.GenerateMessageID()
		id3 := gen.GenerateMessageID()

		if id1 != "M1" {
			t.Errorf("Expected M1, got %s", id1)
		}
		if id2 != "M2" {
			t.Errorf("Expected M2, got %s", id2)
		}
		if id3 != "M3" {
			t.Errorf("Expected M3, got %s", id3)
		}
	})

	t.Run("GenerateBlockID", func(t *testing.T) {
		gen := NewIDGenerator()
		id1 := gen.GenerateBlockID()
		id2 := gen.GenerateBlockID()
		id3 := gen.GenerateBlockID()

		if id1 != "B1" {
			t.Errorf("Expected B1, got %s", id1)
		}
		if id2 != "B2" {
			t.Errorf("Expected B2, got %s", id2)
		}
		if id3 != "B3" {
			t.Errorf("Expected B3, got %s", id3)
		}
	})

	t.Run("GenerateThreadID", func(t *testing.T) {
		gen := NewIDGenerator()
		id1 := gen.GenerateThreadID()
		id2 := gen.GenerateThreadID()
		id3 := gen.GenerateThreadID()

		if id1 != "T1" {
			t.Errorf("Expected T1, got %s", id1)
		}
		if id2 != "T2" {
			t.Errorf("Expected T2, got %s", id2)
		}
		if id3 != "T3" {
			t.Errorf("Expected T3, got %s", id3)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		gen := NewIDGenerator()
		gen.GenerateChunkID()
		gen.GenerateMessageID()
		gen.GenerateBlockID()
		gen.GenerateThreadID()

		gen.Reset()

		chunk, message, block, thread := gen.GetCounters()
		if chunk != 0 || message != 0 || block != 0 || thread != 0 {
			t.Errorf("Expected all counters to be 0 after reset, got chunk=%d, message=%d, block=%d, thread=%d",
				chunk, message, block, thread)
		}

		// Verify IDs start from 1 again
		if id := gen.GenerateChunkID(); id != "C1" {
			t.Errorf("Expected C1 after reset, got %s", id)
		}
		if id := gen.GenerateMessageID(); id != "M1" {
			t.Errorf("Expected M1 after reset, got %s", id)
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		gen := NewIDGenerator()
		var wg sync.WaitGroup
		count := 100

		// Test concurrent chunk ID generation
		wg.Add(count)
		for i := 0; i < count; i++ {
			go func() {
				defer wg.Done()
				gen.GenerateChunkID()
			}()
		}
		wg.Wait()

		chunk, _, _, _ := gen.GetCounters()
		if chunk != uint64(count) {
			t.Errorf("Expected chunk counter to be %d, got %d", count, chunk)
		}
	})

	t.Run("MultipleGenerators", func(t *testing.T) {
		gen1 := NewIDGenerator()
		gen2 := NewIDGenerator()

		id1 := gen1.GenerateMessageID()
		id2 := gen2.GenerateMessageID()

		// Both should start from M1
		if id1 != "M1" || id2 != "M1" {
			t.Errorf("Expected both generators to start from M1, got %s and %s", id1, id2)
		}

		// Advance gen1
		gen1.GenerateMessageID()
		gen1.GenerateMessageID()

		// gen2 should still be at M1
		id2_next := gen2.GenerateMessageID()
		if id2_next != "M2" {
			t.Errorf("Expected gen2 to be at M2, got %s", id2_next)
		}

		// gen1 should be at M3
		id1_next := gen1.GenerateMessageID()
		if id1_next != "M4" {
			t.Errorf("Expected gen1 to be at M4, got %s", id1_next)
		}
	})
}

func TestGenerateNanoID(t *testing.T) {
	id1 := GenerateNanoID()
	id2 := GenerateNanoID()

	// NanoID should be 21 characters by default
	if len(id1) != 21 {
		t.Errorf("Expected NanoID length to be 21, got %d", len(id1))
	}

	// IDs should be unique
	if id1 == id2 {
		t.Error("Expected unique NanoIDs, got duplicates")
	}

	t.Logf("Generated NanoIDs: %s, %s", id1, id2)
}

func TestGenerateCustomID(t *testing.T) {
	id1 := GenerateCustomID("msg")
	id2 := GenerateCustomID("evt")

	// Should have prefix
	if len(id1) < 4 || id1[:4] != "msg_" {
		t.Errorf("Expected ID to start with 'msg_', got %s", id1)
	}
	if len(id2) < 4 || id2[:4] != "evt_" {
		t.Errorf("Expected ID to start with 'evt_', got %s", id2)
	}

	// IDs should be unique
	if id1 == id2 {
		t.Error("Expected unique custom IDs, got duplicates")
	}

	t.Logf("Generated custom IDs: %s, %s", id1, id2)
}
