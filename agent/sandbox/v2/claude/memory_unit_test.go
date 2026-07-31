//go:build unit

package claude_test

import (
	"strings"
	"testing"

	claude "github.com/yaoapp/yao/agent/sandbox/v2/claude"
	"github.com/yaoapp/yao/tai/volume"
)

func TestBuildEnvironmentContextMD_ContainsWorkspaceID(t *testing.T) {
	md := claude.ExportBuildEnvironmentContextMD("ws-abc-123", "/workspace")

	if !strings.Contains(md, "ws-abc-123") {
		t.Fatal("workspace ID not found in output")
	}
	if !strings.Contains(md, "/workspace") {
		t.Fatal("workDir not found in output")
	}
	if !strings.Contains(md, "workspace://ws-abc-123/path/to/file") {
		t.Fatal("workspace:// link template not found")
	}
}

func TestBuildEnvironmentContextMD_HasValidFrontmatter(t *testing.T) {
	md := claude.ExportBuildEnvironmentContextMD("test-id", "/work")

	if !strings.HasPrefix(md, "---\n") {
		t.Fatal("missing opening frontmatter delimiter")
	}
	if !strings.Contains(md, "name: environment-context") {
		t.Fatal("missing name field")
	}
	if !strings.Contains(md, "type: project") {
		t.Fatal("missing type field")
	}
	if !strings.Contains(md, "description:") {
		t.Fatal("missing description field")
	}
}

func TestBuildEnvironmentContextMD_ContainsToolTable(t *testing.T) {
	md := claude.ExportBuildEnvironmentContextMD("id", "/w")

	for _, tool := range []string{"web_search", "skill_list", "board_list", "agent_call"} {
		if !strings.Contains(md, tool) {
			t.Errorf("tool %q not found in memory content", tool)
		}
	}
}

func TestBuildExtensionSkillsMD_EmptySkills(t *testing.T) {
	md := claude.ExportBuildExtensionSkillsMD(nil)
	if md != "" {
		t.Fatalf("expected empty string for nil skills, got %d bytes", len(md))
	}
}

func TestBuildExtensionSkillsMD_ListsAllSkills(t *testing.T) {
	skills := []volume.SkillInfo{
		{Name: "data-analysis", Description: "Analyze data and charts."},
		{Name: "deploy-helper", Description: "Deploy automation."},
	}
	md := claude.ExportBuildExtensionSkillsMD(skills)

	if !strings.Contains(md, "**data-analysis**") {
		t.Error("skill name data-analysis not found")
	}
	if !strings.Contains(md, "Analyze data and charts.") {
		t.Error("skill description not found")
	}
	if !strings.Contains(md, "**deploy-helper**") {
		t.Error("skill name deploy-helper not found")
	}
}

func TestBuildExtensionSkillsMD_HasValidFrontmatter(t *testing.T) {
	skills := []volume.SkillInfo{{Name: "test", Description: "Test skill"}}
	md := claude.ExportBuildExtensionSkillsMD(skills)

	if !strings.HasPrefix(md, "---\n") {
		t.Fatal("missing opening frontmatter delimiter")
	}
	if !strings.Contains(md, "name: extension-skills") {
		t.Fatal("missing name field")
	}
	if !strings.Contains(md, "type: reference") {
		t.Fatal("missing type field")
	}
}

func TestBuildExtensionSkillsMD_ContainsUsageInstructions(t *testing.T) {
	skills := []volume.SkillInfo{{Name: "x", Description: "y"}}
	md := claude.ExportBuildExtensionSkillsMD(skills)

	if !strings.Contains(md, "tai tool skill_list") {
		t.Error("missing skill_list usage hint")
	}
	if !strings.Contains(md, "$CTX_EXT_SKILLS_DIR") {
		t.Error("missing CTX_EXT_SKILLS_DIR reference")
	}
}

func TestBuildMemoryIndexMD_WithExtSkills(t *testing.T) {
	md := claude.ExportBuildMemoryIndexMD(true)

	if !strings.Contains(md, "environment-context.md") {
		t.Error("missing environment-context entry")
	}
	if !strings.Contains(md, "extension-skills.md") {
		t.Error("missing extension-skills entry when hasExtSkills=true")
	}
	if strings.Contains(md, "---") {
		t.Error("MEMORY.md must not contain YAML frontmatter")
	}
	for _, line := range strings.Split(strings.TrimSpace(md), "\n") {
		if len(line) >= 150 {
			t.Errorf("line exceeds 150 chars: %q", line)
		}
	}
}

func TestBuildMemoryIndexMD_WithoutExtSkills(t *testing.T) {
	md := claude.ExportBuildMemoryIndexMD(false)

	if !strings.Contains(md, "environment-context.md") {
		t.Error("missing environment-context entry")
	}
	if strings.Contains(md, "extension-skills.md") {
		t.Error("extension-skills entry should be absent when hasExtSkills=false")
	}
	if strings.Contains(md, "---") {
		t.Error("MEMORY.md must not contain YAML frontmatter")
	}
}
