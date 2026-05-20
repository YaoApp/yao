//go:build unit

package output_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/agent/output"
)

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
	sw := output.NewSafeWriter(mock)
	defer sw.Close()

	n, err := sw.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected 5 bytes written, got %d", n)
	}

	time.Sleep(10 * time.Millisecond)

	if got := mock.String(); got != "hello" {
		t.Errorf("Expected 'hello', got '%s'", got)
	}
}

func TestSafeWriter_ConcurrentWrites(t *testing.T) {
	mock := newMockResponseWriter()
	sw := output.NewSafeWriter(mock)

	numGoroutines := 100
	writesPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				sw.Write([]byte("X"))
			}
		}(i)
	}

	wg.Wait()

	sw.Close()

	expectedLen := numGoroutines * writesPerGoroutine
	if got := len(mock.String()); got != expectedLen {
		t.Errorf("Expected %d bytes, got %d", expectedLen, got)
	}

	if mock.FlushCount() < expectedLen {
		t.Errorf("Expected at least %d flushes, got %d", expectedLen, mock.FlushCount())
	}
}

func TestSafeWriter_NoDataCorruption(t *testing.T) {
	mock := newMockResponseWriter()
	sw := output.NewSafeWriter(mock)

	numGoroutines := 26
	msgLen := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			char := byte('A' + id)
			msg := bytes.Repeat([]byte{char}, msgLen)
			sw.Write(msg)
		}(i)
	}

	wg.Wait()

	sw.Close()

	result := mock.String()
	expectedLen := numGoroutines * msgLen
	if len(result) != expectedLen {
		t.Errorf("Expected %d bytes, got %d", expectedLen, len(result))
	}

	blocks := make(map[byte]int)
	for i := 0; i < len(result); i += msgLen {
		if i+msgLen > len(result) {
			t.Errorf("Unexpected data at end of result")
			break
		}
		block := result[i : i+msgLen]
		firstChar := block[0]
		for j, c := range []byte(block) {
			if c != firstChar {
				t.Errorf("Data corruption detected at position %d: expected %c, got %c", i+j, firstChar, c)
				break
			}
		}
		blocks[firstChar]++
	}

	for char, count := range blocks {
		if count != 1 {
			t.Errorf("Character %c appeared %d times, expected 1", char, count)
		}
	}

	if len(blocks) != numGoroutines {
		t.Errorf("Expected %d distinct blocks, got %d", numGoroutines, len(blocks))
	}
}

func TestSafeWriter_CloseWaitsForPendingWrites(t *testing.T) {
	mock := newMockResponseWriter()
	sw := output.NewSafeWriter(mock)

	numWrites := 1000
	for i := 0; i < numWrites; i++ {
		sw.Write([]byte("X"))
	}

	sw.Close()

	if got := len(mock.String()); got != numWrites {
		t.Errorf("Expected %d bytes after close, got %d", numWrites, got)
	}
}

func TestSafeWriter_WriteAfterClose(t *testing.T) {
	mock := newMockResponseWriter()
	sw := output.NewSafeWriter(mock)

	sw.Write([]byte("before"))
	sw.Close()

	n, err := sw.Write([]byte("after"))
	if err != nil {
		t.Errorf("Write after close should not error: %v", err)
	}
	if n != 0 {
		t.Errorf("Write after close should return 0, got %d", n)
	}

	if got := mock.String(); got != "before" {
		t.Errorf("Expected 'before', got '%s'", got)
	}
}

func TestSafeWriter_ImplementsHTTPInterfaces(t *testing.T) {
	mock := newMockResponseWriter()
	sw := output.NewSafeWriter(mock)
	defer sw.Close()

	var _ http.ResponseWriter = sw

	var _ http.Flusher = sw

	sw.Header().Set("Content-Type", "text/plain")
	if got := mock.Header().Get("Content-Type"); got != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got '%s'", got)
	}
}

func BenchmarkSafeWriter_ConcurrentWrites(b *testing.B) {
	mock := newMockResponseWriter()
	sw := output.NewSafeWriter(mock)
	defer sw.Close()

	data := []byte("benchmark data for SSE streaming")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sw.Write(data)
		}
	})
}

func TestSafeWriter_RealHTTPServer(t *testing.T) {
	recorder := httptest.NewRecorder()
	sw := output.NewSafeWriter(recorder)

	var wg sync.WaitGroup
	numAgents := 10
	messagesPerAgent := 10

	wg.Add(numAgents)
	for i := 0; i < numAgents; i++ {
		go func(agentID int) {
			defer wg.Done()
			for j := 0; j < messagesPerAgent; j++ {
				msg := []byte("data: {\"agent\":" + string(rune('0'+agentID)) + "}\n\n")
				sw.Write(msg)
			}
		}(i)
	}

	wg.Wait()
	sw.Close()

	body := recorder.Body.String()
	expectedMsgs := numAgents * messagesPerAgent

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

func TestSafeWriter_ContextCancellation(t *testing.T) {
	mock := newMockResponseWriter()

	ctx, cancel := context.WithCancel(context.Background())

	sw := output.NewSafeWriterWithContext(ctx, mock)

	sw.Write([]byte("before"))
	time.Sleep(20 * time.Millisecond)

	if got := mock.String(); got != "before" {
		t.Errorf("Expected 'before' before cancel, got '%s'", got)
	}

	cancel()

	sw.Write([]byte("after_cancel"))

	sw.Close()

	select {
	case <-output.DoneForTest(sw):
	default:
		t.Error("run() should have exited after Close()")
	}

	got := mock.String()
	if len(got) < 6 {
		t.Errorf("Expected at least 'before', got '%s'", got)
	}
}

func TestSafeWriter_GoroutineLeak(t *testing.T) {
	numWriters := 100

	var wg sync.WaitGroup
	wg.Add(numWriters)

	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()

			mock := newMockResponseWriter()
			ctx, cancel := context.WithCancel(context.Background())
			sw := output.NewSafeWriterWithContext(ctx, mock)

			sw.Write([]byte("test"))

			if time.Now().UnixNano()%2 == 0 {
				cancel()
				time.Sleep(5 * time.Millisecond)
				sw.Close()
			} else {
				sw.Close()
				cancel()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for goroutines to complete - possible goroutine leak")
	}
}
