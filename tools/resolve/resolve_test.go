package resolve

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorkspaceURI(t *testing.T) {
	tests := []struct {
		uri      string
		wantID   string
		wantPath string
	}{
		{"workspace://ws-123/photo.jpg", "ws-123", "photo.jpg"},
		{"workspace://abc/nested/dir/file.wav", "abc", "nested/dir/file.wav"},
		{"workspace://only-id", "", ""},
		{"workspace://", "", ""},
		{"not-a-workspace-uri", "", ""},
	}

	for _, tt := range tests {
		gotID, gotPath := ParseWorkspaceURI(tt.uri)
		if gotID != tt.wantID || gotPath != tt.wantPath {
			t.Errorf("ParseWorkspaceURI(%q) = (%q, %q), want (%q, %q)",
				tt.uri, gotID, gotPath, tt.wantID, tt.wantPath)
		}
	}
}

func TestToLocalPath_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	localPath, cleanup, err := ToLocalPath(context.Background(), path)
	if err != nil {
		t.Fatalf("ToLocalPath: %v", err)
	}
	defer cleanup()

	if localPath != path {
		t.Errorf("localPath = %q, want %q", localPath, path)
	}
}

func TestToLocalPath_AbsolutePathNotFound(t *testing.T) {
	_, _, err := ToLocalPath(context.Background(), "/nonexistent/path/file.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestToLocalPath_DataURI(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("test data"))
	uri := "data:text/plain;base64," + payload

	localPath, cleanup, err := ToLocalPath(context.Background(), uri)
	if err != nil {
		t.Fatalf("ToLocalPath: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test data" {
		t.Errorf("data = %q, want %q", string(data), "test data")
	}
}

func TestParseAttachURI(t *testing.T) {
	uploader, fileID := parseAttachURI("attach://__yao.attachment/abc123")
	if uploader != "__yao.attachment" || fileID != "abc123" {
		t.Errorf("parseAttachURI = (%q, %q), want (__yao.attachment, abc123)", uploader, fileID)
	}
}
