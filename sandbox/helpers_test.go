package sandbox

import (
	"os"
	"testing"
)

func TestParseMemory(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1024", 1024},
		{"1k", 1024},
		{"1K", 1024},
		{"1m", 1024 * 1024},
		{"1M", 1024 * 1024},
		{"2g", 2 * 1024 * 1024 * 1024},
		{"2G", 2 * 1024 * 1024 * 1024},
		{"1t", 1024 * 1024 * 1024 * 1024},
		{"1.5g", int64(1.5 * 1024 * 1024 * 1024)},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		result := parseMemory(tt.input)
		if result != tt.expected {
			t.Errorf("parseMemory(%s) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestMapToSlice(t *testing.T) {
	// Nil map
	result := mapToSlice(nil)
	if result != nil {
		t.Errorf("mapToSlice(nil) should return nil")
	}

	// Empty map
	result = mapToSlice(map[string]string{})
	if len(result) != 0 {
		t.Errorf("mapToSlice(empty) should return empty slice")
	}

	// Map with values
	m := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}
	result = mapToSlice(m)
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}

	// Check that all items are in format KEY=value
	found := make(map[string]bool)
	for _, item := range result {
		found[item] = true
	}
	if !found["KEY1=value1"] || !found["KEY2=value2"] {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestParseLS(t *testing.T) {
	output := `total 8
drwxr-xr-x 2 sandbox sandbox 4096 1700000000 dir1
-rw-r--r-- 1 sandbox sandbox 100 1700000001 file1.txt
lrwxrwxrwx 1 sandbox sandbox 10 1700000002 link1 -> file1.txt
`

	result := parseLS(output)

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	// Check dir1
	if result[0].Name != "dir1" {
		t.Errorf("expected name 'dir1', got '%s'", result[0].Name)
	}
	if !result[0].IsDir {
		t.Errorf("expected dir1 to be a directory")
	}

	// Check file1.txt
	if result[1].Name != "file1.txt" {
		t.Errorf("expected name 'file1.txt', got '%s'", result[1].Name)
	}
	if result[1].Size != 100 {
		t.Errorf("expected size 100, got %d", result[1].Size)
	}
	if result[1].IsDir {
		t.Errorf("expected file1.txt to be a file, not directory")
	}

	// Check link1
	if result[2].Name != "link1 -> file1.txt" {
		t.Errorf("expected name 'link1 -> file1.txt', got '%s'", result[2].Name)
	}
}

func TestParseStat(t *testing.T) {
	output := "/workspace/test.txt|1024|81a4|1700000000|regular file"

	result := parseStat(output)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != "test.txt" {
		t.Errorf("expected name 'test.txt', got '%s'", result.Name)
	}
	if result.Path != "/workspace/test.txt" {
		t.Errorf("expected path '/workspace/test.txt', got '%s'", result.Path)
	}
	if result.Size != 1024 {
		t.Errorf("expected size 1024, got %d", result.Size)
	}
	if result.IsDir {
		t.Errorf("expected IsDir to be false")
	}
}

func TestParseLSMode(t *testing.T) {
	tests := []struct {
		input    string
		isDir    bool
		readable bool
	}{
		{"drwxr-xr-x", true, true},
		{"-rw-r--r--", false, true},
		{"lrwxrwxrwx", false, true},
		{"-rwx------", false, true},
	}

	for _, tt := range tests {
		mode := parseLSMode(tt.input)
		isDir := mode.IsDir()
		if isDir != tt.isDir {
			t.Errorf("parseLSMode(%s).IsDir() = %v, want %v", tt.input, isDir, tt.isDir)
		}
	}
}

func TestCreateAndExtractTar(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := tmpDir + "/test.txt"
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create tar from file
	reader, err := createTarFromPath(testFile)
	if err != nil {
		t.Fatalf("createTarFromPath failed: %v", err)
	}
	defer reader.Close()

	// Extract to new location
	extractDir, err := os.MkdirTemp("", "sandbox-extract-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(extractDir)

	if err := extractTarToPath(reader, extractDir); err != nil {
		t.Fatalf("extractTarToPath failed: %v", err)
	}

	// Verify extracted file
	extractedFile := extractDir + "/test.txt"
	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(content))
	}
}
