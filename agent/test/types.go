package test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/yaoapp/yao/agent/context"
)

// Status represents the status of a test case execution
type Status string

const (
	// StatusPassed indicates the test passed
	StatusPassed Status = "passed"
	// StatusFailed indicates the test failed
	StatusFailed Status = "failed"
	// StatusSkipped indicates the test was skipped
	StatusSkipped Status = "skipped"
	// StatusError indicates a runtime error occurred
	StatusError Status = "error"
	// StatusTimeout indicates the test timed out
	StatusTimeout Status = "timeout"
)

// OutputFormat represents the output format for test reports
type OutputFormat string

const (
	// FormatJSON outputs JSON format (for CI integration)
	FormatJSON OutputFormat = "json"
	// FormatHTML outputs HTML format (for human review)
	FormatHTML OutputFormat = "html"
	// FormatMarkdown outputs Markdown format (for documentation)
	FormatMarkdown OutputFormat = "markdown"
)

// StabilityClass represents the stability classification of a test case
type StabilityClass string

const (
	// StabilityStable indicates 100% pass rate
	StabilityStable StabilityClass = "stable"
	// StabilityMostlyStable indicates 80-99% pass rate
	StabilityMostlyStable StabilityClass = "mostly_stable"
	// StabilityUnstable indicates 50-79% pass rate
	StabilityUnstable StabilityClass = "unstable"
	// StabilityHighlyUnstable indicates < 50% pass rate
	StabilityHighlyUnstable StabilityClass = "highly_unstable"
)

// InputMode represents the input mode for test cases
type InputMode string

const (
	// InputModeFile indicates input from a JSONL file
	InputModeFile InputMode = "file"
	// InputModeMessage indicates input from a direct message string
	InputModeMessage InputMode = "message"
	// InputModeScript indicates script test mode (testing agent handler scripts)
	InputModeScript InputMode = "script"
)

// Options represents the configuration options for running tests
type Options struct {
	// Input/Output
	// ===============================

	// Input is the input source: either a file path or a direct message
	Input string `json:"input"`

	// InputMode is the input mode (auto-detected from Input)
	InputMode InputMode `json:"input_mode"`

	// OutputFile is the path to write the test report
	// Format is determined by file extension (.json, .html, .md)
	OutputFile string `json:"output_file"`

	// Agent Selection
	// ===============================

	// AgentID is the explicit agent ID to test (optional)
	// If not set, agent is resolved from InputFile path
	AgentID string `json:"agent_id,omitempty"`

	// Connector overrides the agent's default connector (optional)
	Connector string `json:"connector,omitempty"`

	// Test Environment
	// ===============================

	// UserID is the test user ID (-u flag)
	UserID string `json:"user_id,omitempty"`

	// TeamID is the test team ID (-t flag)
	TeamID string `json:"team_id,omitempty"`

	// Locale is the locale for the test context (default: "en-us")
	Locale string `json:"locale,omitempty"`

	// ContextFile is the path to a JSON file containing custom context data (-ctx flag)
	// This allows full customization of authorized info, metadata, etc.
	ContextFile string `json:"context_file,omitempty"`

	// ContextData is the parsed context data from ContextFile
	// This is populated internally after loading the file
	ContextData *ContextConfig `json:"-"`

	// Execution
	// ===============================

	// Timeout is the default timeout for each test case
	// Can be overridden per test case
	Timeout time.Duration `json:"timeout,omitempty"`

	// Parallel is the number of tests to run in parallel
	// Default is 1 (sequential execution)
	Parallel int `json:"parallel,omitempty"`

	// Runs is the number of times to run each test case
	// Default is 1. When > 1, stability metrics are collected
	Runs int `json:"runs,omitempty"`

	// Reporting
	// ===============================

	// ReporterID is the reporter agent ID for custom report generation
	// If not set, default JSONL format is used
	ReporterID string `json:"reporter_id,omitempty"`

	// Behavior
	// ===============================

	// Verbose enables verbose output during test execution
	Verbose bool `json:"verbose,omitempty"`

	// FailFast stops execution on first failure
	FailFast bool `json:"fail_fast,omitempty"`

	// Run is a regex pattern to filter which tests to run (similar to go test -run)
	// Only tests matching the pattern will be executed
	// Example: "TestSystem" matches TestSystemReady, TestSystemError, etc.
	Run string `json:"run,omitempty"`

	// BeforeAll is the global before script (e.g., "scripts:tests.env.BeforeAll")
	// Called once before all test cases
	BeforeAll string `json:"before_all,omitempty"`

	// AfterAll is the global after script (e.g., "scripts:tests.env.AfterAll")
	// Called once after all test cases
	AfterAll string `json:"after_all,omitempty"`

	// DryRun generates test cases without running them
	// Useful for previewing agent-generated test cases
	DryRun bool `json:"dry_run,omitempty"`

	// Simulator is the default simulator agent ID for dynamic mode
	// Can be overridden per test case in JSONL
	Simulator string `json:"simulator,omitempty"`
}

// ContextConfig represents custom context configuration from JSON file
// This allows full customization of the test context including authorized info
type ContextConfig struct {
	// ChatID is the chat session identifier
	// Used to maintain session state across turns in dynamic tests
	ChatID string `json:"chat_id,omitempty"`

	// Authorized contains custom authorization data
	Authorized *AuthorizedConfig `json:"authorized,omitempty"`

	// Metadata contains custom metadata to pass to the context
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Client contains custom client information
	Client *ClientConfig `json:"client,omitempty"`

	// Locale overrides the locale setting
	Locale string `json:"locale,omitempty"`

	// Referer overrides the referer setting
	Referer string `json:"referer,omitempty"`
}

// AuthorizedConfig represents custom authorization configuration
// Matches the structure of types.AuthorizedInfo from openapi/oauth/types
type AuthorizedConfig struct {
	// Sub is the subject identifier (JWT sub claim)
	Sub string `json:"sub,omitempty"`

	// ClientID is the OAuth client ID
	ClientID string `json:"client_id,omitempty"`

	// Scope is the access scope
	Scope string `json:"scope,omitempty"`

	// SessionID is the session identifier
	SessionID string `json:"session_id,omitempty"`

	// UserID is the user identifier
	UserID string `json:"user_id,omitempty"`

	// TeamID is the team identifier
	TeamID string `json:"team_id,omitempty"`

	// TenantID is the tenant identifier
	TenantID string `json:"tenant_id,omitempty"`

	// RememberMe is the remember me flag
	RememberMe bool `json:"remember_me,omitempty"`

	// Constraints contains data access constraints (set by ACL enforcement)
	Constraints *DataConstraintsConfig `json:"constraints,omitempty"`
}

// DataConstraintsConfig represents data access constraints
// Matches the structure of types.DataConstraints from openapi/oauth/types
type DataConstraintsConfig struct {
	// OwnerOnly - only access owner's data
	OwnerOnly bool `json:"owner_only,omitempty"`

	// CreatorOnly - only access creator's data
	CreatorOnly bool `json:"creator_only,omitempty"`

	// EditorOnly - only access editor's data
	EditorOnly bool `json:"editor_only,omitempty"`

	// TeamOnly - only access team's data (filter by team_id)
	TeamOnly bool `json:"team_only,omitempty"`

	// Extra contains user-defined constraints (department, region, etc.)
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// ClientConfig represents custom client configuration
type ClientConfig struct {
	// Type is the client type (e.g., "web", "mobile", "test")
	Type string `json:"type,omitempty"`

	// UserAgent is the client user agent string
	UserAgent string `json:"user_agent,omitempty"`

	// IP is the client IP address
	IP string `json:"ip,omitempty"`
}

// Environment configures the test execution context
type Environment struct {
	// UserID is the user ID for authorized info (-u flag)
	UserID string `json:"user_id"`

	// TeamID is the team ID for authorized info (-t flag)
	TeamID string `json:"team_id"`

	// Locale is the locale (default: "en-us")
	Locale string `json:"locale"`

	// ClientType is the client type (default: "test")
	ClientType string `json:"client_type"`

	// ClientIP is the client IP (default: "127.0.0.1")
	ClientIP string `json:"client_ip"`

	// Referer is the request referer (default: "test")
	Referer string `json:"referer"`

	// Accept is the accept format (default: "standard")
	Accept string `json:"accept"`

	// ContextConfig contains custom context configuration (from -ctx flag)
	ContextConfig *ContextConfig `json:"-"`
}

// NewEnvironment creates a new test environment with defaults
func NewEnvironment(userID, teamID string) *Environment {
	env := &Environment{
		UserID:     userID,
		TeamID:     teamID,
		Locale:     "en-us",
		ClientType: "test",
		ClientIP:   "127.0.0.1",
		Referer:    "test",
		Accept:     "standard",
	}

	// Apply defaults if not set
	if env.UserID == "" {
		env.UserID = "test-user"
	}
	if env.TeamID == "" {
		env.TeamID = "test-team"
	}

	return env
}

// NewEnvironmentWithContext creates a new test environment with custom context config
func NewEnvironmentWithContext(userID, teamID string, ctxConfig *ContextConfig) *Environment {
	env := NewEnvironment(userID, teamID)

	if ctxConfig == nil {
		return env
	}

	env.ContextConfig = ctxConfig

	// Override with context config values
	if ctxConfig.Locale != "" {
		env.Locale = ctxConfig.Locale
	}
	if ctxConfig.Referer != "" {
		env.Referer = ctxConfig.Referer
	}
	if ctxConfig.Client != nil {
		if ctxConfig.Client.Type != "" {
			env.ClientType = ctxConfig.Client.Type
		}
		if ctxConfig.Client.IP != "" {
			env.ClientIP = ctxConfig.Client.IP
		}
	}
	if ctxConfig.Authorized != nil {
		if ctxConfig.Authorized.UserID != "" {
			env.UserID = ctxConfig.Authorized.UserID
		}
		// TeamID takes precedence over TenantID for team override
		if ctxConfig.Authorized.TeamID != "" {
			env.TeamID = ctxConfig.Authorized.TeamID
		} else if ctxConfig.Authorized.TenantID != "" {
			env.TeamID = ctxConfig.Authorized.TenantID
		}
	}

	return env
}

// LoadContextConfig loads context configuration from a JSON file
func LoadContextConfig(filePath string) (*ContextConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read context file: %w", err)
	}

	var config ContextConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse context file: %w", err)
	}

	return &config, nil
}

// Case represents a single test case loaded from JSONL
type Case struct {
	// ID is the unique identifier for this test case (e.g., "T001")
	ID string `json:"id"`

	// Input is the test input, can be:
	// - string: simple text input
	// - map (Message): single message with role and content
	// - []map ([]Message): conversation history
	Input interface{} `json:"input"`

	// Expected is the expected output for validation (optional)
	// If set, the actual output will be compared against this
	Expected interface{} `json:"expected,omitempty"`

	// Assert defines custom assertion rules (optional)
	// If set, these rules will be used instead of simple expected comparison
	// Can be a single assertion or an array of assertions
	Assert interface{} `json:"assert,omitempty"`

	// Environment (per-test case, can be overridden by command line flags)
	// ===============================

	// UserID is the user ID for this test case (overridden by -u flag)
	UserID string `json:"user,omitempty"`

	// TeamID is the team ID for this test case (overridden by -t flag)
	TeamID string `json:"team,omitempty"`

	// Metadata contains additional metadata for the test case
	// This is passed to ctx.Metadata and can be used by Create Hook
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Options contains context options for this test case
	// Supports: connector, skip (history, trace, output, keyword, search), mode
	Options *CaseOptions `json:"options,omitempty"`

	// Skip indicates whether to skip this test case
	Skip bool `json:"skip,omitempty"`

	// Timeout overrides the default timeout for this test case
	// Format: "30s", "1m", "2m30s"
	Timeout string `json:"timeout,omitempty"`

	// Before script function (e.g., "scripts:tests.env.Before")
	// Called before the test case runs, returns data passed to After
	Before string `json:"before,omitempty"`

	// After script function (e.g., "scripts:tests.env.After")
	// Called after the test case completes (pass or fail)
	After string `json:"after,omitempty"`

	// Dynamic Mode Fields
	// ===============================

	// Simulator configures the user simulator for dynamic testing
	// When set, the test runs in dynamic mode with multi-turn conversation
	Simulator *Simulator `json:"simulator,omitempty"`

	// Checkpoints define validation points for dynamic testing
	// Each checkpoint is checked after every agent response
	Checkpoints []*Checkpoint `json:"checkpoints,omitempty"`

	// MaxTurns is the maximum number of conversation turns (default: 20)
	MaxTurns int `json:"max_turns,omitempty"`
}

// Simulator configures the user simulator for dynamic testing
type Simulator struct {
	// Use is the simulator agent ID (no prefix needed)
	Use string `json:"use"`

	// Options for the simulator agent
	Options *SimulatorOptions `json:"options,omitempty"`
}

// SimulatorOptions configures simulator behavior
type SimulatorOptions struct {
	// Metadata passed to the simulator agent
	// Common fields: persona, goal, style
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Connector overrides the simulator's default connector
	Connector string `json:"connector,omitempty"`
}

// Checkpoint defines a validation point in dynamic testing
type Checkpoint struct {
	// ID is the unique identifier for this checkpoint
	ID string `json:"id"`

	// Description is a human-readable description
	Description string `json:"description,omitempty"`

	// Assert defines the assertion to validate
	// Same format as Case.Assert
	Assert interface{} `json:"assert"`

	// After specifies checkpoint IDs that must be reached before this one
	// Used to enforce ordering (e.g., "ask_type" must come before "confirm")
	After []string `json:"after,omitempty"`

	// Required indicates if this checkpoint must be reached (default: true)
	// Optional checkpoints don't cause test failure if not reached
	Required *bool `json:"required,omitempty"`
}

// CaseOptions represents per-test-case context options
// Maps to context.Options fields
type CaseOptions struct {
	// Connector overrides the agent's default connector
	Connector string `json:"connector,omitempty"`

	// Skip configuration
	Skip *CaseSkipOptions `json:"skip,omitempty"`

	// DisableGlobalPrompts temporarily disables global prompts for this request
	DisableGlobalPrompts bool `json:"disable_global_prompts,omitempty"`

	// Search mode, default is true (use pointer to distinguish unset from false)
	Search *bool `json:"search,omitempty"`

	// Mode is the agent mode (default: "chat")
	Mode string `json:"mode,omitempty"`

	// Metadata for passing custom data to hooks (e.g., scenario selection)
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CaseSkipOptions represents skip configuration for a test case
// Maps to context.Skip fields
type CaseSkipOptions struct {
	History bool `json:"history,omitempty"` // Skip history loading
	Trace   bool `json:"trace,omitempty"`   // Skip trace logging
	Output  bool `json:"output,omitempty"`  // Skip output to client
	Keyword bool `json:"keyword,omitempty"` // Skip keyword extraction
	Search  bool `json:"search,omitempty"`  // Skip auto search
}

// Assertion represents a single assertion rule
type Assertion struct {
	// Type is the assertion type:
	// - "equals": exact match (default if expected is set)
	// - "contains": output contains the expected string/value
	// - "not_contains": output does not contain the string/value
	// - "json_path": extract value using JSON path and compare
	// - "regex": match output against regex pattern
	// - "script": run a custom assertion script
	// - "type": check output type (string, object, array, number, boolean)
	// - "schema": validate against JSON schema
	// - "agent": use an agent to validate the response
	Type string `json:"type"`

	// Value is the expected value or pattern (depends on type)
	Value interface{} `json:"value,omitempty"`

	// Path is the JSON path for json_path assertions (e.g., "$.need_search")
	Path string `json:"path,omitempty"`

	// Script is the assertion script name for script assertions
	// The script receives (output, input, expected) and returns {pass: bool, message: string}
	Script string `json:"script,omitempty"`

	// Use specifies the agent/script for validation
	// For agent assertions: "agents:tests.validator-agent" (with prefix)
	// For script assertions: "scripts:tests.validate" (with prefix)
	Use string `json:"use,omitempty"`

	// Options for agent-driven assertions (aligned with context.Options)
	Options *AssertionOptions `json:"options,omitempty"`

	// Message is a custom failure message
	Message string `json:"message,omitempty"`

	// Negate inverts the assertion result
	Negate bool `json:"negate,omitempty"`
}

// AssertionOptions for agent-driven assertions
type AssertionOptions struct {
	// Connector overrides the agent's default connector
	Connector string `json:"connector,omitempty"`

	// Metadata contains custom data passed to the validator agent
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AssertionResult represents the result of an assertion
type AssertionResult struct {
	// Passed indicates whether the assertion passed
	Passed bool `json:"passed"`

	// Message describes the assertion result
	Message string `json:"message,omitempty"`

	// Assertion is the original assertion that was evaluated
	Assertion *Assertion `json:"assertion,omitempty"`

	// Actual is the actual value that was compared
	Actual interface{} `json:"actual,omitempty"`

	// Expected is the expected value
	Expected interface{} `json:"expected,omitempty"`
}

// GetEnvironment returns the effective test environment for this test case
// Priority: command line flags > context config > test case fields > defaults
func (tc *Case) GetEnvironment(opts *Options) *Environment {
	// Start with context config if available, otherwise use defaults
	var env *Environment
	if opts != nil && opts.ContextData != nil {
		env = NewEnvironmentWithContext("", "", opts.ContextData)
	} else {
		env = NewEnvironment("", "")
	}

	// Apply test case specific values
	if tc.UserID != "" {
		env.UserID = tc.UserID
	}
	if tc.TeamID != "" {
		env.TeamID = tc.TeamID
	}

	// Apply command line overrides (highest priority)
	if opts != nil {
		if opts.UserID != "" {
			env.UserID = opts.UserID
		}
		if opts.TeamID != "" {
			env.TeamID = opts.TeamID
		}
		if opts.Locale != "" {
			env.Locale = opts.Locale
		}
	}

	return env
}

// GetMessages converts the Input to a slice of context.Message
// This handles all input formats: string, Message, []Message
func (tc *Case) GetMessages() ([]context.Message, error) {
	return ParseInput(tc.Input)
}

// GetMessagesWithOptions converts the Input to a slice of context.Message with options
// This handles all input formats: string, Message, []Message
// It also processes file:// references in content parts
func (tc *Case) GetMessagesWithOptions(opts *InputOptions) ([]context.Message, error) {
	return ParseInputWithOptions(tc.Input, opts)
}

// GetTimeout returns the timeout duration for this test case
// Returns the override timeout if set, otherwise returns the default
func (tc *Case) GetTimeout(defaultTimeout time.Duration) time.Duration {
	if tc.Timeout == "" {
		return defaultTimeout
	}
	d, err := time.ParseDuration(tc.Timeout)
	if err != nil {
		return defaultTimeout
	}
	return d
}

// Result represents the result of running a single test case
type Result struct {
	// ID is the test case identifier
	ID string `json:"id"`

	// Status is the test execution status
	Status Status `json:"status"`

	// Input is the original test input (for reference in reports)
	Input interface{} `json:"input"`

	// Output is the actual output from the agent
	Output interface{} `json:"output,omitempty"`

	// Expected is the expected output (if specified in test case)
	Expected interface{} `json:"expected,omitempty"`

	// DurationMs is the execution duration in milliseconds
	DurationMs int64 `json:"duration_ms"`

	// Error contains the error message if status is failed/error/timeout
	Error string `json:"error,omitempty"`

	// Options contains the context options used for this test case
	Options *CaseOptions `json:"options,omitempty"`

	// Metadata contains additional result metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RunDetail represents the result of a single run in stability testing
type RunDetail struct {
	// Run is the run number (1-based)
	Run int `json:"run"`

	// Status is the execution status for this run
	Status Status `json:"status"`

	// DurationMs is the execution duration in milliseconds
	DurationMs int64 `json:"duration_ms"`

	// Output is the output from this run
	Output interface{} `json:"output,omitempty"`

	// Error contains the error message if this run failed
	Error string `json:"error,omitempty"`
}

// StabilityResult represents the stability analysis result for a test case
type StabilityResult struct {
	// ID is the test case identifier
	ID string `json:"id"`

	// Input is the original test input
	Input interface{} `json:"input"`

	// Expected is the expected output (if specified)
	Expected interface{} `json:"expected,omitempty"`

	// Runs is the total number of runs
	Runs int `json:"runs"`

	// Passed is the number of runs that passed
	Passed int `json:"passed"`

	// Failed is the number of runs that failed
	Failed int `json:"failed"`

	// PassRate is the pass rate percentage (0-100)
	PassRate float64 `json:"pass_rate"`

	// Consistency is a measure of output consistency (0-1)
	// 1.0 means all outputs are identical, lower values indicate variation
	Consistency float64 `json:"consistency"`

	// Stable indicates whether the test is considered stable
	Stable bool `json:"stable"`

	// StabilityClass is the stability classification
	StabilityClass StabilityClass `json:"stability_class"`

	// Timing statistics
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	MinDurationMs  int64   `json:"min_duration_ms"`
	MaxDurationMs  int64   `json:"max_duration_ms"`
	StdDeviationMs float64 `json:"std_deviation_ms"`

	// RunDetails contains details for each run
	RunDetails []*RunDetail `json:"run_details"`
}

// CalculateStability calculates stability metrics from run details
func (sr *StabilityResult) CalculateStability() {
	if len(sr.RunDetails) == 0 {
		return
	}

	sr.Runs = len(sr.RunDetails)
	sr.Passed = 0
	sr.Failed = 0

	var totalDuration int64
	sr.MinDurationMs = math.MaxInt64
	sr.MaxDurationMs = 0

	for _, rd := range sr.RunDetails {
		if rd.Status == StatusPassed {
			sr.Passed++
		} else {
			sr.Failed++
		}

		totalDuration += rd.DurationMs
		if rd.DurationMs < sr.MinDurationMs {
			sr.MinDurationMs = rd.DurationMs
		}
		if rd.DurationMs > sr.MaxDurationMs {
			sr.MaxDurationMs = rd.DurationMs
		}
	}

	// Calculate pass rate
	sr.PassRate = float64(sr.Passed) / float64(sr.Runs) * 100

	// Calculate average duration
	sr.AvgDurationMs = float64(totalDuration) / float64(sr.Runs)

	// Calculate standard deviation
	var sumSquares float64
	for _, rd := range sr.RunDetails {
		diff := float64(rd.DurationMs) - sr.AvgDurationMs
		sumSquares += diff * diff
	}
	sr.StdDeviationMs = math.Sqrt(sumSquares / float64(sr.Runs))

	// Determine stability classification
	sr.StabilityClass = ClassifyStability(sr.PassRate)
	sr.Stable = sr.PassRate == 100

	// Calculate consistency (simplified: based on pass rate)
	sr.Consistency = sr.PassRate / 100
}

// ClassifyStability returns the stability classification based on pass rate
func ClassifyStability(passRate float64) StabilityClass {
	switch {
	case passRate == 100:
		return StabilityStable
	case passRate >= 80:
		return StabilityMostlyStable
	case passRate >= 50:
		return StabilityUnstable
	default:
		return StabilityHighlyUnstable
	}
}

// Summary contains aggregated statistics for the test run
type Summary struct {
	// Total number of test cases
	Total int `json:"total"`

	// Passed number of test cases that passed
	Passed int `json:"passed"`

	// Failed number of test cases that failed
	Failed int `json:"failed"`

	// Skipped number of test cases that were skipped
	Skipped int `json:"skipped"`

	// Errors number of test cases with runtime errors
	Errors int `json:"errors"`

	// Timeouts number of test cases that timed out
	Timeouts int `json:"timeouts"`

	// DurationMs is the total execution duration in milliseconds
	DurationMs int64 `json:"duration_ms"`

	// AgentID is the ID of the agent being tested
	AgentID string `json:"agent_id"`

	// AgentPath is the file path of the agent (for path-based resolution)
	AgentPath string `json:"agent_path,omitempty"`

	// Connector is the connector used for the test
	Connector string `json:"connector"`

	// Stability metrics (when Runs > 1)
	// ===============================

	// RunsPerCase is the number of runs per test case
	RunsPerCase int `json:"runs_per_case,omitempty"`

	// TotalRuns is the total number of runs (Total * RunsPerCase)
	TotalRuns int `json:"total_runs,omitempty"`

	// OverallPassRate is the overall pass rate percentage
	OverallPassRate float64 `json:"overall_pass_rate,omitempty"`

	// StableCases is the number of cases with 100% pass rate
	StableCases int `json:"stable_cases,omitempty"`

	// UnstableCases is the number of cases with < 100% pass rate
	UnstableCases int `json:"unstable_cases,omitempty"`
}

// Report represents the complete test report
type Report struct {
	// Summary contains aggregated statistics
	Summary *Summary `json:"summary"`

	// Environment contains the test environment configuration
	Environment *Environment `json:"environment,omitempty"`

	// Results contains individual test results (for single run)
	Results []*Result `json:"results,omitempty"`

	// StabilityResults contains stability analysis results (for multiple runs)
	StabilityResults []*StabilityResult `json:"stability_results,omitempty"`

	// Metadata contains additional report metadata
	Metadata *ReportMetadata `json:"metadata"`
}

// ReportMetadata contains metadata about the test report
type ReportMetadata struct {
	// StartedAt is when the test run started
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the test run completed
	CompletedAt time.Time `json:"completed_at"`

	// Version is the Yao version
	Version string `json:"version"`

	// InputFile is the path to the input file
	InputFile string `json:"input_file"`

	// OutputFile is the path to the output file
	OutputFile string `json:"output_file"`

	// Options contains the test options used
	Options *Options `json:"options,omitempty"`
}

// HasFailures returns true if there are any failed, error, or timeout tests
func (r *Report) HasFailures() bool {
	return r.Summary.Failed > 0 || r.Summary.Errors > 0 || r.Summary.Timeouts > 0
}

// PassRate returns the pass rate as a percentage (0-100)
func (r *Report) PassRate() float64 {
	if r.Summary.Total == 0 {
		return 0
	}
	return float64(r.Summary.Passed) / float64(r.Summary.Total) * 100
}

// IsStabilityTest returns true if this is a stability test (multiple runs)
func (r *Report) IsStabilityTest() bool {
	return r.Summary.RunsPerCase > 1
}

// AgentInfo contains information about the agent being tested
type AgentInfo struct {
	// ID is the agent identifier
	ID string `json:"id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Description is the agent description
	Description string `json:"description,omitempty"`

	// Path is the file system path to the agent
	Path string `json:"path"`

	// Connector is the default connector
	Connector string `json:"connector"`

	// Type is the agent type (e.g., "worker", "assistant")
	Type string `json:"type,omitempty"`
}

// ReporterInput is the input passed to a custom reporter agent
type ReporterInput struct {
	// Report is the test report to format
	Report *Report `json:"report"`

	// Format is the desired output format
	Format string `json:"format"`

	// Options contains additional formatting options
	Options *ReporterOptions `json:"options,omitempty"`
}

// ReporterOptions contains options for custom reporter agents
type ReporterOptions struct {
	// Verbose includes detailed output in the report
	Verbose bool `json:"verbose,omitempty"`

	// IncludeOutputs includes full outputs in the report
	IncludeOutputs bool `json:"include_outputs,omitempty"`

	// IncludeInputs includes full inputs in the report
	IncludeInputs bool `json:"include_inputs,omitempty"`

	// MaxOutputLength limits the output length in the report
	MaxOutputLength int `json:"max_output_length,omitempty"`

	// Theme is the report theme (for HTML reports)
	Theme string `json:"theme,omitempty"`

	// Title is the report title
	Title string `json:"title,omitempty"`
}
