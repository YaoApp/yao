//go:build unit

package shared_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
)

func TestInjectSystemSkills_CopiesAllFiles(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	skills := fstest.MapFS{
		"skills/yao-web/SKILL.md":     {Data: []byte("web skill")},
		"skills/yao-process/SKILL.md": {Data: []byte("process skill")},
		"skills/yao-doc/SKILL.md":     {Data: []byte("doc skill")},
	}

	if err := shared.InjectSystemSkills(ws, skills, ".claude/skills"); err != nil {
		t.Fatalf("InjectSystemSkills: %v", err)
	}

	for _, tc := range []struct {
		path string
		want string
	}{
		{".claude/skills/yao-web/SKILL.md", "web skill"},
		{".claude/skills/yao-process/SKILL.md", "process skill"},
		{".claude/skills/yao-doc/SKILL.md", "doc skill"},
	} {
		data, err := os.ReadFile(filepath.Join(dir, tc.path))
		if err != nil {
			t.Errorf("ReadFile(%s): %v", tc.path, err)
			continue
		}
		if string(data) != tc.want {
			t.Errorf("%s = %q, want %q", tc.path, data, tc.want)
		}
	}
}

func TestInjectSystemSkills_EmptyFS(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	skills := fstest.MapFS{
		"skills": {Mode: fs.ModeDir},
	}

	if err := shared.InjectSystemSkills(ws, skills, ".claude/skills"); err != nil {
		t.Fatalf("InjectSystemSkills: %v", err)
	}
}

func TestAppendSystemPrompt_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	content := []byte("## Yao System Tools\ntai tool ...")
	if err := shared.AppendSystemPrompt(ws, "CLAUDE.md", content); err != nil {
		t.Fatalf("AppendSystemPrompt: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)
	if got == "" {
		t.Fatal("file should not be empty")
	}
	assertContains(t, got, shared.ExportSystemToolsMarker)
	assertContains(t, got, "Yao System Tools")
}

func TestAppendSystemPrompt_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	existing := []byte("# My Project\n\nExisting content.\n")
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), existing, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	content := []byte("## System Tools\n")
	if err := shared.AppendSystemPrompt(ws, "CLAUDE.md", content); err != nil {
		t.Fatalf("AppendSystemPrompt: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)
	assertContains(t, got, "My Project")
	assertContains(t, got, shared.ExportSystemToolsMarker)
	assertContains(t, got, "System Tools")
}

func TestAppendSystemPrompt_Idempotent(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	content := []byte("## Yao System Tools\n")

	if err := shared.AppendSystemPrompt(ws, "AGENTS.md", content); err != nil {
		t.Fatalf("first call: %v", err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))

	if err := shared.AppendSystemPrompt(ws, "AGENTS.md", content); err != nil {
		t.Fatalf("second call: %v", err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))

	if string(first) != string(second) {
		t.Errorf("second call modified the file (not idempotent):\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestAppendSystemPrompt_UpdatesExistingContent(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	oldContent := []byte("## Old Tools\nweb_search only\n")
	if err := shared.AppendSystemPrompt(ws, "CLAUDE.md", oldContent); err != nil {
		t.Fatalf("first call: %v", err)
	}

	newContent := []byte("## Updated Tools\nweb_search + image_read\n")
	if err := shared.AppendSystemPrompt(ws, "CLAUDE.md", newContent); err != nil {
		t.Fatalf("second call: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	got := string(data)
	assertContains(t, got, "image_read")
	assertContains(t, got, shared.ExportSystemToolsMarker)
	if strings.Contains(got, "Old Tools") {
		t.Error("old content should have been replaced")
	}
}

func TestAppendSystemPrompt_UpdatesPreservesUserContent(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	userContent := []byte("# My Project\n\nUser notes.\n")
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), userContent, 0644); err != nil {
		t.Fatal(err)
	}

	oldContent := []byte("## Old Tools\n")
	if err := shared.AppendSystemPrompt(ws, "CLAUDE.md", oldContent); err != nil {
		t.Fatal(err)
	}

	newContent := []byte("## Updated Tools\nimage_read added\n")
	if err := shared.AppendSystemPrompt(ws, "CLAUDE.md", newContent); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	got := string(data)
	assertContains(t, got, "My Project")
	assertContains(t, got, "User notes")
	assertContains(t, got, "image_read")
	if strings.Contains(got, "Old Tools") {
		t.Error("old injected content should have been replaced")
	}
}

// --- InjectChatMemory tests ---

func TestInjectChatMemory_EmptyChatID(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)
	if err := shared.InjectChatMemory(ws, "", "AGENTS.md"); err != nil {
		t.Fatalf("expected nil for empty chatID, got: %v", err)
	}
}

func TestInjectChatMemory_NoFilesExist(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)
	if err := shared.InjectChatMemory(ws, "chat-123", "AGENTS.md"); err != nil {
		t.Fatalf("expected nil when no memory files exist, got: %v", err)
	}
}

func TestInjectChatMemory_InjectsAllThreeFiles(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	chatDir := filepath.Join(dir, ".yao", "chat-abc")
	if err := os.MkdirAll(chatDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(chatDir, "INSTRUCTION.md"), []byte("Be concise."), 0644)
	os.WriteFile(filepath.Join(chatDir, "PREFERENCE.md"), []byte("Use tabs."), 0644)
	os.WriteFile(filepath.Join(chatDir, "MEMORY.md"), []byte("Built feature X."), 0644)

	if err := shared.InjectChatMemory(ws, "chat-abc", "AGENTS.md"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	got := string(data)
	assertContains(t, got, shared.ExportChatMemoryMarker)
	assertContains(t, got, "INSTRUCTION.md")
	assertContains(t, got, "Be concise.")
	assertContains(t, got, "PREFERENCE.md")
	assertContains(t, got, "Use tabs.")
	assertContains(t, got, "MEMORY.md")
	assertContains(t, got, "Built feature X.")
}

func TestInjectChatMemory_SkipsEmptyFiles(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	chatDir := filepath.Join(dir, ".yao", "chat-empty")
	os.MkdirAll(chatDir, 0755)
	os.WriteFile(filepath.Join(chatDir, "MEMORY.md"), []byte("progress note"), 0644)
	os.WriteFile(filepath.Join(chatDir, "INSTRUCTION.md"), []byte(""), 0644)

	if err := shared.InjectChatMemory(ws, "chat-empty", "AGENTS.md"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	got := string(data)
	assertContains(t, got, "progress note")
	if strings.Contains(got, "INSTRUCTION.md") {
		t.Error("empty INSTRUCTION.md should be skipped")
	}
}

func TestInjectChatMemory_Idempotent(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	chatDir := filepath.Join(dir, ".yao", "chat-idem")
	os.MkdirAll(chatDir, 0755)
	os.WriteFile(filepath.Join(chatDir, "MEMORY.md"), []byte("fact A"), 0644)

	shared.InjectChatMemory(ws, "chat-idem", "AGENTS.md")
	first, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))

	shared.InjectChatMemory(ws, "chat-idem", "AGENTS.md")
	second, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))

	if string(first) != string(second) {
		t.Error("InjectChatMemory is not idempotent")
	}
}

func TestInjectChatMemory_CoexistsWithSystemPrompt(t *testing.T) {
	dir := t.TempDir()
	ws := newDirFS(dir)

	shared.AppendSystemPrompt(ws, "AGENTS.md", []byte("## System Tools\n"))

	chatDir := filepath.Join(dir, ".yao", "chat-coexist")
	os.MkdirAll(chatDir, 0755)
	os.WriteFile(filepath.Join(chatDir, "MEMORY.md"), []byte("user note"), 0644)

	if err := shared.InjectChatMemory(ws, "chat-coexist", "AGENTS.md"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	got := string(data)
	assertContains(t, got, shared.ExportSystemToolsMarker)
	assertContains(t, got, shared.ExportChatMemoryMarker)
	assertContains(t, got, "System Tools")
	assertContains(t, got, "user note")
}

// --- Tests for attachments.go pure functions ---

func TestExtensionFromContentType(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/svg+xml", ".svg"},
		{"application/pdf", ".pdf"},
		{"text/plain", ".txt"},
		{"text/html", ".html"},
		{"text/css", ".css"},
		{"text/javascript", ".js"},
		{"application/javascript", ".js"},
		{"application/json", ".json"},
		{"application/zip", ".zip"},
		{"application/octet-stream", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := shared.ExtensionFromContentType(tc.input)
		if got != tc.want {
			t.Errorf("ExtensionFromContentType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatFileSize(t *testing.T) {
	cases := []struct {
		input int
		want  string
	}{
		{0, "0B"},
		{512, "512B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1024 * 1024, "1.0MB"},
		{1024*1024 + 512*1024, "1.5MB"},
		{10 * 1024 * 1024, "10.0MB"},
	}
	for _, tc := range cases {
		got := shared.FormatFileSize(tc.input)
		if got != tc.want {
			t.Errorf("FormatFileSize(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- Helpers ---

func assertContains(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("string does not contain %q:\n%s", sub, s)
	}
}

type dirFS struct {
	root string
}

func newDirFS(root string) *dirFS { return &dirFS{root: root} }

func (d *dirFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.root, name))
}

func (d *dirFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filepath.Join(d.root, name), data, perm)
}

func (d *dirFS) MkdirAll(name string, perm os.FileMode) error {
	return os.MkdirAll(filepath.Join(d.root, name), perm)
}
