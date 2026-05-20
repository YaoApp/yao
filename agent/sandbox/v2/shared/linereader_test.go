//go:build unit

package shared_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	os.Exit(m.Run())
}

func TestReadJSONLine_ShortLine(t *testing.T) {
	input := `{"type":"system","text":"hello"}` + "\n"
	r := bufio.NewReaderSize(strings.NewReader(input), 64*1024)

	line, skipped, err := shared.ReadJSONLine(r)
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

	line, skipped, err := shared.ReadJSONLine(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skipped {
		t.Fatal("expected skipped=false")
	}
	if len(line) != 0 {
		t.Fatalf("expected empty line, got %d bytes", len(line))
	}

	line, skipped, err = shared.ReadJSONLine(r)
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
	payload := strings.Repeat("x", 200*1024)
	input := payload + "\n"
	r := bufio.NewReaderSize(strings.NewReader(input), 64*1024)

	line, skipped, err := shared.ReadJSONLine(r)
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
	totalSize := shared.MaxLineSize + 1024
	src := &repeatingReader{char: 'A', remaining: totalSize}
	combined := io.MultiReader(src, strings.NewReader("\n{\"type\":\"ok\"}\n"))
	r := bufio.NewReaderSize(combined, 64*1024)

	line, skipped, err := shared.ReadJSONLine(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !skipped {
		t.Fatal("expected skipped=true for oversized line")
	}
	if line != nil {
		t.Fatalf("expected nil line when skipped, got %d bytes", len(line))
	}

	line, skipped, err = shared.ReadJSONLine(r)
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

	_, _, err := shared.ReadJSONLine(r)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestReadJSONLine_EOFDuringDrain(t *testing.T) {
	totalSize := shared.MaxLineSize + 1024
	src := &repeatingReader{char: 'B', remaining: totalSize}
	r := bufio.NewReaderSize(src, 64*1024)

	_, skipped, err := shared.ReadJSONLine(r)
	if err == nil && !skipped {
		t.Fatal("expected either error or skip")
	}
	if err != nil && err != io.EOF {
		t.Fatalf("expected io.EOF during drain, got %v", err)
	}
}

func TestReadJSONLine_MultiLineMixed(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString(`{"type":"start"}`)
	buf.WriteByte('\n')

	oversized := strings.Repeat("Z", shared.MaxLineSize+100)
	buf.WriteString(oversized)
	buf.WriteByte('\n')

	buf.WriteString(`{"type":"end"}`)
	buf.WriteByte('\n')

	r := bufio.NewReaderSize(&buf, 64*1024)

	line, skipped, err := shared.ReadJSONLine(r)
	if err != nil {
		t.Fatalf("line 1 error: %v", err)
	}
	if skipped {
		t.Fatal("line 1 should not be skipped")
	}
	if string(line) != `{"type":"start"}` {
		t.Fatalf("line 1 unexpected: %q", string(line))
	}

	line, skipped, err = shared.ReadJSONLine(r)
	if err != nil {
		t.Fatalf("line 2 error: %v", err)
	}
	if !skipped {
		t.Fatal("line 2 should be skipped")
	}
	if line != nil {
		t.Fatal("line 2 should return nil")
	}

	line, skipped, err = shared.ReadJSONLine(r)
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

	_, _, err := shared.ReadJSONLine(r)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != expectedErr.Error() {
		t.Fatalf("expected %q, got %q", expectedErr, err)
	}
}

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

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}
