package task

import "fmt"

// extractRecentText extracts text content from the last N messages in ringBuffer
func extractRecentText(dc *DaemonContext, n int) []string {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	start := len(dc.ringBuffer) - n
	if start < 0 {
		start = 0
	}

	var texts []string
	for _, msg := range dc.ringBuffer[start:] {
		switch msg.Type {
		case "text", "error", "execute":
			if content, ok := msg.Props["content"].(string); ok && content != "" {
				role := "assistant"
				if r, ok := msg.Props["role"].(string); ok {
					role = r
				}
				texts = append(texts, fmt.Sprintf("[%s] %s", role, content))
			}
		}
	}
	return texts
}
