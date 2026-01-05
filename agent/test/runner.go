package test

import (
	stdContext "context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
)

// Executor executes test cases against an agent
type Executor struct {
	opts         *Options
	output       *OutputWriter
	resolver     Resolver
	loader       Loader
	hookExecutor *HookExecutor
	agentPath    string // Path to the agent being tested
}

// NewRunner creates a new test runner
func NewRunner(opts *Options) *Executor {
	return &Executor{
		opts:         opts,
		output:       NewOutputWriter(opts.Verbose),
		resolver:     NewResolver(),
		loader:       NewLoader(),
		hookExecutor: NewHookExecutor(opts.Verbose),
	}
}

// Run executes all test cases and returns a report
func (r *Executor) Run() (*Report, error) {
	// For script test mode, use script runner
	if r.opts.InputMode == InputModeScript {
		return r.RunScriptTests()
	}

	// For direct message mode, use simplified output (development mode)
	if r.opts.InputMode == InputModeMessage {
		return r.RunDirect()
	}

	return r.RunTests()
}

// RunScriptTests executes script tests and returns a report
func (r *Executor) RunScriptTests() (*Report, error) {
	scriptRunner := NewScriptRunner(r.opts)
	scriptReport, err := scriptRunner.Run()
	if err != nil {
		return nil, err
	}

	// Convert to standard report for unified output handling
	report := scriptReport.ToReport()

	// Write output if specified
	if r.opts.OutputFile != "" {
		err = r.writeOutput(report)
		if err != nil {
			r.output.Error("Failed to write output: %s", err.Error())
		} else {
			r.output.OutputFile(r.opts.OutputFile)
		}
	}

	// Print final result
	r.output.FinalResult(!report.HasFailures())

	return report, nil
}

// RunDirect executes a single direct message and outputs the result directly
// This is optimized for development/debugging scenarios
func (r *Executor) RunDirect() (*Report, error) {
	// Resolve agent
	agentInfo, err := r.resolver.Resolve(r.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve agent: %w", err)
	}

	// Get assistant
	ast, err := assistant.Get(agentInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant: %w", err)
	}

	// Create test case from message
	tc := CreateTestCaseFromMessage(r.opts.Input)

	// Create context
	chatID := GenerateChatID(tc.ID, 1)
	ctx := NewTestContextFromOptions(chatID, agentInfo.ID, r.opts, tc)
	defer ctx.Release()

	// Build context options
	opts := buildContextOptions(tc, r.opts)

	// Create timeout context
	timeout := tc.GetTimeout(r.opts.Timeout)
	timeoutCtx, cancel := stdContext.WithTimeout(ctx.Context, timeout)
	defer cancel()
	ctx.Context = timeoutCtx

	// Parse input to messages with file loading support
	inputOpts := r.getInputOptions()
	messages, err := tc.GetMessagesWithOptions(inputOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Run the agent
	response, err := ast.Stream(ctx, messages, opts)

	// Check for timeout
	if timeoutCtx.Err() != nil {
		return nil, fmt.Errorf("timeout after %s", timeout)
	}

	// Check for error
	if err != nil {
		return nil, err
	}

	// Extract and print output directly
	output := extractOutput(response)
	r.output.DirectOutput(output)

	// Determine connector: user-specified > agent default
	connector := r.opts.Connector
	if connector == "" {
		connector = agentInfo.Connector
	}

	// Return minimal report (for exit code handling)
	return &Report{
		Summary: &Summary{
			Total:     1,
			Passed:    1,
			AgentID:   agentInfo.ID,
			Connector: connector,
		},
	}, nil
}

// RunTests executes test cases from file and generates a report
func (r *Executor) RunTests() (*Report, error) {
	startTime := time.Now()

	// Print header
	r.output.Header("Agent Test")

	// Resolve agent
	agentInfo, err := r.resolver.Resolve(r.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve agent: %w", err)
	}

	r.output.Info("Agent: %s", agentInfo.ID)
	r.agentPath = agentInfo.Path // Store agent path for hook execution
	if r.opts.Connector != "" {
		r.output.Info("Connector: %s (override)", r.opts.Connector)
	} else if agentInfo.Connector != "" {
		r.output.Info("Connector: %s", agentInfo.Connector)
	}

	// Load test cases based on input source
	var testCases []*Case
	inputSource := ParseInputSource(r.opts.Input)

	switch inputSource.Type {
	case InputSourceAgent:
		// Generate test cases using agent
		r.output.Info("Generating test cases from agent: %s", inputSource.Value)
		targetInfo := &TargetAgentInfo{
			ID:          agentInfo.ID,
			Description: agentInfo.Description,
		}
		testCases, err = r.loader.LoadFromAgent(inputSource.Value, targetInfo, inputSource.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to generate test cases: %w", err)
		}
		r.output.Info("Generated: %d test cases", len(testCases))

	case InputSourceScript:
		// Generate test cases using script (if it's a generator script, not test script)
		// Note: scripts. prefix without "scripts:" is handled by RunScriptTests
		if strings.HasPrefix(r.opts.Input, "scripts:") {
			scriptRef := strings.TrimPrefix(r.opts.Input, "scripts:")
			r.output.Info("Generating test cases from script: %s", scriptRef)
			targetInfo := &TargetAgentInfo{
				ID:          agentInfo.ID,
				Description: agentInfo.Description,
			}
			testCases, err = r.loader.LoadFromScript(scriptRef, targetInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to generate test cases from script: %w", err)
			}
			r.output.Info("Generated: %d test cases", len(testCases))
		} else {
			// This is a test script (scripts.xxx format), handled by RunScriptTests
			return nil, fmt.Errorf("script test mode should be handled by RunScriptTests")
		}

	default:
		// File mode - load from JSONL
		testCases, err = r.loader.LoadFile(r.opts.Input)
		if err != nil {
			return nil, fmt.Errorf("failed to load test cases: %w", err)
		}
		r.output.Info("Input: %s (%d test cases)", r.opts.Input, len(testCases))
	}

	// Handle dry-run mode - just output the generated test cases
	if r.opts.DryRun {
		r.output.Info("Dry-run mode: outputting generated test cases")
		return r.outputDryRun(testCases, agentInfo)
	}

	// Filter skipped tests
	activeTests := FilterSkipped(testCases)
	skippedCount := len(testCases) - len(activeTests)
	if skippedCount > 0 {
		r.output.Warning("Skipped: %d test cases", skippedCount)
	}

	// Filter by --run pattern if specified
	if r.opts.Run != "" {
		runPattern, err := regexp.Compile(r.opts.Run)
		if err != nil {
			return nil, fmt.Errorf("invalid --run pattern %q: %w", r.opts.Run, err)
		}
		activeTests = FilterByPattern(activeTests, runPattern)
		if len(activeTests) == 0 {
			return nil, fmt.Errorf("no test cases match pattern %q", r.opts.Run)
		}
		r.output.Info("Filter: %q (%d test cases match)", r.opts.Run, len(activeTests))
	}

	// Load context config if specified
	if r.opts.ContextFile != "" {
		ctxConfig, err := LoadContextConfig(r.opts.ContextFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load context file: %w", err)
		}
		r.opts.ContextData = ctxConfig
		r.output.Info("Context: %s", r.opts.ContextFile)
	}

	// Set options on hook executor (for context data access in hooks)
	r.hookExecutor.SetOptions(r.opts)

	// Print test info
	if r.opts.Runs > 1 {
		r.output.Info("Runs: %d per test case (stability analysis)", r.opts.Runs)
	}
	r.output.Info("Timeout: %s", r.opts.Timeout)
	if r.opts.Parallel > 1 {
		r.output.Info("Parallel: %d", r.opts.Parallel)
	}

	// Get assistant
	ast, err := assistant.Get(agentInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant: %w", err)
	}

	// Determine connector: user-specified > agent default
	connector := r.opts.Connector
	if connector == "" {
		connector = agentInfo.Connector
	}

	// Create report
	report := &Report{
		Summary: &Summary{
			Total:       len(testCases),
			AgentID:     agentInfo.ID,
			AgentPath:   agentInfo.Path,
			Connector:   connector,
			RunsPerCase: r.opts.Runs,
		},
		Environment: NewEnvironment(r.opts.UserID, r.opts.TeamID),
		Metadata: &ReportMetadata{
			StartedAt: startTime,
			InputFile: r.opts.Input,
			Options:   r.opts,
		},
	}

	// Execute global BeforeAll if specified
	var globalBeforeData interface{}
	if r.opts.BeforeAll != "" {
		r.output.Info("BeforeAll: %s", r.opts.BeforeAll)
		var err error
		globalBeforeData, err = r.hookExecutor.ExecuteBeforeAll(r.opts.BeforeAll, activeTests, agentInfo.Path)
		if err != nil {
			return nil, fmt.Errorf("beforeAll script failed: %w", err)
		}
	}

	// Ensure AfterAll runs even if tests fail
	defer func() {
		if r.opts.AfterAll != "" {
			r.output.Info("AfterAll: %s", r.opts.AfterAll)
			if err := r.hookExecutor.ExecuteAfterAll(r.opts.AfterAll, report.Results, globalBeforeData, agentInfo.Path); err != nil {
				r.output.Warning("afterAll script failed: %s", err.Error())
			}
		}
	}()

	// Run tests
	r.output.SubHeader("Running Tests")

	if r.opts.Runs > 1 {
		// Stability testing mode
		report.StabilityResults = r.runStabilityTests(ast, activeTests, agentInfo.ID)
		r.calculateStabilitySummary(report)
	} else {
		// Single run mode
		report.Results = r.runSingleTests(ast, activeTests, agentInfo.ID)
		r.calculateSingleSummary(report)
	}

	// Add skipped count
	report.Summary.Skipped = skippedCount

	// Complete report
	report.Summary.DurationMs = time.Since(startTime).Milliseconds()
	report.Metadata.CompletedAt = time.Now()

	// Print summary
	r.output.Summary(report.Summary, time.Since(startTime))

	// Write output
	if r.opts.OutputFile != "" {
		err = r.writeOutput(report)
		if err != nil {
			r.output.Error("Failed to write output: %s", err.Error())
		} else {
			r.output.OutputFile(r.opts.OutputFile)
		}
	}

	// Print final result
	r.output.FinalResult(!report.HasFailures())

	return report, nil
}

// runSingleTests runs each test case once
func (r *Executor) runSingleTests(ast *assistant.Assistant, testCases []*Case, agentID string) []*Result {
	results := make([]*Result, 0, len(testCases))

	if r.opts.Parallel > 1 {
		// Parallel execution
		results = r.runParallel(ast, testCases, agentID)
	} else {
		// Sequential execution
		for i, tc := range testCases {
			result := r.runSingleTest(ast, tc, agentID, 1)
			results = append(results, result)

			// Check fail-fast
			if r.opts.FailFast && result.Status != StatusPassed && result.Status != StatusSkipped {
				r.output.Warning("Stopping due to --fail-fast (failed at test %d/%d)", i+1, len(testCases))
				break
			}
		}
	}

	return results
}

// runParallel runs tests in parallel
func (r *Executor) runParallel(ast *assistant.Assistant, testCases []*Case, agentID string) []*Result {
	results := make([]*Result, len(testCases))
	var wg sync.WaitGroup
	sem := make(chan struct{}, r.opts.Parallel)

	for i, tc := range testCases {
		wg.Add(1)
		go func(idx int, testCase *Case) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			results[idx] = r.runSingleTest(ast, testCase, agentID, 1)
		}(i, tc)
	}

	wg.Wait()
	return results
}

// runSingleTest runs a single test case
func (r *Executor) runSingleTest(ast *assistant.Assistant, tc *Case, agentID string, runNum int) *Result {
	// Check if this is a dynamic mode test
	if tc.IsDynamicMode() {
		return r.runDynamicTest(ast, tc, agentID)
	}

	// Get input summary for display
	inputSummary := SummarizeInput(tc.Input, 50)
	r.output.TestStart(tc.ID, inputSummary, runNum)

	startTime := time.Now()

	// Create result
	result := &Result{
		ID:       tc.ID,
		Input:    tc.Input,
		Expected: tc.Expected,
		Options:  tc.Options,
	}

	// Execute before script if specified
	var beforeData interface{}
	if tc.Before != "" {
		var err error
		beforeData, err = r.hookExecutor.ExecuteBefore(tc.Before, tc, r.agentPath)
		if err != nil {
			result.Status = StatusError
			result.Error = fmt.Sprintf("before script failed: %s", err.Error())
			result.DurationMs = time.Since(startTime).Milliseconds()
			r.output.TestResult(result.Status, time.Since(startTime))
			r.output.TestError(result.Error)
			// Note: after script is NOT called when before fails
			return result
		}
	}

	// Ensure after script runs even if test fails (but only if before succeeded)
	defer func() {
		if tc.After != "" && (tc.Before == "" || beforeData != nil || result.Status != StatusError || !isBeforeError(result.Error)) {
			if err := r.hookExecutor.ExecuteAfter(tc.After, tc, result, beforeData, r.agentPath); err != nil {
				r.output.Warning("after script failed: %s", err.Error())
			}
		}
	}()

	// Parse input to messages with file loading support
	// BaseDir is derived from the input file directory
	inputOpts := r.getInputOptions()
	messages, err := tc.GetMessagesWithOptions(inputOpts)
	if err != nil {
		result.Status = StatusError
		result.Error = fmt.Sprintf("failed to parse input: %s", err.Error())
		result.DurationMs = time.Since(startTime).Milliseconds()
		r.output.TestResult(result.Status, time.Since(startTime))
		r.output.TestError(result.Error)
		return result
	}

	// Create context
	chatID := GenerateChatID(tc.ID, runNum)
	ctx := NewTestContextFromOptions(chatID, agentID, r.opts, tc)
	defer ctx.Release()

	// Build context options from test case and runner options
	opts := buildContextOptions(tc, r.opts)

	// Create timeout context
	timeout := tc.GetTimeout(r.opts.Timeout)
	timeoutCtx, cancel := stdContext.WithTimeout(ctx.Context, timeout)
	defer cancel()
	ctx.Context = timeoutCtx

	// Run the test
	response, err := ast.Stream(ctx, messages, opts)

	duration := time.Since(startTime)
	result.DurationMs = duration.Milliseconds()

	// Check for timeout
	if timeoutCtx.Err() != nil {
		result.Status = StatusTimeout
		result.Error = fmt.Sprintf("timeout after %s", timeout)
		r.output.TestResult(result.Status, duration)
		r.output.TestError(result.Error)
		return result
	}

	// Check for error
	if err != nil {
		result.Status = StatusError
		result.Error = err.Error()
		r.output.TestResult(result.Status, duration)
		r.output.TestError(result.Error)
		return result
	}

	// Extract output
	result.Output = extractOutput(response)

	// Validate result using asserter (with response for tool_called assertions)
	asserter := NewAsserter().WithResponse(response)
	passed, errMsg := asserter.Validate(tc, result.Output)
	if passed {
		result.Status = StatusPassed
	} else {
		result.Status = StatusFailed
		result.Error = errMsg
	}

	r.output.TestResult(result.Status, duration)
	if result.Status == StatusFailed {
		r.output.TestError(result.Error)
	}
	r.output.TestOutput(fmt.Sprintf("%v", result.Output))

	return result
}

// runDynamicTest runs a dynamic (simulator-driven) test case
func (r *Executor) runDynamicTest(ast *assistant.Assistant, tc *Case, agentID string) *Result {
	// Output test start for dynamic mode
	r.output.DynamicTestStart(tc.ID, len(tc.Checkpoints))

	startTime := time.Now()

	// Execute before script if specified
	var beforeData interface{}
	if tc.Before != "" {
		var err error
		beforeData, err = r.hookExecutor.ExecuteBefore(tc.Before, tc, r.agentPath)
		if err != nil {
			result := &Result{
				ID:         tc.ID,
				Status:     StatusError,
				Error:      fmt.Sprintf("before script failed: %s", err.Error()),
				DurationMs: time.Since(startTime).Milliseconds(),
			}
			r.output.TestResult(result.Status, time.Since(startTime))
			r.output.TestError(result.Error)
			return result
		}
	}

	// Create dynamic runner and execute
	dynamicRunner := NewDynamicRunner(r.opts)
	dynamicResult := dynamicRunner.RunDynamic(ast, tc, agentID)

	// Convert to standard result
	result := dynamicResult.ToResult()

	// Execute after script if specified (before outputting result)
	if tc.After != "" && (tc.Before == "" || beforeData != nil || result.Status != StatusError || !isBeforeError(result.Error)) {
		if err := r.hookExecutor.ExecuteAfter(tc.After, tc, result, beforeData, r.agentPath); err != nil {
			r.output.Warning("after script failed: %s", err.Error())
		}
	}

	// Output result
	duration := time.Duration(result.DurationMs) * time.Millisecond
	r.output.DynamicTestResult(result.Status, dynamicResult.TotalTurns, len(tc.Checkpoints), duration)

	if result.Error != "" {
		r.output.TestError(result.Error)
	}

	return result
}

// isBeforeError checks if the error message indicates a before script failure
func isBeforeError(errMsg string) bool {
	return len(errMsg) > 0 && errMsg[:min(len(errMsg), 20)] == "before script failed"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// runStabilityTests runs each test case multiple times for stability analysis
func (r *Executor) runStabilityTests(ast *assistant.Assistant, testCases []*Case, agentID string) []*StabilityResult {
	results := make([]*StabilityResult, 0, len(testCases))

	for _, tc := range testCases {
		sr := &StabilityResult{
			ID:         tc.ID,
			Input:      tc.Input,
			Expected:   tc.Expected,
			RunDetails: make([]*RunDetail, 0, r.opts.Runs),
		}

		// Run multiple times
		for run := 1; run <= r.opts.Runs; run++ {
			result := r.runSingleTest(ast, tc, agentID, run)

			rd := &RunDetail{
				Run:        run,
				Status:     result.Status,
				DurationMs: result.DurationMs,
				Output:     result.Output,
				Error:      result.Error,
			}
			sr.RunDetails = append(sr.RunDetails, rd)
		}

		// Calculate stability metrics
		sr.CalculateStability()

		// Print stability result
		r.output.StabilityResult(sr)

		results = append(results, sr)

		// Check fail-fast
		if r.opts.FailFast && !sr.Stable {
			r.output.Warning("Stopping due to --fail-fast (unstable test: %s)", tc.ID)
			break
		}
	}

	return results
}

// calculateSingleSummary calculates summary for single run mode
func (r *Executor) calculateSingleSummary(report *Report) {
	for _, result := range report.Results {
		switch result.Status {
		case StatusPassed:
			report.Summary.Passed++
		case StatusFailed:
			report.Summary.Failed++
		case StatusError:
			report.Summary.Errors++
		case StatusTimeout:
			report.Summary.Timeouts++
		}
	}
}

// calculateStabilitySummary calculates summary for stability mode
func (r *Executor) calculateStabilitySummary(report *Report) {
	report.Summary.TotalRuns = len(report.StabilityResults) * r.opts.Runs

	var totalPassRate float64
	for _, sr := range report.StabilityResults {
		if sr.Stable {
			report.Summary.StableCases++
			report.Summary.Passed++
		} else {
			report.Summary.UnstableCases++
			report.Summary.Failed++
		}
		totalPassRate += sr.PassRate
	}

	if len(report.StabilityResults) > 0 {
		report.Summary.OverallPassRate = totalPassRate / float64(len(report.StabilityResults))
	}
}

// writeOutput writes the test report to the output file
func (r *Executor) writeOutput(report *Report) error {
	file, err := os.Create(r.opts.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Get reporter based on -r flag or file extension
	reporter := GetReporterWithAgent(r.opts.ReporterID, r.opts.OutputFile, r.opts.Verbose)

	// If using agent reporter, set context
	if agentReporter, ok := reporter.(*AgentReporter); ok {
		// Create a context for the reporter agent call
		ctx := NewTestContext("reporter", r.opts.ReporterID, report.Environment)
		defer ctx.Release()
		agentReporter.SetContext(ctx)
	}

	// Write report using the reporter
	return reporter.Write(report, file)
}

// buildContextOptions builds context.Options from test case and runner options
// Priority: test case options > runner options > defaults
func buildContextOptions(tc *Case, runnerOpts *Options) *context.Options {
	opts := &context.Options{
		Skip: &context.Skip{
			History: true, // Default: skip history loading - input already contains full conversation
		},
	}

	// Apply test case options if specified
	if tc.Options != nil {
		// Connector: test case > runner
		if tc.Options.Connector != "" {
			opts.Connector = tc.Options.Connector
		}

		// Mode
		if tc.Options.Mode != "" {
			opts.Mode = tc.Options.Mode
		}

		// DisableGlobalPrompts
		if tc.Options.DisableGlobalPrompts {
			opts.DisableGlobalPrompts = true
		}

		// Search (pointer to distinguish unset from false)
		if tc.Options.Search != nil {
			opts.Search = tc.Options.Search
		}

		// Metadata for hooks
		if tc.Options.Metadata != nil {
			opts.Metadata = tc.Options.Metadata
		}

		// Skip options from test case
		if tc.Options.Skip != nil {
			opts.Skip.Trace = tc.Options.Skip.Trace
			opts.Skip.Output = tc.Options.Skip.Output
			opts.Skip.Keyword = tc.Options.Skip.Keyword
			opts.Skip.Search = tc.Options.Skip.Search
			// Note: History defaults to true for tests
		}
	}

	// Runner connector override (highest priority)
	if runnerOpts != nil && runnerOpts.Connector != "" {
		opts.Connector = runnerOpts.Connector
	}

	return opts
}

// extractOutput extracts the output from the agent response
// Priority: Next hook data > Completion content > Tool results message > nil
func extractOutput(response *context.Response) interface{} {
	if response == nil {
		return nil
	}

	// Prefer Next hook data if available and non-empty
	// response.Next is already the Data value (not NextHookResponse struct)
	if response.Next != nil && !isEmptyValue(response.Next) {
		return response.Next
	}

	// Fall back to completion response
	if response.Completion != nil {
		// If content is non-empty, return it
		if response.Completion.Content != nil && !isEmptyValue(response.Completion.Content) {
			return response.Completion.Content
		}
	}

	// If no content but tools were executed, extract message from tool results
	// This handles the case where LLM calls tools but doesn't generate text
	if len(response.Tools) > 0 {
		return extractToolResultMessage(response.Tools)
	}

	return nil
}

// extractToolResultMessage extracts the message field from tool results
// Returns the first non-empty message found, or a summary of tool calls
func extractToolResultMessage(tools []context.ToolCallResponse) interface{} {
	if len(tools) == 0 {
		return nil
	}

	// Try to extract "message" field from tool results first
	for _, tool := range tools {
		if tool.Result != nil {
			// Try to get message from result map
			if resultMap, ok := tool.Result.(map[string]interface{}); ok {
				if msg, exists := resultMap["message"]; exists && msg != nil {
					if msgStr, ok := msg.(string); ok && msgStr != "" {
						return msgStr
					}
				}
			}
		}
	}

	// No message found, generate a summary of tool calls
	var summaries []string
	for _, tool := range tools {
		toolName := tool.Tool
		if toolName == "" {
			toolName = "unknown"
		}
		// Extract key info from result if possible
		if tool.Result != nil {
			if resultMap, ok := tool.Result.(map[string]interface{}); ok {
				// Try common result fields
				if action, ok := resultMap["action"].(string); ok {
					summaries = append(summaries, fmt.Sprintf("[%s: %s]", toolName, action))
					continue
				}
				if success, ok := resultMap["success"].(bool); ok {
					status := "failed"
					if success {
						status = "success"
					}
					summaries = append(summaries, fmt.Sprintf("[%s: %s]", toolName, status))
					continue
				}
			}
		}
		summaries = append(summaries, fmt.Sprintf("[%s]", toolName))
	}

	if len(summaries) > 0 {
		return strings.Join(summaries, " ")
	}
	return nil
}

// isEmptyValue checks if a value is considered "empty" for output purposes
func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}

	// Use reflection to check for typed nil (e.g., *NextHookResponse(nil))
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case map[string]interface{}:
		return len(val) == 0
	case []interface{}:
		return len(val) == 0
	case *context.NextHookResponse:
		// Check if NextHookResponse is effectively empty
		if val == nil {
			return true
		}
		return val.Data == nil && val.Delegate == nil
	}

	return false
}

// validateOutput validates the actual output against expected
func validateOutput(actual, expected interface{}) bool {
	// Simple JSON comparison
	actualJSON, err1 := jsoniter.Marshal(actual)
	expectedJSON, err2 := jsoniter.Marshal(expected)

	if err1 != nil || err2 != nil {
		return false
	}

	return string(actualJSON) == string(expectedJSON)
}

// getInputOptions returns InputOptions based on the runner configuration
// BaseDir is derived from the input file directory (for file mode) or current working directory
func (r *Executor) getInputOptions() *InputOptions {
	opts := &InputOptions{}

	// For file mode, use the input file's directory as base
	if r.opts.InputMode == InputModeFile && r.opts.Input != "" {
		// Resolve path considering YAO_ROOT
		resolvedPath := ResolvePathWithYaoRoot(r.opts.Input)
		opts.BaseDir = filepath.Dir(resolvedPath)
	}
	// For message mode, BaseDir remains empty (uses current working directory)

	return opts
}

// outputDryRun outputs generated test cases without running them
func (r *Executor) outputDryRun(testCases []*Case, agentInfo *AgentInfo) (*Report, error) {
	r.output.Info("Generated Test Cases:")

	// Output each test case as JSONL
	for _, tc := range testCases {
		data, err := jsoniter.Marshal(tc)
		if err != nil {
			r.output.Warning("Failed to marshal test case %s: %s", tc.ID, err.Error())
			continue
		}
		fmt.Println(string(data))
	}

	// Write to output file if specified
	if r.opts.OutputFile != "" {
		file, err := os.Create(r.opts.OutputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		for _, tc := range testCases {
			data, err := jsoniter.Marshal(tc)
			if err != nil {
				continue
			}
			file.WriteString(string(data) + "\n")
		}

		r.output.Info("Output written to: %s", r.opts.OutputFile)
	}

	// Return a minimal report
	connector := r.opts.Connector
	if connector == "" {
		connector = agentInfo.Connector
	}

	return &Report{
		Summary: &Summary{
			Total:     len(testCases),
			AgentID:   agentInfo.ID,
			Connector: connector,
		},
	}, nil
}
