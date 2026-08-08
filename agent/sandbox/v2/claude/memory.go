package claude

import (
	"fmt"
	"strings"

	"github.com/yaoapp/yao/tai/volume"
	"github.com/yaoapp/yao/tai/workspace"
)

// injectAutoMemory writes environment-context.md and extension-skills.md into
// the CC auto-memory directory. Best-effort: all errors are silently ignored.
func injectAutoMemory(ws workspace.FS, assistantID, workDir string, hasVision bool) {
	memDir := ".yao/assistants/" + assistantID + "/memory"
	_ = ws.MkdirAll(memDir, 0755)

	wsID, _ := ws.GetID()
	envContent := buildEnvironmentContextMD(wsID, workDir, hasVision)
	_ = ws.WriteFile(memDir+"/environment-context.md", []byte(envContent), 0644)

	skills, _ := ws.ListSkills(".yao/skills")
	if len(skills) > 0 {
		skillContent := buildExtensionSkillsMD(skills)
		_ = ws.WriteFile(memDir+"/extension-skills.md", []byte(skillContent), 0644)
	}

	// Seed MEMORY.md only if it does not exist yet.
	// CC maintains this file itself during sessions — overwriting would
	// destroy index entries CC added in previous conversations.
	memoryMDPath := memDir + "/MEMORY.md"
	if _, err := ws.Stat(memoryMDPath); err != nil {
		memoryMD := buildMemoryIndexMD(len(skills) > 0)
		_ = ws.WriteFile(memoryMDPath, []byte(memoryMD), 0644)
	}
}

func buildEnvironmentContextMD(workspaceID, workDir string, hasVision bool) string {
	var b strings.Builder
	b.WriteString(`---
name: environment-context
description: Current workspace ID, tai tool summary, and cross-workspace access. ALWAYS recall when outputting workspace:// links, using tai tools, or referencing other workspaces.
type: project
---

## Current Workspace

`)
	fmt.Fprintf(&b, "- **Workspace ID**: %s\n", workspaceID)
	fmt.Fprintf(&b, "- **Working Directory**: %s\n", workDir)
	b.WriteString(`
When referencing files for the user, use Markdown links: ` + "`[path/to/file](workspace://" + workspaceID + "/path/to/file)`" + `.
The client renders these as clickable links opening the file in the editor.

## Accessing Other Workspaces

Use ` + "`tai tool workspace_list`" + ` to discover available workspaces, then:
- ` + "`tai tool workspace_file_list --workspace_id <id> --path /`" + `
- ` + "`tai tool workspace_file_read --workspace_id <id> --path /file.txt`" + `
- ` + "`tai tool workspace_file_write --workspace_id <id> --path /file.txt --content \"...\"`" + `

## tai tool Summary

Calling convention: ` + "`tai tool <name> --param value [--param2 value2 ...]`" + `

| Tool | Description |
|------|-------------|
| web_search | Search the web for real-time info |
| web_fetch | Fetch and read a web page |
| process_call | Execute a Yao Process |
| process_allowed | Check allowed processes |
| doc_list | Search/list process docs |
| doc_inspect | Get detailed process docs |
| image_read | Analyze images (vision) |
| image_generate | Generate images from text |
| agent_list | List available agents |
| agent_call | Call another AI expert |
| secret_list | List secret names |
| secret_read | Read a secret value |
| board_list | List kanban boards |
| task_list | List/filter tasks |
| workspace_list | List all workspaces |
| workspace_get | Get workspace details |
| workspace_file_list | List files in a workspace |
| workspace_file_read | Read file from workspace |
| workspace_file_write | Write file to workspace |
| workspace_git_config | Get or set workspace Git configuration |
| workspace_git_credential | Manage workspace HTTPS Git credentials |
| workspace_ssh_key | Manage workspace SSH keys |
| clip_write | Store a content clip |
| clip_read | Read a clip by ID |
| skill_list | List installed skills (filter by type: system/assistant/extension) |

Skills in ` + "`$HOME/.claude/skills/`" + ` are auto-loaded with full parameter docs.
`)
	if hasVision {
		b.WriteString("\n**Image files**: You have native vision. Use the `Read` tool to view images directly.\n")
	} else {
		b.WriteString("\n**Image files**: Your model does NOT have vision. Use `tai tool image_read --image_path <path>` to analyze images.\n")
	}
	return b.String()
}

func buildMemoryIndexMD(hasExtSkills bool) string {
	var b strings.Builder
	b.WriteString("- [Environment Context](environment-context.md) -- workspace ID, tai tools, cross-workspace access\n")
	if hasExtSkills {
		b.WriteString("- [Extension Skills](extension-skills.md) -- installed task extensions and skill creation guide\n")
	}
	return b.String()
}

func buildExtensionSkillsMD(skills []volume.SkillInfo) string {
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`---
name: extension-skills
description: Installed task extension skills and their triggers. Recall when user asks about available tools, skills, capabilities, task extensions, or what you can do.
type: reference
---

Extension skills directory: $CTX_EXT_SKILLS_DIR

Installed skills:
`)
	for _, s := range skills {
		fmt.Fprintf(&b, "- **%s**: %s\n", s.Name, s.Description)
	}
	b.WriteString(`
To get the latest full list: ` + "`tai tool skill_list '{}'`" + `
To read a skill: ` + "`cat <path>`" + ` then follow its instructions.
To create a new skill: place ` + "`<name>/SKILL.md`" + ` under $CTX_EXT_SKILLS_DIR.
`)
	return b.String()
}
