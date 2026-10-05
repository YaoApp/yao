package shared

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"
)

const systemToolsMarker = "<!-- Yao System Tools (auto-injected) -->"
const chatMemoryMarker = "<!-- Chat Memory (auto-injected) -->"

// writerFS is the minimal filesystem interface needed by the injection helpers.
// workspace.FS satisfies this interface.
type writerFS interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(name string, perm os.FileMode) error
}

// InjectSystemSkills copies SKILL files from an embed.FS into the workspace.
// The skills parameter should be an embed.FS produced by `//go:embed skills`,
// where each file has a path like "skills/yao-web/SKILL.md". This function
// strips the "skills/" prefix and writes files into targetDir (e.g. ".claude/skills").
func InjectSystemSkills(ws writerFS, skills fs.FS, targetDir string) error {
	return fs.WalkDir(skills, "skills", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(p, "skills/")
		dst := path.Join(targetDir, rel)

		data, err := fs.ReadFile(skills, p)
		if err != nil {
			return err
		}

		dir := path.Dir(dst)
		if err := ws.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return ws.WriteFile(dst, data, 0644)
	})
}

// InjectAgentDefinitions copies agent definition files from an embed.FS into the
// workspace. The agents parameter should be an embed.FS produced by
// `//go:embed agents`, where each file has a path like "agents/a2a.md". This
// function strips the "agents/" prefix and writes files into targetDir
// (e.g. ".claude/agents").
func InjectAgentDefinitions(ws writerFS, agents fs.FS, targetDir string) error {
	return fs.WalkDir(agents, "agents", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(p, "agents/")
		dst := path.Join(targetDir, rel)

		data, err := fs.ReadFile(agents, p)
		if err != nil {
			return err
		}

		dir := path.Dir(dst)
		if err := ws.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return ws.WriteFile(dst, data, 0644)
	})
}

// AppendSystemPrompt injects content into a file in the workspace using an
// idempotent marker. If the marker already exists, the injected section is
// replaced with the new content (so updates propagate to existing sandboxes).
// If the file does not exist it is created with just the marker + content.
func AppendSystemPrompt(ws writerFS, filename string, content []byte) error {
	return appendWithMarker(ws, filename, systemToolsMarker, content)
}

// InjectChatMemory reads per-chat MEMORY.md, INSTRUCTION.md and PREFERENCE.md
// from .yao/<chatID>/ and appends them to the agent instructions file using
// an idempotent marker. Files that don't exist are silently skipped.
// Returns nil when chatID is empty or no files are found.
func InjectChatMemory(ws writerFS, chatID, agentsFile string) error {
	if chatID == "" {
		return nil
	}
	dir := ".yao/" + chatID
	var sections []byte
	for _, name := range []string{"INSTRUCTION.md", "PREFERENCE.md", "MEMORY.md"} {
		data, err := ws.ReadFile(dir + "/" + name)
		if err != nil {
			continue
		}
		if len(data) == 0 {
			continue
		}
		sections = append(sections, []byte("\n## "+name+"\n\n")...)
		sections = append(sections, data...)
		sections = append(sections, '\n')
	}
	if len(sections) == 0 {
		return nil
	}
	return appendWithMarker(ws, agentsFile, chatMemoryMarker, sections)
}

// appendWithMarker injects content into a file using an idempotent marker.
// If the marker already exists the injected section is replaced; otherwise
// it is appended after a separator. A missing file is created.
func appendWithMarker(ws writerFS, filename, marker string, content []byte) error {
	existing, err := ws.ReadFile(filename)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		header := []byte(marker + "\n\n")
		return ws.WriteFile(filename, append(header, content...), 0644)
	}

	idx := bytes.Index(existing, []byte(marker))
	if idx >= 0 {
		injected := append([]byte(marker+"\n\n"), content...)
		prefix := existing[:idx]
		merged := append(bytes.TrimRight(prefix, "\n\r\t "), []byte("\n\n---\n\n")...)
		if idx == 0 {
			merged = nil
		}
		return ws.WriteFile(filename, append(merged, injected...), 0644)
	}

	separator := []byte("\n\n---\n\n" + marker + "\n\n")
	merged := append(existing, append(separator, content...)...)
	return ws.WriteFile(filename, merged, 0644)
}
