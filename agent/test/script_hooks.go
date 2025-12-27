package test

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/context"
	"rogchap.com/v8go"
)

// HookExecutor executes before/after scripts from *_test.ts files
// Scripts are loaded via V8 and executed directly, not via Process()
type HookExecutor struct {
	verbose      bool
	output       *OutputWriter
	loadedDirs   map[string]bool // Track which directories have been loaded
	agentContext *context.Context
	opts         *Options // Test options (includes ContextData from --ctx)
}

// NewHookExecutor creates a new hook executor
func NewHookExecutor(verbose bool) *HookExecutor {
	return &HookExecutor{
		verbose:    verbose,
		output:     NewOutputWriter(verbose),
		loadedDirs: make(map[string]bool),
	}
}

// SetAgentContext sets the agent context for script execution
func (h *HookExecutor) SetAgentContext(ctx *context.Context) {
	h.agentContext = ctx
}

// SetOptions sets the test options for hook execution
func (h *HookExecutor) SetOptions(opts *Options) {
	h.opts = opts
}

// HookRef represents a parsed hook reference
// Format: "src/env_test.ts:Before" or just "Before" (uses default test file)
type HookRef struct {
	ScriptFile string // e.g., "env_test.ts"
	Function   string // e.g., "Before"
}

// ParseHookRef parses a hook reference string
// Formats:
//   - "Before" -> uses first *_test.ts file found
//   - "env_test.Before" -> uses src/env_test.ts
//   - "src/env_test.Before" -> uses src/env_test.ts
func ParseHookRef(ref string) (*HookRef, error) {
	if ref == "" {
		return nil, fmt.Errorf("empty hook reference")
	}

	// Split by last dot to get function name
	lastDot := strings.LastIndex(ref, ".")
	if lastDot == -1 {
		// Just function name, will use default test file
		return &HookRef{
			ScriptFile: "", // Will be resolved later
			Function:   ref,
		}, nil
	}

	scriptPart := ref[:lastDot]
	funcName := ref[lastDot+1:]

	// Normalize script file name
	scriptFile := scriptPart
	if !strings.HasSuffix(scriptFile, "_test") {
		scriptFile += "_test"
	}
	scriptFile += ".ts"

	// Remove "src/" prefix if present
	scriptFile = strings.TrimPrefix(scriptFile, "src/")

	return &HookRef{
		ScriptFile: scriptFile,
		Function:   funcName,
	}, nil
}

// LoadTestScripts loads all *_test.ts scripts from the agent's src directory
// Returns the script IDs that were loaded
func (h *HookExecutor) LoadTestScripts(agentPath string) ([]string, error) {
	srcDir := filepath.Join(agentPath, "src")

	// Convert to relative path for application.App
	// application.App expects paths relative to YAO_ROOT
	relSrcDir := srcDir
	if application.App != nil {
		if rel, err := filepath.Rel(application.App.Root(), srcDir); err == nil {
			relSrcDir = rel
		}
	}

	// Check if already loaded (use absolute path as key)
	// No logging for already loaded - this is normal and happens frequently
	if h.loadedDirs[srcDir] {
		return nil, nil
	}

	// Check if src directory exists
	exists, err := application.App.Exists(relSrcDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil // No src directory, not an error
	}

	var loadedScripts []string
	exts := []string{"*_test.ts", "*_test.js"}

	err = application.App.Walk(relSrcDir, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		// Only load *_test.ts/js files
		base := filepath.Base(file)
		if !strings.HasSuffix(base, "_test.ts") && !strings.HasSuffix(base, "_test.js") {
			return nil
		}

		// Generate script ID (use relative path for consistency)
		scriptID := generateHookScriptID(file, relSrcDir)

		// Load the script (file path from Walk is relative to App root)
		_, err := v8.Load(file, scriptID)
		if err != nil {
			if h.verbose {
				h.output.Warning("Failed to load hook script %s: %v", base, err)
			}
			return nil // Continue loading other scripts
		}

		loadedScripts = append(loadedScripts, scriptID)
		return nil
	}, exts...)

	if err != nil {
		return nil, fmt.Errorf("failed to walk src directory: %w", err)
	}

	h.loadedDirs[srcDir] = true

	// Log summary only once when scripts are first loaded
	if h.verbose && len(loadedScripts) > 0 {
		h.output.Verbose("Loaded %d hook scripts from %s", len(loadedScripts), relSrcDir)
	}

	return loadedScripts, nil
}

// generateHookScriptID generates a script ID for hook scripts
// Example: assistants/test/src/env_test.ts -> hook.env_test
func generateHookScriptID(filePath string, srcDir string) string {
	filePath = filepath.ToSlash(filePath)
	srcDir = filepath.ToSlash(srcDir)

	relPath := strings.TrimPrefix(filePath, srcDir+"/")
	relPath = strings.TrimPrefix(relPath, "/")
	relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))

	return "hook." + strings.ReplaceAll(relPath, "/", ".")
}

// FindTestScript finds a loaded test script by pattern
// If scriptFile is empty, returns the first *_test script found
func (h *HookExecutor) FindTestScript(scriptFile string) (*v8.Script, string, error) {
	if scriptFile != "" {
		// Look for specific script
		scriptID := "hook." + strings.TrimSuffix(scriptFile, ".ts")
		scriptID = strings.TrimSuffix(scriptID, ".js")

		if script, ok := v8.Scripts[scriptID]; ok {
			return script, scriptID, nil
		}
		return nil, "", fmt.Errorf("hook script not found: %s (id: %s)", scriptFile, scriptID)
	}

	// Find first *_test script
	for id, script := range v8.Scripts {
		if strings.HasPrefix(id, "hook.") && strings.Contains(id, "_test") {
			return script, id, nil
		}
	}

	return nil, "", fmt.Errorf("no hook test script found")
}

// ExecuteBefore executes a Before function from a test script
func (h *HookExecutor) ExecuteBefore(ref string, testCase *Case, agentPath string) (interface{}, error) {
	hookRef, err := ParseHookRef(ref)
	if err != nil {
		return nil, err
	}

	// Ensure scripts are loaded
	if _, err := h.LoadTestScripts(agentPath); err != nil {
		return nil, fmt.Errorf("failed to load test scripts: %w", err)
	}

	// Find the script
	script, scriptID, err := h.FindTestScript(hookRef.ScriptFile)
	if err != nil {
		return nil, err
	}

	if h.verbose {
		h.output.Verbose("Executing %s from %s", hookRef.Function, scriptID)
	}

	// Execute the function
	return h.executeHookFunction(script, hookRef.Function, testCase, nil, nil)
}

// ExecuteAfter executes an After function from a test script
func (h *HookExecutor) ExecuteAfter(ref string, testCase *Case, result *Result, beforeData interface{}, agentPath string) error {
	hookRef, err := ParseHookRef(ref)
	if err != nil {
		return err
	}

	// Ensure scripts are loaded
	if _, err := h.LoadTestScripts(agentPath); err != nil {
		return fmt.Errorf("failed to load test scripts: %w", err)
	}

	// Find the script
	script, scriptID, err := h.FindTestScript(hookRef.ScriptFile)
	if err != nil {
		return err
	}

	if h.verbose {
		h.output.Verbose("Executing %s from %s", hookRef.Function, scriptID)
	}

	// Execute the function
	_, err = h.executeHookFunction(script, hookRef.Function, testCase, result, beforeData)
	return err
}

// ExecuteBeforeAll executes a BeforeAll function
func (h *HookExecutor) ExecuteBeforeAll(ref string, testCases []*Case, agentPath string) (interface{}, error) {
	hookRef, err := ParseHookRef(ref)
	if err != nil {
		return nil, err
	}

	// Ensure scripts are loaded
	if _, err := h.LoadTestScripts(agentPath); err != nil {
		return nil, fmt.Errorf("failed to load test scripts: %w", err)
	}

	// Find the script
	script, scriptID, err := h.FindTestScript(hookRef.ScriptFile)
	if err != nil {
		return nil, err
	}

	if h.verbose {
		h.output.Verbose("Executing %s from %s", hookRef.Function, scriptID)
	}

	// Execute with test cases array
	return h.executeHookFunctionWithCases(script, hookRef.Function, testCases)
}

// ExecuteAfterAll executes an AfterAll function
func (h *HookExecutor) ExecuteAfterAll(ref string, results []*Result, beforeData interface{}, agentPath string) error {
	hookRef, err := ParseHookRef(ref)
	if err != nil {
		return err
	}

	// Ensure scripts are loaded
	if _, err := h.LoadTestScripts(agentPath); err != nil {
		return fmt.Errorf("failed to load test scripts: %w", err)
	}

	// Find the script
	script, scriptID, err := h.FindTestScript(hookRef.ScriptFile)
	if err != nil {
		return err
	}

	if h.verbose {
		h.output.Verbose("Executing %s from %s", hookRef.Function, scriptID)
	}

	// Execute with results array
	_, err = h.executeHookFunctionWithResults(script, hookRef.Function, results, beforeData)
	return err
}

// executeHookFunction executes a hook function with test case context
func (h *HookExecutor) executeHookFunction(script *v8.Script, funcName string, testCase *Case, result *Result, beforeData interface{}) (interface{}, error) {
	// Create script context
	scriptCtx, err := script.NewContext("", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create script context: %w", err)
	}
	defer scriptCtx.Close()

	v8ctx := scriptCtx.Context

	// Set share data
	if err := h.setShareData(v8ctx); err != nil {
		return nil, err
	}

	// Get the function
	global := v8ctx.Global()
	fnValue, err := global.Get(funcName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function %s: %w", funcName, err)
	}

	if fnValue.IsUndefined() || fnValue.IsNull() {
		return nil, fmt.Errorf("function %s not defined", funcName)
	}

	if !fnValue.IsFunction() {
		return nil, fmt.Errorf("%s is not a function", funcName)
	}

	fn, err := fnValue.AsFunction()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to function: %w", err)
	}

	// Build arguments
	args, err := h.buildHookArgs(v8ctx, testCase, result, beforeData)
	if err != nil {
		return nil, err
	}

	// Convert to v8go.Valuer slice for Call
	valuerArgs := make([]v8go.Valuer, len(args))
	for i, arg := range args {
		valuerArgs[i] = arg
	}

	// Call the function
	jsResult, err := fn.Call(global, valuerArgs...)
	if err != nil {
		return nil, fmt.Errorf("hook function %s failed: %w", funcName, err)
	}

	// Convert result to Go value
	if jsResult == nil || jsResult.IsUndefined() || jsResult.IsNull() {
		return nil, nil
	}

	goResult, err := bridge.GoValue(jsResult, v8ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result: %w", err)
	}

	// Extract data field if present
	if resultMap, ok := goResult.(map[string]interface{}); ok {
		if data, exists := resultMap["data"]; exists {
			return data, nil
		}
	}

	return goResult, nil
}

// executeHookFunctionWithCases executes BeforeAll with test cases array
func (h *HookExecutor) executeHookFunctionWithCases(script *v8.Script, funcName string, testCases []*Case) (interface{}, error) {
	scriptCtx, err := script.NewContext("", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create script context: %w", err)
	}
	defer scriptCtx.Close()

	v8ctx := scriptCtx.Context

	if err := h.setShareData(v8ctx); err != nil {
		return nil, err
	}

	global := v8ctx.Global()
	fnValue, err := global.Get(funcName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function %s: %w", funcName, err)
	}

	if fnValue.IsUndefined() || fnValue.IsNull() {
		return nil, fmt.Errorf("function %s not defined", funcName)
	}

	if !fnValue.IsFunction() {
		return nil, fmt.Errorf("%s is not a function", funcName)
	}

	fn, err := fnValue.AsFunction()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to function: %w", err)
	}

	// Build ctx argument
	ctxJS, err := h.buildCtxArg(v8ctx)
	if err != nil {
		return nil, err
	}

	// Convert test cases to JS array
	casesJS, err := h.testCasesToJS(v8ctx, testCases)
	if err != nil {
		return nil, err
	}

	jsResult, err := fn.Call(global, ctxJS, casesJS)
	if err != nil {
		return nil, fmt.Errorf("hook function %s failed: %w", funcName, err)
	}

	if jsResult == nil || jsResult.IsUndefined() || jsResult.IsNull() {
		return nil, nil
	}

	goResult, err := bridge.GoValue(jsResult, v8ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result: %w", err)
	}

	if resultMap, ok := goResult.(map[string]interface{}); ok {
		if data, exists := resultMap["data"]; exists {
			return data, nil
		}
	}

	return goResult, nil
}

// executeHookFunctionWithResults executes AfterAll with results array
func (h *HookExecutor) executeHookFunctionWithResults(script *v8.Script, funcName string, results []*Result, beforeData interface{}) (interface{}, error) {
	scriptCtx, err := script.NewContext("", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create script context: %w", err)
	}
	defer scriptCtx.Close()

	v8ctx := scriptCtx.Context

	if err := h.setShareData(v8ctx); err != nil {
		return nil, err
	}

	global := v8ctx.Global()
	fnValue, err := global.Get(funcName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function %s: %w", funcName, err)
	}

	if fnValue.IsUndefined() || fnValue.IsNull() {
		return nil, fmt.Errorf("function %s not defined", funcName)
	}

	if !fnValue.IsFunction() {
		return nil, fmt.Errorf("%s is not a function", funcName)
	}

	fn, err := fnValue.AsFunction()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to function: %w", err)
	}

	// Build ctx argument
	ctxJS, err := h.buildCtxArg(v8ctx)
	if err != nil {
		return nil, err
	}

	// Convert results to JS array
	resultsJS, err := h.resultsToJS(v8ctx, results)
	if err != nil {
		return nil, err
	}

	// Convert beforeData to JS
	beforeDataJS, err := bridge.JsValue(v8ctx, beforeData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert beforeData: %w", err)
	}

	jsResult, err := fn.Call(global, ctxJS, resultsJS, beforeDataJS)
	if err != nil {
		return nil, fmt.Errorf("hook function %s failed: %w", funcName, err)
	}

	if jsResult == nil || jsResult.IsUndefined() || jsResult.IsNull() {
		return nil, nil
	}

	goResult, err := bridge.GoValue(jsResult, v8ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result: %w", err)
	}

	return goResult, nil
}

// setShareData sets the share data for script execution
func (h *HookExecutor) setShareData(v8ctx *v8go.Context) error {
	var authorized map[string]interface{}
	if h.agentContext != nil && h.agentContext.Authorized != nil {
		authorized = h.agentContext.Authorized.AuthorizedToMap()
	}

	return bridge.SetShareData(v8ctx, v8ctx.Global(), &bridge.Share{
		Sid:        "",
		Root:       false,
		Global:     nil,
		Authorized: authorized,
	})
}

// buildCtxArg builds the context argument for hook functions
func (h *HookExecutor) buildCtxArg(v8ctx *v8go.Context) (*v8go.Value, error) {
	ctxMap := map[string]interface{}{
		"locale": "en",
	}

	// Use ContextData from --ctx flag if available
	if h.opts != nil && h.opts.ContextData != nil {
		cfg := h.opts.ContextData
		if cfg.Locale != "" {
			ctxMap["locale"] = cfg.Locale
		}
		if cfg.Authorized != nil {
			authorized := map[string]interface{}{}
			if cfg.Authorized.UserID != "" {
				authorized["user_id"] = cfg.Authorized.UserID
			}
			if cfg.Authorized.TeamID != "" {
				authorized["team_id"] = cfg.Authorized.TeamID
			}
			if cfg.Authorized.TenantID != "" {
				authorized["tenant_id"] = cfg.Authorized.TenantID
			}
			if cfg.Authorized.Sub != "" {
				authorized["sub"] = cfg.Authorized.Sub
			}
			ctxMap["authorized"] = authorized
		}
		if cfg.Metadata != nil {
			ctxMap["metadata"] = cfg.Metadata
		}
	}

	return bridge.JsValue(v8ctx, ctxMap)
}

// buildHookArgs builds the arguments for a hook function call
// Arguments order: ctx, testCase, result (for After), beforeData (for After)
func (h *HookExecutor) buildHookArgs(v8ctx *v8go.Context, testCase *Case, result *Result, beforeData interface{}) ([]*v8go.Value, error) {
	var args []*v8go.Value

	// Arg 1: ctx (context) - build from opts.ContextData if available
	ctxMap := map[string]interface{}{
		"locale": "en",
	}

	// Use ContextData from --ctx flag if available
	if h.opts != nil && h.opts.ContextData != nil {
		cfg := h.opts.ContextData
		if cfg.Locale != "" {
			ctxMap["locale"] = cfg.Locale
		}
		if cfg.Authorized != nil {
			authorized := map[string]interface{}{}
			if cfg.Authorized.UserID != "" {
				authorized["user_id"] = cfg.Authorized.UserID
			}
			if cfg.Authorized.TeamID != "" {
				authorized["team_id"] = cfg.Authorized.TeamID
			}
			if cfg.Authorized.TenantID != "" {
				authorized["tenant_id"] = cfg.Authorized.TenantID
			}
			if cfg.Authorized.Sub != "" {
				authorized["sub"] = cfg.Authorized.Sub
			}
			ctxMap["authorized"] = authorized
		}
		if cfg.Metadata != nil {
			ctxMap["metadata"] = cfg.Metadata
		}
	} else if testCase != nil {
		// Fallback to test case fields
		if testCase.UserID != "" {
			ctxMap["user_id"] = testCase.UserID
		}
		if testCase.TeamID != "" {
			ctxMap["team_id"] = testCase.TeamID
		}
		// Build authorized info
		authorized := map[string]interface{}{}
		if testCase.UserID != "" {
			authorized["user_id"] = testCase.UserID
		}
		if testCase.TeamID != "" {
			authorized["team_id"] = testCase.TeamID
		}
		if len(authorized) > 0 {
			ctxMap["authorized"] = authorized
		}
	}

	ctxJS, err := bridge.JsValue(v8ctx, ctxMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ctx: %w", err)
	}
	args = append(args, ctxJS)

	// Arg 2: testCase
	if testCase != nil {
		tcMap := map[string]interface{}{
			"id":    testCase.ID,
			"input": testCase.Input,
		}
		if testCase.Metadata != nil {
			tcMap["metadata"] = testCase.Metadata
		}
		if testCase.Assert != nil {
			tcMap["assert"] = testCase.Assert
		}
		// Include simulator options for dynamic tests
		if testCase.Simulator != nil {
			tcMap["simulator"] = testCase.Simulator
		}

		tcJS, err := bridge.JsValue(v8ctx, tcMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert testCase: %w", err)
		}
		args = append(args, tcJS)
	} else {
		// Pass empty object if no testCase
		emptyJS, _ := bridge.JsValue(v8ctx, map[string]interface{}{})
		args = append(args, emptyJS)
	}

	// Arg 2: result (for After)
	if result != nil {
		resultMap := map[string]interface{}{
			"id":          result.ID,
			"status":      string(result.Status),
			"duration_ms": result.DurationMs,
		}
		if result.Output != nil {
			resultMap["output"] = result.Output
		}
		if result.Error != "" {
			resultMap["error"] = result.Error
		}

		resultJS, err := bridge.JsValue(v8ctx, resultMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert result: %w", err)
		}
		args = append(args, resultJS)
	}

	// Arg 3: beforeData (for After)
	if beforeData != nil {
		beforeDataJS, err := bridge.JsValue(v8ctx, beforeData)
		if err != nil {
			return nil, fmt.Errorf("failed to convert beforeData: %w", err)
		}
		args = append(args, beforeDataJS)
	}

	return args, nil
}

// testCasesToJS converts test cases to a JS array
func (h *HookExecutor) testCasesToJS(v8ctx *v8go.Context, testCases []*Case) (*v8go.Value, error) {
	cases := make([]map[string]interface{}, len(testCases))
	for i, tc := range testCases {
		cases[i] = map[string]interface{}{
			"id":    tc.ID,
			"input": tc.Input,
		}
		if tc.Metadata != nil {
			cases[i]["metadata"] = tc.Metadata
		}
	}

	return bridge.JsValue(v8ctx, cases)
}

// resultsToJS converts results to a JS array
func (h *HookExecutor) resultsToJS(v8ctx *v8go.Context, results []*Result) (*v8go.Value, error) {
	resultMaps := make([]map[string]interface{}, len(results))
	for i, r := range results {
		resultMaps[i] = map[string]interface{}{
			"id":          r.ID,
			"status":      string(r.Status),
			"duration_ms": r.DurationMs,
		}
		if r.Output != nil {
			resultMaps[i]["output"] = r.Output
		}
		if r.Error != "" {
			resultMaps[i]["error"] = r.Error
		}
	}

	return bridge.JsValue(v8ctx, resultMaps)
}
