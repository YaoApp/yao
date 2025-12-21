package assistant

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/agent/assistant/hook"
)

// scriptsMutex protects concurrent v8.Load calls and Scripts map access
var scriptsMutex sync.Mutex

// Execute execute the script
func (s *Script) Execute(ctx context.Context, method string, args ...interface{}) (interface{}, error) {
	return s.ExecuteWithAuthorized(ctx, method, nil, args...)
}

// ExecuteWithAuthorized execute the script with authorized information
func (s *Script) ExecuteWithAuthorized(ctx context.Context, method string, authorized map[string]interface{}, args ...interface{}) (interface{}, error) {
	if s == nil || s.Script == nil {
		return nil, nil
	}

	scriptCtx, err := s.NewContext("", nil)
	if err != nil {
		return nil, err
	}
	defer scriptCtx.Close()

	// Set authorized information if available
	if authorized != nil {
		scriptCtx.WithAuthorized(authorized)
	}

	// Call the method with provided arguments as-is
	result, err := scriptCtx.CallWith(ctx, method, args...)

	// Return error as-is (including "not defined" errors)
	return result, err
}

// LoadScripts loads all scripts from a src directory path
// It scans for .ts and .js files (excluding index.ts which is the hook script)
// Returns the HookScript and a map of other scripts
func LoadScripts(srcDir string) (*hook.Script, map[string]*Script, error) {
	// Check if src directory exists
	exists, err := application.App.Exists(srcDir)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, nil // No src directory
	}

	var hookScript *hook.Script
	scripts := make(map[string]*Script)
	var loadErr error

	// Walk through src directory to find all script files
	exts := []string{"*.ts", "*.js"}
	err = application.App.Walk(srcDir, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		// file is the full path, root is srcDir
		// Get relative path for determining if it's index
		relPath := strings.TrimPrefix(file, root+"/")

		// Skip test files (*_test.ts, *_test.js)
		if strings.HasSuffix(relPath, "_test.ts") || strings.HasSuffix(relPath, "_test.js") {
			return nil
		}

		// Check if it's the root index.ts/js (hook script)
		// Only src/index.ts is the hook script, not src/foo/index.ts
		isRootIndex := relPath == "index.ts" || relPath == "index.js"

		if isRootIndex {
			scriptsMutex.Lock()
			script, err := loadScriptFile(file)
			scriptsMutex.Unlock()
			if err != nil {
				loadErr = fmt.Errorf("failed to load hook script %s: %w", file, err)
				return loadErr
			}
			hookScript = script
		} else {
			// Generate script ID from relative path
			scriptID := generateScriptID(file, root)

			// Load the script (v8.Load is not thread-safe)
			scriptsMutex.Lock()
			script, err := loadScriptV8(file)
			if err != nil {
				scriptsMutex.Unlock()
				loadErr = fmt.Errorf("failed to load script %s: %w", file, err)
				return loadErr
			}
			scripts[scriptID] = &Script{Script: script}
			scriptsMutex.Unlock()
		}

		return nil
	}, exts...)

	if loadErr != nil {
		return nil, nil, loadErr
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to walk src directory: %w", err)
	}

	return hookScript, scripts, nil
}

// generateScriptID generates a script ID from file path
// Example: assistants/test/src/foo/bar/test.ts -> foo.bar.test
func generateScriptID(filePath string, srcDir string) string {
	// Normalize path separators
	filePath = filepath.ToSlash(filePath)
	srcDir = filepath.ToSlash(srcDir)

	// Remove src directory prefix
	relPath := strings.TrimPrefix(filePath, srcDir+"/")
	relPath = strings.TrimPrefix(relPath, "/")

	// Remove file extension
	relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))

	// Replace path separators with dots
	scriptID := strings.ReplaceAll(relPath, "/", ".")

	return scriptID
}

// loadScriptFile loads a hook script from file
func loadScriptFile(file string) (*hook.Script, error) {
	id := makeScriptID(file, "")
	script, err := v8.Load(file, id)
	if err != nil {
		return nil, err
	}

	return &hook.Script{Script: script}, nil
}

// loadScriptFromSource loads a script from source code
// Uses MakeScriptInMemory which supports TypeScript syntax without file resolution
func loadScriptFromSource(source string, file string) (*v8.Script, error) {
	script, err := v8.MakeScriptInMemory([]byte(source), file, 5*time.Second, true)
	if err != nil {
		return nil, err
	}
	return script, nil
}

// loadScriptV8 loads a v8.Script from file (used for non-hook scripts)
func loadScriptV8(file string) (*v8.Script, error) {
	id := makeScriptID(file, "")
	script, err := v8.Load(file, id)
	if err != nil {
		return nil, err
	}
	return script, nil
}

// makeScriptID generates the script ID for v8.Load
// Converts file path to a dot-separated ID
// Example: assistants/tests/fullfields/src/index.ts -> assistants.tests.fullfields.src.index
func makeScriptID(file string, root string) string {
	// Remove root prefix if provided
	id := file
	if root != "" {
		id = strings.TrimPrefix(file, root+"/")
	}

	// Remove extension
	id = strings.TrimSuffix(id, filepath.Ext(id))

	// Replace path separators with dots
	id = strings.ReplaceAll(id, "/", ".")
	id = strings.ReplaceAll(id, string(filepath.Separator), ".")

	return id
}

// LoadScriptsFromData loads scripts from data map
// Handles script/scripts/source fields with priority: script > scripts > source > file system
func LoadScriptsFromData(data map[string]interface{}, assistantID string) (*hook.Script, map[string]*Script, error) {
	// Priority 1: script field (hook script from string source)
	if data["script"] != nil {
		switch v := data["script"].(type) {
		case string:
			file := fmt.Sprintf("assistants/%s/src/index.ts", assistantID)
			script, err := loadScriptFromSource(v, file)
			if err != nil {
				return nil, nil, err
			}
			hookScript := &hook.Script{Script: script}

			// Load other scripts if provided
			scripts, err := loadScriptsField(data["scripts"])
			if err != nil {
				return nil, nil, err
			}

			return hookScript, scripts, nil
		case *hook.Script:
			scripts, err := loadScriptsField(data["scripts"])
			if err != nil {
				return nil, nil, err
			}
			return v, scripts, nil
		case *v8.Script:
			scripts, err := loadScriptsField(data["scripts"])
			if err != nil {
				return nil, nil, err
			}
			return &hook.Script{Script: v}, scripts, nil
		}
	}

	// Priority 2: scripts field (map of scripts)
	if data["scripts"] != nil {
		// First extract index if present
		var hookScript *hook.Script
		if scriptsMap, ok := data["scripts"].(map[string]interface{}); ok {
			if indexSource, hasIndex := scriptsMap["index"]; hasIndex {
				switch v := indexSource.(type) {
				case string:
					file := fmt.Sprintf("assistants/%s/src/index.ts", assistantID)
					script, err := loadScriptFromSource(v, file)
					if err != nil {
						return nil, nil, err
					}
					hookScript = &hook.Script{Script: script}
				case *Script:
					hookScript = &hook.Script{Script: v.Script}
				case *v8.Script:
					hookScript = &hook.Script{Script: v}
				}
			}
		}

		// Then load other scripts (loadScriptsField automatically filters out index)
		scripts, err := loadScriptsField(data["scripts"])
		if err != nil {
			return nil, nil, err
		}

		return hookScript, scripts, nil
	}

	// Priority 3: source field (legacy hook script from source)
	if source, ok := data["source"].(string); ok && source != "" {
		script, err := loadSource(source, assistantID)
		if err != nil {
			return nil, nil, err
		}
		return script, nil, nil
	}

	// Priority 4: file system (scan src directory)
	srcDir := fmt.Sprintf("assistants/%s/src", assistantID)
	return LoadScripts(srcDir)
}

// loadScriptsField parses scripts field from data
// Note: "index" is always filtered out as it's reserved for HookScript
func loadScriptsField(scriptsData interface{}) (map[string]*Script, error) {
	if scriptsData == nil {
		return nil, nil
	}

	scripts := make(map[string]*Script)

	switch v := scriptsData.(type) {
	case map[string]*Script:
		for id, script := range v {
			if id == "index" {
				continue // Skip index
			}
			scripts[id] = script
		}
		return scripts, nil
	case map[string]*v8.Script:
		for id, script := range v {
			if id == "index" {
				continue // Skip index
			}
			scripts[id] = &Script{Script: script}
		}
		return scripts, nil
	case map[string]interface{}:
		for id, item := range v {
			if id == "index" {
				continue // Skip index
			}
			switch s := item.(type) {
			case *Script:
				scripts[id] = s
			case *v8.Script:
				scripts[id] = &Script{Script: s}
			case string:
				// Load script from source code
				file := fmt.Sprintf("script_%s", id)
				script, err := loadScriptFromSource(s, file)
				if err != nil {
					return nil, fmt.Errorf("failed to load script %s: %w", id, err)
				}
				scripts[id] = &Script{Script: script}
			}
		}
		return scripts, nil
	}

	return nil, nil
}

// RegisterScripts registers all scripts as process handlers
// Handler naming: agents.<assistantID>.<scriptID>
func (ast *Assistant) RegisterScripts() error {
	if len(ast.Scripts) == 0 {
		return nil
	}

	assistantID := ast.ID
	handlers := make(map[string]process.Handler)

	for scriptID, script := range ast.Scripts {
		// Create handler for this script
		handlers[scriptID] = makeScriptHandler(script)
	}

	// Register the handler group dynamically
	groupName := fmt.Sprintf("agents.%s", assistantID)
	process.RegisterDynamicGroup(groupName, handlers)

	return nil
}

// UnregisterScripts unregisters all scripts from process handlers
func (ast *Assistant) UnregisterScripts() error {
	if len(ast.Scripts) == 0 {
		return nil
	}

	assistantID := ast.ID

	for scriptID := range ast.Scripts {
		handlerID := fmt.Sprintf("agents.%s.%s", strings.ToLower(assistantID), strings.ToLower(scriptID))
		delete(process.Handlers, handlerID)
	}

	return nil
}

// makeScriptHandler creates a process handler for a script
func makeScriptHandler(script *Script) process.Handler {
	return func(p *process.Process) interface{} {
		// Extract method name from process
		method := p.Method

		// Get arguments from process
		args := p.Args

		// Convert authorized info to map if available
		var authorized map[string]interface{}
		if p.Authorized != nil {
			authorized = p.Authorized.AuthorizedToMap()
		}

		// Execute the script with authorized information
		result, err := script.ExecuteWithAuthorized(p.Context, method, authorized, args...)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		return result
	}
}
