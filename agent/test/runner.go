package test

import (
	"bufio"
	stdContext "context"
	"fmt"
	"os"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
)

// Executor executes test cases against an agent
type Executor struct {
	opts     *Options
	output   *OutputWriter
	resolver Resolver
	loader   Loader
}

// NewRunner creates a new test runner
func NewRunner(opts *Options) *Executor {
	return &Executor{
		opts:     opts,
		output:   NewOutputWriter(opts.Verbose),
		resolver: NewResolver(),
		loader:   NewLoader(),
	}
}

// Run executes all test cases and returns a report
func (r *Executor) Run() (*Report, error) {
	// For direct message mode, use simplified output (development mode)
	if r.opts.InputMode == InputModeMessage {
		return r.RunDirect()
	}

	return r.RunTests()
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

	// Set options: skip history (input already contains conversation), connector override
	opts := &context.Options{
		Skip: &context.Skip{
			History: true, // Skip history loading - input already contains full conversation
		},
	}
	if r.opts.Connector != "" {
		opts.Connector = r.opts.Connector
	}

	// Create timeout context
	timeout := tc.GetTimeout(r.opts.Timeout)
	timeoutCtx, cancel := stdContext.WithTimeout(ctx.Context, timeout)
	defer cancel()
	ctx.Context = timeoutCtx

	// Parse input to messages
	messages, err := tc.GetMessages()
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

	// Return minimal report (for exit code handling)
	return &Report{
		Summary: &Summary{
			Total:   1,
			Passed:  1,
			AgentID: agentInfo.ID,
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
	if r.opts.Connector != "" {
		r.output.Info("Connector: %s (override)", r.opts.Connector)
	} else if agentInfo.Connector != "" {
		r.output.Info("Connector: %s", agentInfo.Connector)
	}

	// Load test cases
	var testCases []*Case

	// File mode - load from JSONL
	testCases, err = r.loader.LoadFile(r.opts.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to load test cases: %w", err)
	}
	r.output.Info("Input: %s (%d test cases)", r.opts.Input, len(testCases))

	// Filter skipped tests
	activeTests := FilterSkipped(testCases)
	skippedCount := len(testCases) - len(activeTests)
	if skippedCount > 0 {
		r.output.Warning("Skipped: %d test cases", skippedCount)
	}

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

	// Create report
	report := &Report{
		Summary: &Summary{
			Total:       len(testCases),
			AgentID:     agentInfo.ID,
			AgentPath:   agentInfo.Path,
			Connector:   r.opts.Connector,
			RunsPerCase: r.opts.Runs,
		},
		Environment: NewEnvironment(r.opts.UserID, r.opts.TeamID),
		Metadata: &ReportMetadata{
			StartedAt: startTime,
			InputFile: r.opts.Input,
			Options:   r.opts,
		},
	}

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
	// Get input summary for display
	inputSummary := SummarizeInput(tc.Input, 50)
	r.output.TestStart(tc.ID, inputSummary, runNum)

	startTime := time.Now()

	// Create result
	result := &Result{
		ID:       tc.ID,
		Input:    tc.Input,
		Expected: tc.Expected,
	}

	// Parse input to messages
	messages, err := tc.GetMessages()
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

	// Set options: skip history (input already contains conversation), connector override
	opts := &context.Options{
		Skip: &context.Skip{
			History: true, // Skip history loading - input already contains full conversation
		},
	}
	if r.opts.Connector != "" {
		opts.Connector = r.opts.Connector
	}

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

	// Validate result using asserter
	asserter := NewAsserter()
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

// writeJSONLine writes a JSON line to the writer
func writeJSONLine(writer *bufio.Writer, data interface{}) error {
	line, err := jsoniter.Marshal(data)
	if err != nil {
		return err
	}
	_, err = writer.Write(line)
	if err != nil {
		return err
	}
	_, err = writer.WriteString("\n")
	return err
}

// extractOutput extracts the output from the agent response
func extractOutput(response interface{}) interface{} {
	if response == nil {
		return nil
	}

	// Try to get completion content from context.Response
	if resp, ok := response.(*context.Response); ok {
		if resp.Completion != nil {
			return resp.Completion.Content
		}
		if resp.Next != nil {
			return resp.Next
		}
	}

	return response
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
