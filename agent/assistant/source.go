package assistant

import (
	"fmt"
	"time"

	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/assistant/hook"
)

// loadSource loads hook script from source code string
// The source field stores TypeScript code directly
// Priority: script field > source field (if script exists, source is ignored)
func loadSource(source string, assistantID string) (*hook.Script, error) {
	if source == "" {
		return nil, nil
	}

	// Generate a virtual file path for the script
	file := fmt.Sprintf("assistants/%s/source.ts", assistantID)

	script, err := v8.MakeScript([]byte(source), file, 5*time.Second, true)
	if err != nil {
		return nil, fmt.Errorf("failed to compile source script: %w", err)
	}

	return &hook.Script{Script: script}, nil
}

// TODO: Future enhancement - support multiple files merged with special comment delimiter
// Format: // file: index.ts
// This would allow splitting large scripts into multiple logical files while storing as single source
// func loadSourceMultiFile(source string, assistantID string) (*hook.Script, error) {
//     // Parse source by "// file: xxx.ts" delimiter
//     // Merge and compile
// }
