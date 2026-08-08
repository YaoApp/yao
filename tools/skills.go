package tools

import (
	"bytes"
	"embed"
	"text/template"
)

// SkillsFS contains the capability-grouped SKILL.md files for injection
// into sandbox workspaces. Each SKILL teaches the LLM how to use a group
// of system tools via `tai tool <name>`.
//
//go:embed skills
var SkillsFS embed.FS

// systemPromptTmpl is the raw Go template for the system prompt.
//
//go:embed prompts/system-tools.md.tmpl
var systemPromptTmpl string

// SystemPrompt is the rendered prompt with conservative defaults (no vision).
// Retained for backward compatibility with callers that don't need dynamic rendering.
var SystemPrompt = func() []byte {
	b, _ := RenderSystemPrompt(false)
	return b
}()

// promptData carries model capabilities into the system prompt template.
type promptData struct {
	HasVision bool
}

// RenderSystemPrompt renders the system-tools template with the given
// model capability.  When hasVision is true, the "Image Files" section
// that instructs the agent to use image_read is omitted (the model can
// see images natively).  When false or on error, the section is included
// as a conservative fallback.
func RenderSystemPrompt(hasVision bool) ([]byte, error) {
	tmpl, err := template.New("system-tools").Parse(systemPromptTmpl)
	if err != nil {
		return []byte(systemPromptTmpl), err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, promptData{HasVision: hasVision}); err != nil {
		return []byte(systemPromptTmpl), err
	}
	return buf.Bytes(), nil
}

// AgentsFS contains the agent definition files (e.g. a2a.md) for injection
// into sandbox workspaces at .claude/agents/. These define sub-agent behaviors
// that Claude Code can spawn via its native Agent tool.
//
//go:embed agents
var AgentsFS embed.FS
