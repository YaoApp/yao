package test

import "time"

// ScriptInfo contains information about the script being tested
type ScriptInfo struct {
	// ID is the script identifier (e.g., "scripts.expense.setup")
	ID string `json:"id"`

	// Assistant is the assistant directory name (e.g., "expense")
	Assistant string `json:"assistant"`

	// Module is the module name (e.g., "setup")
	Module string `json:"module"`

	// ScriptPath is the path to the main script file (e.g., "expense/src/setup.ts")
	ScriptPath string `json:"script_path"`

	// TestPath is the path to the test script file (e.g., "expense/src/setup_test.ts")
	TestPath string `json:"test_path"`
}

// ScriptTestCase represents a single script test function
type ScriptTestCase struct {
	// Name is the test function name (e.g., "TestSystemReady")
	Name string `json:"name"`

	// Function is the full function reference
	Function string `json:"function"`
}

// ScriptTestResult represents the result of running a script test function
type ScriptTestResult struct {
	// Name is the test function name
	Name string `json:"name"`

	// Status is the test execution status
	Status Status `json:"status"`

	// DurationMs is the execution duration in milliseconds
	DurationMs int64 `json:"duration_ms"`

	// Error contains the error message if the test failed
	Error string `json:"error,omitempty"`

	// Assertion contains assertion failure details
	Assertion *ScriptAssertionInfo `json:"assertion,omitempty"`

	// Logs contains log messages from the test
	Logs []string `json:"logs,omitempty"`
}

// ScriptAssertionInfo contains details about an assertion failure
type ScriptAssertionInfo struct {
	// Type is the assertion type (e.g., "Equal", "True")
	Type string `json:"type"`

	// Expected is the expected value
	Expected interface{} `json:"expected,omitempty"`

	// Actual is the actual value
	Actual interface{} `json:"actual,omitempty"`

	// Message is the custom failure message
	Message string `json:"message,omitempty"`
}

// ScriptTestSummary contains aggregated statistics for script tests
type ScriptTestSummary struct {
	// Total number of test functions
	Total int `json:"total"`

	// Passed number of test functions that passed
	Passed int `json:"passed"`

	// Failed number of test functions that failed
	Failed int `json:"failed"`

	// Skipped number of test functions that were skipped
	Skipped int `json:"skipped"`

	// DurationMs is the total execution duration in milliseconds
	DurationMs int64 `json:"duration_ms"`
}

// ScriptTestReport represents the complete script test report
type ScriptTestReport struct {
	// Type indicates this is a script test report
	Type string `json:"type"` // "script_test"

	// Script is the script identifier (e.g., "scripts.expense.setup")
	Script string `json:"script"`

	// ScriptPath is the path to the test script file
	ScriptPath string `json:"script_path"`

	// Summary contains aggregated statistics
	Summary *ScriptTestSummary `json:"summary"`

	// Environment contains the test environment configuration
	Environment *Environment `json:"environment"`

	// Results contains individual test results
	Results []*ScriptTestResult `json:"results"`

	// Metadata contains additional report metadata
	Metadata *ScriptTestMetadata `json:"metadata"`
}

// ScriptTestMetadata contains metadata about the script test report
type ScriptTestMetadata struct {
	// StartedAt is when the test run started
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the test run completed
	CompletedAt time.Time `json:"completed_at"`

	// Version is the Yao version
	Version string `json:"version"`
}

// HasFailures returns true if there are any failed tests
func (r *ScriptTestReport) HasFailures() bool {
	return r.Summary.Failed > 0
}

// PassRate returns the pass rate as a percentage (0-100)
func (r *ScriptTestReport) PassRate() float64 {
	if r.Summary.Total == 0 {
		return 0
	}
	return float64(r.Summary.Passed) / float64(r.Summary.Total) * 100
}

// ToReport converts ScriptTestReport to a standard Report for unified reporting
func (r *ScriptTestReport) ToReport() *Report {
	return &Report{
		Summary: &Summary{
			Total:      r.Summary.Total,
			Passed:     r.Summary.Passed,
			Failed:     r.Summary.Failed,
			Skipped:    r.Summary.Skipped,
			DurationMs: r.Summary.DurationMs,
			AgentID:    r.Script,
			AgentPath:  r.ScriptPath,
		},
		Environment: r.Environment,
		Results:     r.toResults(),
		Metadata: &ReportMetadata{
			StartedAt:   r.Metadata.StartedAt,
			CompletedAt: r.Metadata.CompletedAt,
			Version:     r.Metadata.Version,
		},
	}
}

// toResults converts script test results to standard results
func (r *ScriptTestReport) toResults() []*Result {
	results := make([]*Result, len(r.Results))
	for i, sr := range r.Results {
		results[i] = &Result{
			ID:         sr.Name,
			Status:     sr.Status,
			Input:      sr.Name,
			DurationMs: sr.DurationMs,
			Error:      sr.Error,
		}
	}
	return results
}
