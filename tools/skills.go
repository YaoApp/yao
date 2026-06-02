package tools

import "embed"

// SkillsFS contains the capability-grouped SKILL.md files for injection
// into sandbox workspaces. Each SKILL teaches the LLM how to use a group
// of system tools via `tai tool <name>`.
//
//go:embed skills
var SkillsFS embed.FS

// SystemPrompt is the shared content appended to both CLAUDE.md and AGENTS.md
// in sandbox workspaces. It provides environment variable documentation and
// the `tai tool` calling convention. Stored as a single source file to prevent
// content drift between the two runner instruction files.
//
//go:embed prompts/system-tools.md
var SystemPrompt []byte

// AgentsFS contains the agent definition files (e.g. a2a.md) for injection
// into sandbox workspaces at .claude/agents/. These define sub-agent behaviors
// that Claude Code can spawn via its native Agent tool.
//
//go:embed agents
var AgentsFS embed.FS
