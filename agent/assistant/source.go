package assistant

import (
	"fmt"
	"strings"
	"time"

	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/assistant/hook"
)

// loadSource loads hook script from source code string
// The source field stores TypeScript code directly (but without imports)
// Priority: script field > source field (if script exists, source is ignored)
// Note: Uses MakeScriptInMemory which supports TypeScript syntax without file resolution.
func loadSource(source string, assistantID string) (*hook.Script, error) {
	if source == "" {
		return nil, nil
	}

	// Use virtual .ts path for TypeScript support
	// MakeScriptInMemory handles TypeScript transform without file system access
	virtualFile := fmt.Sprintf("assistants/%s/source.ts", strings.ReplaceAll(assistantID, ".", "/"))

	script, err := v8.MakeScriptInMemory([]byte(source), virtualFile, 5*time.Second, true)
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
