package tools

import (
	"io/fs"
	"strings"
	"testing"
)

func TestSkillsFS_ContainsAllSkills(t *testing.T) {
	expected := map[string]bool{
		"skills/yao-web/SKILL.md":     false,
		"skills/yao-process/SKILL.md": false,
		"skills/yao-doc/SKILL.md":     false,
		"skills/yao-image/SKILL.md":   false,
		"skills/yao-audio/SKILL.md":   false,
		"skills/yao-agent/SKILL.md":   false,
		"skills/yao-board/SKILL.md":   false,
	}

	err := fs.WalkDir(SkillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if _, ok := expected[path]; ok {
			expected[path] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}

	for path, found := range expected {
		if !found {
			t.Errorf("expected file not found in SkillsFS: %s", path)
		}
	}
}

func TestSkillsFS_FrontmatterFields(t *testing.T) {
	skills := []struct {
		path string
		name string
	}{
		{"skills/yao-web/SKILL.md", "yao-web"},
		{"skills/yao-process/SKILL.md", "yao-process"},
		{"skills/yao-doc/SKILL.md", "yao-doc"},
		{"skills/yao-image/SKILL.md", "yao-image"},
		{"skills/yao-audio/SKILL.md", "yao-audio"},
		{"skills/yao-agent/SKILL.md", "yao-agent"},
		{"skills/yao-board/SKILL.md", "yao-board"},
	}

	for _, s := range skills {
		data, err := fs.ReadFile(SkillsFS, s.path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", s.path, err)
		}
		content := string(data)

		if !strings.Contains(content, "name: "+s.name) {
			t.Errorf("%s: missing 'name: %s' in frontmatter", s.path, s.name)
		}
		if !strings.Contains(content, "description:") {
			t.Errorf("%s: missing 'description:' in frontmatter", s.path)
		}
		if !strings.Contains(content, "ALWAYS invoke this skill") {
			t.Errorf("%s: description missing directive 'ALWAYS invoke this skill'", s.path)
		}
	}
}

func TestSystemPrompt_NonEmpty(t *testing.T) {
	if len(SystemPrompt) == 0 {
		t.Fatal("SystemPrompt is empty")
	}

	content := string(SystemPrompt)

	markers := []string{
		"Yao Sandbox Environment",
		"Yao System Tools",
		"tai tool",
		"$WORKDIR",
		"$CTX_SKILLS_DIR",
	}
	for _, m := range markers {
		if !strings.Contains(content, m) {
			t.Errorf("SystemPrompt missing expected marker: %q", m)
		}
	}
}

func TestRenderSystemPrompt_WithVision(t *testing.T) {
	result, err := RenderSystemPrompt(true)
	if err != nil {
		t.Fatalf("RenderSystemPrompt(true) error: %v", err)
	}
	content := string(result)
	if !strings.Contains(content, "### Image Files") {
		t.Error("with vision: should contain '### Image Files' section")
	}
	if !strings.Contains(content, "native vision") {
		t.Error("with vision: should mention native vision capability")
	}
	if strings.Contains(content, "always use `image_read`") {
		t.Error("with vision: should NOT contain image_read directive")
	}
	if !strings.Contains(content, "### Audio Files") {
		t.Error("with vision: should still contain '### Audio Files' section")
	}
}

func TestRenderSystemPrompt_WithoutVision(t *testing.T) {
	result, err := RenderSystemPrompt(false)
	if err != nil {
		t.Fatalf("RenderSystemPrompt(false) error: %v", err)
	}
	content := string(result)
	if !strings.Contains(content, "### Image Files") {
		t.Error("without vision: should contain '### Image Files' section")
	}
	if !strings.Contains(content, "always use `image_read`") {
		t.Error("without vision: should contain 'always use image_read' directive")
	}
}

func TestRenderSystemPrompt_DefaultFallback(t *testing.T) {
	result, err := RenderSystemPrompt(false)
	if err != nil {
		t.Fatalf("RenderSystemPrompt(false) error: %v", err)
	}
	content := string(result)
	if !strings.Contains(content, "### Image Files") {
		t.Error("default (false): should include Image Files section as conservative fallback")
	}

	// SystemPrompt global var uses false → should also contain the section
	if !strings.Contains(string(SystemPrompt), "### Image Files") {
		t.Error("SystemPrompt global: should include Image Files section (fallback)")
	}
}
