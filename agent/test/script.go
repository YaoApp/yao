package test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/context"
	"rogchap.com/v8go"
)

// ScriptRunner executes script tests
type ScriptRunner struct {
	opts   *Options
	output *OutputWriter
}

// NewScriptRunner creates a new script test runner
func NewScriptRunner(opts *Options) *ScriptRunner {
	return &ScriptRunner{
		opts:   opts,
		output: NewOutputWriter(opts.Verbose),
	}
}

// ResolveScript resolves the script path from scripts.xxx.yyy or scripts.xxx.yyy.zzz format
func ResolveScript(input string) (*ScriptInfo, error) {
	// Remove "scripts." prefix
	path := strings.TrimPrefix(input, "scripts.")

	// Split into parts:
	// "expense.setup" -> ["expense", "setup"]
	// "expense.submission.validation" -> ["expense", "submission", "validation"]
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid script path: %s (expected format: scripts.assistant.module or scripts.assistant.sub_agent.module)", input)
	}

	// Build paths based on number of parts
	var basePaths []string
	var assistantDir, moduleName string

	if len(parts) == 2 {
		// Format: scripts.expense.setup
		// assistantDir: expense
		// moduleName: setup
		assistantDir = parts[0]
		moduleName = parts[1]
		basePaths = []string{
			filepath.Join("assistants", assistantDir, "src"),
			filepath.Join(assistantDir, "src"),
		}
	} else {
		// Format: scripts.expense.submission.validation (sub-agent)
		// assistantDir: expense/submission (or expense.submission)
		// moduleName: validation
		assistantDir = strings.Join(parts[:len(parts)-1], "/")
		moduleName = parts[len(parts)-1]
		basePaths = []string{
			filepath.Join("assistants", assistantDir, "src"),
			filepath.Join(assistantDir, "src"),
		}
	}

	var scriptPath, testPath string
	for _, basePath := range basePaths {
		// Check for TypeScript files first, then JavaScript
		for _, ext := range []string{".ts", ".js"} {
			candidateScript := filepath.Join(basePath, moduleName+ext)
			candidateTest := filepath.Join(basePath, moduleName+"_test"+ext)

			// Check if test file exists
			exists, err := application.App.Exists(candidateTest)
			if err == nil && exists {
				scriptPath = candidateScript
				testPath = candidateTest
				break
			}
		}
		if testPath != "" {
			break
		}
	}

	if testPath == "" {
		return nil, fmt.Errorf("test file not found for %s (tried: %s)", input, strings.Join(basePaths, ", "))
	}

	return &ScriptInfo{
		ID:         input,
		Assistant:  assistantDir,
		Module:     moduleName,
		ScriptPath: scriptPath,
		TestPath:   testPath,
	}, nil
}

// DiscoverTests finds all Test* functions in the script
func DiscoverTests(scriptPath string) ([]*ScriptTestCase, error) {
	// Read the script file
	content, err := application.App.Read(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script: %w", err)
	}

	// Parse the script to find Test* functions
	// We use a simple regex-like approach to find function declarations
	tests := make([]*ScriptTestCase, 0)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Match function declarations: function TestXxx( or export function TestXxx(
		if strings.Contains(line, "function Test") {
			// Extract function name
			name := extractFunctionName(line)
			if name != "" && strings.HasPrefix(name, "Test") {
				tests = append(tests, &ScriptTestCase{
					Name:     name,
					Function: name,
				})
			}
		}
	}

	return tests, nil
}

// extractFunctionName extracts the function name from a line
func extractFunctionName(line string) string {
	// Remove "export" prefix if present
	line = strings.TrimPrefix(line, "export ")
	line = strings.TrimSpace(line)

	// Match "function Name("
	if !strings.HasPrefix(line, "function ") {
		return ""
	}

	line = strings.TrimPrefix(line, "function ")

	// Find the opening parenthesis
	idx := strings.Index(line, "(")
	if idx == -1 {
		return ""
	}

	return strings.TrimSpace(line[:idx])
}

// filterTests filters test cases by a regex pattern (similar to go test -run)
func (r *ScriptRunner) filterTests(tests []*ScriptTestCase, pattern string) ([]*ScriptTestCase, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	filtered := make([]*ScriptTestCase, 0)
	for _, tc := range tests {
		if re.MatchString(tc.Name) {
			filtered = append(filtered, tc)
		}
	}

	return filtered, nil
}

// Run executes all script tests and returns a report
func (r *ScriptRunner) Run() (*ScriptTestReport, error) {
	startTime := time.Now()

	// Resolve script
	scriptInfo, err := ResolveScript(r.opts.Input)
	if err != nil {
		return nil, err
	}

	// Print header
	r.output.Header("Script Test")
	r.output.Info("Script: %s", scriptInfo.TestPath)

	// Discover tests
	tests, err := DiscoverTests(scriptInfo.TestPath)
	if err != nil {
		return nil, err
	}

	// Filter tests by -run pattern if specified
	if r.opts.Run != "" {
		tests, err = r.filterTests(tests, r.opts.Run)
		if err != nil {
			return nil, fmt.Errorf("invalid -run pattern: %w", err)
		}
		r.output.Info("Tests: %d functions (filtered by: %s)", len(tests), r.opts.Run)
	} else {
		r.output.Info("Tests: %d functions", len(tests))
	}

	if len(tests) == 0 {
		r.output.Warning("No tests to run")
	}

	// Load context config if specified
	var ctxConfig *ContextConfig
	if r.opts.ContextFile != "" {
		var err error
		ctxConfig, err = LoadContextConfig(r.opts.ContextFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load context file: %w", err)
		}
		r.output.Info("Context: %s", r.opts.ContextFile)
	}

	// Create environment with optional context config
	var env *Environment
	if ctxConfig != nil {
		env = NewEnvironmentWithContext(r.opts.UserID, r.opts.TeamID, ctxConfig)
	} else {
		env = NewEnvironment(r.opts.UserID, r.opts.TeamID)
	}
	r.output.Info("User: %s", env.UserID)
	r.output.Info("Team: %s", env.TeamID)

	// Load all scripts from src directory (including the test file)
	// This ensures imports can be resolved properly
	srcDir := filepath.Dir(scriptInfo.TestPath)
	loadedCount, err := r.loadAllScripts(srcDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load scripts: %w", err)
	}
	r.output.Info("Loaded: %d scripts", loadedCount)

	// Create report
	report := &ScriptTestReport{
		Type:        "script_test",
		Script:      scriptInfo.ID,
		ScriptPath:  scriptInfo.TestPath,
		Summary:     &ScriptTestSummary{Total: len(tests)},
		Environment: env,
		Results:     make([]*ScriptTestResult, 0, len(tests)),
		Metadata: &ScriptTestMetadata{
			StartedAt: startTime,
		},
	}

	// Run tests
	r.output.SubHeader("Running Tests")

	for _, tc := range tests {
		result := r.runScriptTest(tc, scriptInfo, env)
		report.Results = append(report.Results, result)

		// Update summary
		switch result.Status {
		case StatusPassed:
			report.Summary.Passed++
		case StatusFailed, StatusError:
			// Both Failed and Error count as failures
			report.Summary.Failed++
		case StatusSkipped:
			report.Summary.Skipped++
		}

		// Check fail-fast (stop on both Failed and Error)
		if r.opts.FailFast && (result.Status == StatusFailed || result.Status == StatusError) {
			r.output.Warning("Stopping due to --fail-fast")
			break
		}
	}

	// Complete report
	report.Summary.DurationMs = time.Since(startTime).Milliseconds()
	report.Metadata.CompletedAt = time.Now()

	// Print summary
	r.output.ScriptTestSummary(report.Summary, time.Since(startTime))

	return report, nil
}

// runScriptTest runs a single script test function
func (r *ScriptRunner) runScriptTest(tc *ScriptTestCase, scriptInfo *ScriptInfo, env *Environment) *ScriptTestResult {
	r.output.TestStart(tc.Name, "", 1)
	startTime := time.Now()

	result := &ScriptTestResult{
		Name:   tc.Name,
		Status: StatusPassed,
	}

	// Create testing.T object
	testingT := NewTestingT(tc.Name)

	// Create agent context
	chatID := fmt.Sprintf("script-test-%s", tc.Name)
	agentCtx := NewTestContext(chatID, scriptInfo.Assistant, env)
	defer agentCtx.Release()

	// Execute the test function
	err := r.executeTestFunction(tc, scriptInfo, testingT, agentCtx)

	duration := time.Since(startTime)
	result.DurationMs = duration.Milliseconds()
	result.Logs = testingT.Logs()

	if err != nil {
		result.Status = StatusError
		result.Error = err.Error()
		r.output.TestResult(result.Status, duration)
		r.output.TestError(result.Error)
		return result
	}

	if testingT.Skipped() {
		result.Status = StatusSkipped
		r.output.TestResult(result.Status, duration)
		return result
	}

	if testingT.Failed() {
		result.Status = StatusFailed
		errors := testingT.Errors()
		if len(errors) > 0 {
			result.Error = errors[0]
		}
		result.Assertion = testingT.AssertionInfo()
		r.output.TestResult(result.Status, duration)
		r.output.TestError(result.Error)
		return result
	}

	r.output.TestResult(result.Status, duration)
	return result
}

// loadAllScripts loads all scripts from the src directory
// This ensures that imports can be resolved properly
func (r *ScriptRunner) loadAllScripts(srcDir string) (int, error) {
	count := 0

	// Check if src directory exists
	exists, err := application.App.Exists(srcDir)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, fmt.Errorf("src directory not found: %s", srcDir)
	}

	// Walk through src directory to find all script files
	exts := []string{"*.ts", "*.js"}

	err = application.App.Walk(srcDir, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		// Get relative path
		relPath := strings.TrimPrefix(file, root+"/")

		// Generate script ID from file path
		scriptID := generateTestScriptID(file, root)

		// Load the script
		_, err := v8.Load(file, scriptID)
		if err != nil {
			// Log warning but continue loading other scripts
			if r.opts.Verbose {
				r.output.Warning("Failed to load %s: %v", relPath, err)
			}
			return nil
		}

		count++
		if r.opts.Verbose {
			r.output.Verbose("Loaded: %s", relPath)
		}

		return nil
	}, exts...)

	if err != nil {
		return count, fmt.Errorf("failed to walk src directory: %w", err)
	}

	return count, nil
}

// generateTestScriptID generates a script ID from file path for testing
func generateTestScriptID(filePath string, srcDir string) string {
	// Normalize path separators
	filePath = filepath.ToSlash(filePath)
	srcDir = filepath.ToSlash(srcDir)

	// Remove src directory prefix
	relPath := strings.TrimPrefix(filePath, srcDir+"/")
	relPath = strings.TrimPrefix(relPath, "/")

	// Remove file extension
	relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))

	// Replace path separators with dots and add test prefix
	scriptID := "test." + strings.ReplaceAll(relPath, "/", ".")

	return scriptID
}

// executeTestFunction executes a single test function using V8
func (r *ScriptRunner) executeTestFunction(tc *ScriptTestCase, scriptInfo *ScriptInfo, testingT *TestingT, agentCtx *context.Context) (execErr error) {
	// Recover from panics thrown by Process calls
	// Even if JavaScript try-catch catches the error, we want to fail the test
	defer func() {
		if r := recover(); r != nil {
			execErr = fmt.Errorf("panic in test function: %v", r)
		}
	}()

	// Get the test script (already loaded by loadAllScripts)
	testScriptID := generateTestScriptID(scriptInfo.TestPath, filepath.Dir(scriptInfo.TestPath))
	script, ok := v8.Scripts[testScriptID]
	if !ok {
		return fmt.Errorf("test script not found: %s (id: %s)", scriptInfo.TestPath, testScriptID)
	}

	// Create a new script context
	scriptCtx, err := script.NewContext("", nil)
	if err != nil {
		return fmt.Errorf("failed to create script context: %w", err)
	}
	defer scriptCtx.Close()

	// Get the V8 context
	v8ctx := scriptCtx.Context

	// Set share data with authorized info for Process calls
	// This is needed because we call fn.Call directly instead of scriptCtx.Call
	var authorized map[string]interface{}
	if agentCtx.Authorized != nil {
		authorized = agentCtx.Authorized.AuthorizedToMap()
	}
	err = bridge.SetShareData(v8ctx, v8ctx.Global(), &bridge.Share{
		Sid:        "",
		Root:       false,
		Global:     nil,
		Authorized: authorized,
	})
	if err != nil {
		return fmt.Errorf("failed to set share data: %w", err)
	}

	// Create testing.T JavaScript object
	testingTObj, err := NewTestingTObject(v8ctx, testingT)
	if err != nil {
		return fmt.Errorf("failed to create testing.T object: %w", err)
	}

	// Create agent context JavaScript object
	agentCtxObj, err := agentCtx.JsValue(v8ctx)
	if err != nil {
		return fmt.Errorf("failed to create agent context object: %w", err)
	}

	// Get the test function
	global := v8ctx.Global()
	fnValue, err := global.Get(tc.Function)
	if err != nil {
		return fmt.Errorf("failed to get test function %s: %w", tc.Function, err)
	}

	if !fnValue.IsFunction() {
		return fmt.Errorf("test function %s is not a function", tc.Function)
	}

	fn, err := fnValue.AsFunction()
	if err != nil {
		return fmt.Errorf("failed to convert to function: %w", err)
	}

	// Call the test function with (t, ctx)
	result, err := fn.Call(global, testingTObj, agentCtxObj)
	if err != nil {
		// Check if this is an assertion failure or a real error
		if testingT.Failed() {
			// Assertion failure - already recorded
			return nil
		}
		return fmt.Errorf("test function error: %w", err)
	}

	// Check if the result is a JavaScript Error (thrown by bridge.JsException)
	if result != nil && result.IsNativeError() {
		// Get error message from Error object
		if result.IsObject() {
			obj, err := result.AsObject()
			if err == nil {
				if msgVal, err := obj.Get("message"); err == nil && !msgVal.IsUndefined() {
					return fmt.Errorf("test threw exception: %s", msgVal.String())
				}
			}
		}
		return fmt.Errorf("test threw exception: %s", result.String())
	}

	return nil
}

// RegisterTestingGlobals registers testing-related global functions for V8
// This is called once during initialization
func RegisterTestingGlobals() {
	v8.RegisterFunction("__testing_log", testingLogEmbed)
}

// testingLogEmbed provides a console.log-like function for tests
func testingLogEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		parts := make([]string, len(args))
		for i, arg := range args {
			goVal, err := bridge.GoValue(arg, info.Context())
			if err != nil {
				parts[i] = arg.String()
			} else {
				parts[i] = fmt.Sprintf("%v", goVal)
			}
		}
		fmt.Println(strings.Join(parts, " "))
		return v8go.Undefined(iso)
	})
}
