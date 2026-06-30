package task

import "fmt"

// extractRecentText merges streaming delta fragments in ringBuffer into complete
// messages (grouped by MessageID), then returns the last N complete texts.
func extractRecentText(dc *DaemonContext, n int) []string {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	type mergedMsg struct {
		role    string
		content string
	}

	var ordered []string
	merged := map[string]*mergedMsg{}

	for _, msg := range dc.ringBuffer {
		switch msg.Type {
		case "text":
			content, _ := msg.Props["content"].(string)
			if content == "" {
				continue
			}
			mid := msg.MessageID
			if mid == "" {
				mid = fmt.Sprintf("_anon_%d", len(ordered))
				ordered = append(ordered, mid)
				role := "assistant"
				if r, _ := msg.Props["role"].(string); r != "" {
					role = r
				}
				merged[mid] = &mergedMsg{role: role, content: content}
				continue
			}
			if m, exists := merged[mid]; exists {
				m.content += content
			} else {
				ordered = append(ordered, mid)
				role := "assistant"
				if r, _ := msg.Props["role"].(string); r != "" {
					role = r
				}
				merged[mid] = &mergedMsg{role: role, content: content}
			}

		case "execute":
			tool, _ := msg.Props["tool"].(string)
			output, _ := msg.Props["output"].(string)
			if tool == "" && output == "" {
				continue
			}
			mid := msg.MessageID
			if mid == "" {
				mid = fmt.Sprintf("_exec_%d", len(ordered))
			}
			if m, exists := merged[mid]; exists {
				if output != "" && m.content == "" {
					m.content = fmt.Sprintf("[tool:%s] %s", tool, truncate(output, 300))
				}
			} else {
				ordered = append(ordered, mid)
				txt := fmt.Sprintf("[tool:%s]", tool)
				if output != "" {
					txt = fmt.Sprintf("[tool:%s] %s", tool, truncate(output, 300))
				}
				merged[mid] = &mergedMsg{role: "assistant", content: txt}
			}

		case "error":
			content, _ := msg.Props["message"].(string)
			if content == "" {
				content, _ = msg.Props["content"].(string)
			}
			if content == "" {
				continue
			}
			mid := msg.MessageID
			if mid == "" {
				mid = fmt.Sprintf("_err_%d", len(ordered))
			}
			if _, exists := merged[mid]; !exists {
				ordered = append(ordered, mid)
				merged[mid] = &mergedMsg{role: "system", content: content}
			}
		}
	}

	// Take last N complete messages
	start := 0
	if len(ordered) > n {
		start = len(ordered) - n
	}

	var texts []string
	for _, mid := range ordered[start:] {
		m := merged[mid]
		if m.content != "" {
			texts = append(texts, fmt.Sprintf("[%s] %s", m.role, truncate(m.content, 2000)))
		}
	}
	return texts
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
