package shared

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestReadJSONLine_ShortLine(t *testing.T) {
	input := `{"type":"system","text":"hello"}` + "\n"
	r := bufio.NewReaderSize(strings.NewReader(input), 64*1024)

	line, skipped, err := ReadJSONLine(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Fatal("expected skipped=false")
	}
	if string(line) != `{"type":"system","text":"hello"}` {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestReadJSONLine_EmptyLine(t *testing.T) {
	input := "\n" + `{"type":"ok"}` + "\n"
	r := bufio.NewReaderSize(strings.NewReader(input), 64*1024)

	line, skipped, err := ReadJSONLine(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Fatal("expected skipped=false")
	}
	if len(line) != 0 {
		t.Fatalf("expected empty line, got %d bytes", len(line))
	}

	line, skipped, err = ReadJSONLine(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Fatal("expected skipped=false for second line")
	}
	if string(line) != `{"type":"ok"}` {
		t.Fatalf("unexpected second line: %q", line)
	}
}

func TestReadJSONLine_MediumLine(t *testing.T) {
	payload := strings.Repeat("x", 200*1024) // 200KB — exceeds 64KB buffer, requires multiple ReadLine chunks
	input := payload + "\n"
	r := bufio.NewReaderSize(strings.NewReader(input), 64*1024)

	line, skipped, err := ReadJSONLine(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Fatal("expected skipped=false for 200KB line")
	}
	if len(line) != 200*1024 {
		t.Fatalf("expected 200KB, got %d bytes", len(line))
	}
}

func TestReadJSONLine_OversizedLine(t *testing.T) {
	saved := MaxLineSize
	defer func() {
		// MaxLineSize is a const; we test by embedding a smaller threshold.
		// Since we can't reassign a const, we build a line that exceeds 50MB.
		// Instead, we use a custom helper to keep the test fast.
		_ = saved
	}()

	// Build a line just over MaxLineSize. To keep allocations reasonable
	// in tests, we use a custom reader that streams repeated bytes.
	totalSize := MaxLineSize + 1024 // slightly over 50MB
	src := &repeatingReader{char: 'A', remaining: totalSize}
	// Append a newline + a short "next" line so we can verify recovery.
	combined := io.MultiReader(src, strings.NewReader("\n{\"type\":\"ok\"}\n"))
	r := bufio.NewReaderSize(combined, 64*1024)

	line, skipped, err := ReadJSONLine(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !skipped {
		t.Fatal("expected skipped=true for oversized line")
	}
	if line != nil {
		t.Fatalf("expected nil line when skipped, got %d bytes", len(line))
	}

	// Next line should be readable normally
	line, skipped, err = ReadJSONLine(r)
	if err != nil {
		t.Fatalf("unexpected error reading next line: %v", err)
	}
	if skipped {
		t.Fatal("expected skipped=false for recovery line")
	}
	if string(line) != `{"type":"ok"}` {
		t.Fatalf("unexpected recovery line: %q", line)
	}
}

func TestReadJSONLine_EOF(t *testing.T) {
	r := bufio.NewReaderSize(strings.NewReader(""), 64*1024)

	_, _, err := ReadJSONLine(r)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestReadJSONLine_EOFDuringDrain(t *testing.T) {
	// Oversized line without trailing newline — EOF during drain
	totalSize := MaxLineSize + 1024
	src := &repeatingReader{char: 'B', remaining: totalSize}
	r := bufio.NewReaderSize(src, 64*1024)

	_, skipped, err := ReadJSONLine(r)
	// During drain, ReadLine will eventually hit EOF
	if err == nil && !skipped {
		t.Fatal("expected either error or skip")
	}
	if err != nil && err != io.EOF {
		t.Fatalf("expected io.EOF during drain, got %v", err)
	}
}

func TestReadJSONLine_MultiLineMixed(t *testing.T) {
	// Line 1: normal short
	// Line 2: oversized
	// Line 3: normal short (verify recovery)
	var buf bytes.Buffer
	buf.WriteString(`{"type":"start"}`)
	buf.WriteByte('\n')

	oversized := strings.Repeat("Z", MaxLineSize+100)
	buf.WriteString(oversized)
	buf.WriteByte('\n')

	buf.WriteString(`{"type":"end"}`)
	buf.WriteByte('\n')

	r := bufio.NewReaderSize(&buf, 64*1024)

	// Line 1
	line, skipped, err := ReadJSONLine(r)
	if err != nil {
		t.Fatalf("line 1 error: %v", err)
	}
	if skipped {
		t.Fatal("line 1 should not be skipped")
	}
	if string(line) != `{"type":"start"}` {
		t.Fatalf("line 1 unexpected: %q", string(line))
	}

	// Line 2 (oversized)
	line, skipped, err = ReadJSONLine(r)
	if err != nil {
		t.Fatalf("line 2 error: %v", err)
	}
	if !skipped {
		t.Fatal("line 2 should be skipped")
	}
	if line != nil {
		t.Fatal("line 2 should return nil")
	}

	// Line 3 (recovery)
	line, skipped, err = ReadJSONLine(r)
	if err != nil {
		t.Fatalf("line 3 error: %v", err)
	}
	if skipped {
		t.Fatal("line 3 should not be skipped")
	}
	if string(line) != `{"type":"end"}` {
		t.Fatalf("line 3 unexpected: %q", string(line))
	}
}

func TestReadJSONLine_IOError(t *testing.T) {
	expectedErr := fmt.Errorf("simulated IO failure")
	r := bufio.NewReaderSize(&errorReader{err: expectedErr}, 64*1024)

	_, _, err := ReadJSONLine(r)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("expected %q, got %q", expectedErr, err)
	}
}

// repeatingReader streams a single character `remaining` times without a newline,
// then returns io.EOF. Used to test oversized lines without allocating huge buffers.
type repeatingReader struct {
	char      byte
	remaining int
}

func (r *repeatingReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > r.remaining {
		n = r.remaining
	}
	for i := 0; i < n; i++ {
		p[i] = r.char
	}
	r.remaining -= n
	return n, nil
}

// errorReader always returns the configured error.
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}
