package workspace

import (
	"runtime"
	"strings"
	"testing"
)

func TestParseHostURI(t *testing.T) {
	tests := []struct {
		raw    string
		scheme string
		path   string
		isHost bool
	}{
		{"local:///abs/path", "local", "abs/path", true},
		{"local:///home/user/file.txt", "local", "home/user/file.txt", true},
		{"local:///C:/Users/file.txt", "local", "C:/Users/file.txt", true},
		{"tmp:///rel/path", "tmp", "rel/path", true},
		{"tmp:///upload-123/file.bin", "tmp", "upload-123/file.bin", true},
		{"some/workspace/path", "", "some/workspace/path", false},
		{"file.txt", "", "file.txt", false},
		{"", "", "", false},
	}
	for _, tc := range tests {
		u := parseHostURI(tc.raw)
		if u.Scheme != tc.scheme {
			t.Errorf("parseHostURI(%q).Scheme = %q, want %q", tc.raw, u.Scheme, tc.scheme)
		}
		if u.Path != tc.path {
			t.Errorf("parseHostURI(%q).Path = %q, want %q", tc.raw, u.Path, tc.path)
		}
		if u.IsHost != tc.isHost {
			t.Errorf("parseHostURI(%q).IsHost = %v, want %v", tc.raw, u.IsHost, tc.isHost)
		}
	}
}

func TestResolveAbsHostPath_Local_Unix(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"home/user/file.txt", "/home/user/file.txt"},
		{"var/data", "/var/data"},
		{"single", "/single"},
	}
	for _, tc := range tests {
		got, err := resolveAbsHostPath(hostURI{Scheme: "local", Path: tc.path})
		if err != nil {
			t.Errorf("resolveAbsHostPath(local, %q): %v", tc.path, err)
			continue
		}
		if runtime.GOOS != "windows" {
			if got != tc.want {
				t.Errorf("resolveAbsHostPath(local, %q) = %q, want %q", tc.path, got, tc.want)
			}
		}
	}
}

func TestResolveAbsHostPath_Local_WindowsDriveLetter(t *testing.T) {
	tests := []struct {
		path string
	}{
		{"C:/Users/Max/file.txt"},
		{"D:/data/workspace"},
	}
	for _, tc := range tests {
		got, err := resolveAbsHostPath(hostURI{Scheme: "local", Path: tc.path})
		if err != nil {
			t.Errorf("resolveAbsHostPath(local, %q): %v", tc.path, err)
			continue
		}
		if got == "" {
			t.Errorf("resolveAbsHostPath(local, %q) returned empty", tc.path)
			continue
		}
		if strings.HasPrefix(got, "/") && runtime.GOOS != "windows" {
			t.Errorf("resolveAbsHostPath(local, %q) = %q, should not prepend / to drive letter path", tc.path, got)
		}
		if !strings.Contains(got, "Users") && strings.Contains(tc.path, "Users") {
			t.Errorf("resolveAbsHostPath(local, %q) = %q, path content lost", tc.path, got)
		}
	}
}

func TestResolveAbsHostPath_Tmp(t *testing.T) {
	got, err := resolveAbsHostPath(hostURI{Scheme: "tmp", Path: "upload-123/file.bin"})
	if err != nil {
		t.Fatalf("resolveAbsHostPath(tmp, normal): %v", err)
	}
	if !strings.Contains(got, "upload-123") {
		t.Errorf("resolveAbsHostPath(tmp) = %q, missing path segment", got)
	}
}

func TestResolveAbsHostPath_Tmp_Traversal(t *testing.T) {
	_, err := resolveAbsHostPath(hostURI{Scheme: "tmp", Path: "../etc/passwd"})
	if err == nil {
		t.Error("resolveAbsHostPath(tmp, traversal) should return error")
	}
}

func TestResolveAbsHostPath_NonHostURI(t *testing.T) {
	_, err := resolveAbsHostPath(hostURI{Scheme: "", Path: "some/workspace/path"})
	if err == nil {
		t.Error("resolveAbsHostPath(non-host) should return error")
	}
}
