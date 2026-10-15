package tai

import (
	"encoding/json"
	"strings"
)

// extractToolSummary returns a short human-readable summary from tool
// name and input JSON, used for the ExecuteProps "summary" field.
func extractToolSummary(toolName string, inputJSON string) string {
	if inputJSON == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &obj); err != nil {
		return ""
	}

	switch strings.ToLower(toolName) {
	case "bash", "execute":
		if desc, ok := obj["description"].(string); ok && desc != "" {
			return truncateStr(desc, 80)
		}
		if cmd, ok := obj["command"].(string); ok {
			return truncateStr(cmd, 80)
		}
	case "write", "create":
		if fp, ok := obj["file_path"].(string); ok {
			return fp
		}
	case "read":
		if fp, ok := obj["file_path"].(string); ok {
			return fp
		}
	case "edit":
		if fp, ok := obj["file_path"].(string); ok {
			return fp
		}
	case "glob", "grep", "list":
		if p, ok := obj["pattern"].(string); ok {
			return truncateStr(p, 80)
		}
		if p, ok := obj["path"].(string); ok {
			return truncateStr(p, 80)
		}
	case "skill":
		if n, ok := obj["name"].(string); ok {
			return truncateStr(n, 80)
		}
		if d, ok := obj["description"].(string); ok {
			return truncateStr(d, 80)
		}
	}

	for _, key := range []string{"description", "path", "file_path", "command", "url", "query", "prompt", "task", "name"} {
		if v, ok := obj[key].(string); ok {
			return truncateStr(v, 80)
		}
	}
	return ""
}

// truncateStr trims and shortens a string to max characters.
func truncateStr(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
