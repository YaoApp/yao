package output

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// mockResponseWriter is a thread-safe mock for testing
type mockResponseWriter struct {
	mu       sync.Mutex
	buf      bytes.Buffer
	header   http.Header
	flushed  int
	writeErr error
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		header: make(http.Header),
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.buf.Write(data)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {}

func (m *mockResponseWriter) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushed++
}

func (m *mockResponseWriter) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.String()
}

func (m *mockResponseWriter) FlushCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.flushed
}

func TestSafeWriter_BasicWrite(t *testing.T) {
	mock := newMockResponseWriter()
	sw := NewSafeWriter(mock)
	defer sw.Close()

	// Write some data
	n, err := sw.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected 5 bytes written, got %d", n)
	}

	// Wait for async write to complete
	time.Sleep(10 * time.Millisecond)

	// Verify data was written
	if got := mock.String(); got != "hello" {
		t.Errorf("Expected 'hello', got '%s'", got)
	}
}

func TestSafeWriter_ConcurrentWrites(t *testing.T) {
	mock := newMockResponseWriter()
	sw := NewSafeWriter(mock)

	// Number of concurrent goroutines
	numGoroutines := 100
	// Number of writes per goroutine
	writesPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				sw.Write([]byte("X"))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Close and wait for all writes to be processed
	sw.Close()

	// Verify all data was written (no data loss)
	expectedLen := numGoroutines * writesPerGoroutine
	if got := len(mock.String()); got != expectedLen {
		t.Errorf("Expected %d bytes, got %d", expectedLen, got)
	}

	// Verify flush was called (at least once per write)
	if mock.FlushCount() < expectedLen {
		t.Errorf("Expected at least %d flushes, got %d", expectedLen, mock.FlushCount())
	}
}

func TestSafeWriter_NoDataCorruption(t *testing.T) {
	mock := newMockResponseWriter()
	sw := NewSafeWriter(mock)

	// Use exactly 26 goroutines (one per letter A-Z) to avoid duplicates
	numGoroutines := 26
	// Message to write (with unique content per goroutine)
	msgLen := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch concurrent writes with different content
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			// Create a message with repeating character (unique per goroutine)
			char := byte('A' + id)
			msg := bytes.Repeat([]byte{char}, msgLen)
			sw.Write(msg)
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Close and wait for all writes to be processed
	sw.Close()

	// Verify total length
	result := mock.String()
	expectedLen := numGoroutines * msgLen
	if len(result) != expectedLen {
		t.Errorf("Expected %d bytes, got %d", expectedLen, len(result))
	}

	// Verify no interleaving (each message should be contiguous)
	// Check that we have exactly numGoroutines distinct blocks
	blocks := make(map[byte]int)
	for i := 0; i < len(result); i += msgLen {
		if i+msgLen > len(result) {
			t.Errorf("Unexpected data at end of result")
			break
		}
		block := result[i : i+msgLen]
		// Verify block is homogeneous (all same character)
		firstChar := block[0]
		for j, c := range []byte(block) {
			if c != firstChar {
				t.Errorf("Data corruption detected at position %d: expected %c, got %c", i+j, firstChar, c)
				break
			}
		}
		blocks[firstChar]++
	}

	// Each character should appear exactly once (one block per goroutine)
	for char, count := range blocks {
		if count != 1 {
			t.Errorf("Character %c appeared %d times, expected 1", char, count)
		}
	}

	// Verify we got all 26 letters
	if len(blocks) != numGoroutines {
		t.Errorf("Expected %d distinct blocks, got %d", numGoroutines, len(blocks))
	}
}

func TestSafeWriter_CloseWaitsForPendingWrites(t *testing.T) {
	mock := newMockResponseWriter()
	sw := NewSafeWriter(mock)

	// Write a large number of messages
	numWrites := 1000
	for i := 0; i < numWrites; i++ {
		sw.Write([]byte("X"))
	}

	// Close should wait for all writes to complete
	sw.Close()

	// Verify all data was written
	if got := len(mock.String()); got != numWrites {
		t.Errorf("Expected %d bytes after close, got %d", numWrites, got)
	}
}

func TestSafeWriter_WriteAfterClose(t *testing.T) {
	mock := newMockResponseWriter()
	sw := NewSafeWriter(mock)

	sw.Write([]byte("before"))
	sw.Close()

	// Write after close should be silently ignored
	n, err := sw.Write([]byte("after"))
	if err != nil {
		t.Errorf("Write after close should not error: %v", err)
	}
	if n != 0 {
		t.Errorf("Write after close should return 0, got %d", n)
	}

	// Verify only "before" was written
	if got := mock.String(); got != "before" {
		t.Errorf("Expected 'before', got '%s'", got)
	}
}

func TestSafeWriter_ImplementsHTTPInterfaces(t *testing.T) {
	mock := newMockResponseWriter()
	sw := NewSafeWriter(mock)
	defer sw.Close()

	// Verify it implements http.ResponseWriter
	var _ http.ResponseWriter = sw

	// Verify it implements http.Flusher
	var _ http.Flusher = sw

	// Test Header()
	sw.Header().Set("Content-Type", "text/plain")
	if got := mock.Header().Get("Content-Type"); got != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got '%s'", got)
	}
}

// BenchmarkSafeWriter_ConcurrentWrites benchmarks concurrent write performance
func BenchmarkSafeWriter_ConcurrentWrites(b *testing.B) {
	mock := newMockResponseWriter()
	sw := NewSafeWriter(mock)
	defer sw.Close()

	data := []byte("benchmark data for SSE streaming")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sw.Write(data)
		}
	})
}

// TestSafeWriter_RealHTTPServer tests SafeWriter with a real HTTP server
func TestSafeWriter_RealHTTPServer(t *testing.T) {
	// This test verifies SafeWriter works correctly with httptest.ResponseRecorder
	// which is commonly used in testing HTTP handlers

	recorder := httptest.NewRecorder()
	sw := NewSafeWriter(recorder)

	// Simulate concurrent SSE writes from multiple sub-agents
	var wg sync.WaitGroup
	numAgents := 10
	messagesPerAgent := 10

	wg.Add(numAgents)
	for i := 0; i < numAgents; i++ {
		go func(agentID int) {
			defer wg.Done()
			for j := 0; j < messagesPerAgent; j++ {
				// Simulate SSE message format
				msg := []byte("data: {\"agent\":" + string(rune('0'+agentID)) + "}\n\n")
				sw.Write(msg)
			}
		}(i)
	}

	wg.Wait()
	sw.Close()

	// Verify response contains all messages (no data loss)
	body := recorder.Body.String()
	expectedMsgs := numAgents * messagesPerAgent

	// Count number of "data: " prefixes
	count := 0
	for i := 0; i < len(body); i++ {
		if i+6 <= len(body) && body[i:i+6] == "data: " {
			count++
		}
	}

	if count != expectedMsgs {
		t.Errorf("Expected %d messages, found %d", expectedMsgs, count)
	}
}

// TestSafeWriter_ContextCancellation tests that SafeWriter handles context cancellation
// This is critical for enterprise applications to prevent goroutine leaks
func TestSafeWriter_ContextCancellation(t *testing.T) {
	mock := newMockResponseWriter()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	sw := NewSafeWriterWithContext(ctx, mock)

	// Write some data and wait for it to be processed
	sw.Write([]byte("before"))
	time.Sleep(20 * time.Millisecond)

	// Verify "before" was written
	if got := mock.String(); got != "before" {
		t.Errorf("Expected 'before' before cancel, got '%s'", got)
	}

	// Cancel context (simulates client disconnect)
	cancel()

	// Write after context cancellation - these may or may not be written
	// depending on timing (select may pick ctx.Done() first)
	sw.Write([]byte("after_cancel"))

	// Close properly cleans up
	sw.Close()

	// After close, run() has exited
	select {
	case <-sw.done:
		// Good - run() has exited
	default:
		t.Error("run() should have exited after Close()")
	}

	// The key guarantee: run() goroutine exits cleanly, no leak
	// Data written before cancel is preserved
	got := mock.String()
	if len(got) < 6 { // At least "before" should be there
		t.Errorf("Expected at least 'before', got '%s'", got)
	}
}

// TestSafeWriter_GoroutineLeak tests that SafeWriter doesn't leak goroutines
func TestSafeWriter_GoroutineLeak(t *testing.T) {
	// Create many SafeWriters and ensure they all clean up properly
	numWriters := 100

	var wg sync.WaitGroup
	wg.Add(numWriters)

	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()

			mock := newMockResponseWriter()
			ctx, cancel := context.WithCancel(context.Background())
			sw := NewSafeWriterWithContext(ctx, mock)

			// Write some data
			sw.Write([]byte("test"))

			// Randomly either close normally or cancel context
			if time.Now().UnixNano()%2 == 0 {
				cancel()
				time.Sleep(5 * time.Millisecond)
				sw.Close()
			} else {
				sw.Close()
				cancel() // Cancel after close is safe
			}
		}()
	}

	// All goroutines should complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All completed successfully
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for goroutines to complete - possible goroutine leak")
	}
}
