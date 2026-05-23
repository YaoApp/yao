package test

import (
	"context"
	"io"
)

// Runner is the interface for test execution
type Runner interface {
	// Run executes all test cases and returns the report
	Run(ctx context.Context) (*Report, error)

	// RunCase executes a single test case
	RunCase(ctx context.Context, tc *Case) (*Result, error)

	// GetAgentInfo returns information about the agent being tested
	GetAgentInfo() *AgentInfo

	// SetProgressCallback sets a callback for progress updates
	SetProgressCallback(callback ProgressCallback)
}

// ProgressCallback is called during test execution to report progress
// Parameters:
//   - current: current test index (1-based)
//   - total: total number of tests
//   - result: result of the current test (nil if not yet completed)
type ProgressCallback func(current, total int, result *Result)

// Reporter is the interface for generating test reports
type Reporter interface {
	// Generate generates a report from the test results
	Generate(report *Report) error

	// Write writes the report to the given writer
	Write(report *Report, w io.Writer) error
}

// Loader is the interface for loading test cases
type Loader interface {
	// Load loads test cases from the input source
	Load() ([]*Case, error)

	// LoadFile loads test cases from a JSONL file
	LoadFile(path string) ([]*Case, error)

	// LoadFromAgent generates test cases using a generator agent
	LoadFromAgent(agentID string, targetInfo *TargetAgentInfo, params map[string]interface{}) ([]*Case, error)

	// LoadFromScript generates test cases using a script
	LoadFromScript(scriptRef string, targetInfo *TargetAgentInfo) ([]*Case, error)
}

// Resolver is the interface for resolving agent information
type Resolver interface {
	// Resolve resolves the agent from options
	// Priority: explicit AgentID > path-based detection
	Resolve(opts *Options) (*AgentInfo, error)

	// ResolveFromPath resolves the agent by traversing up from the input file path
	ResolveFromPath(inputPath string) (*AgentInfo, error)
}

// Validator is the interface for validating test outputs
type Validator interface {
	// Validate compares actual output against expected output
	// Returns nil if validation passes, error otherwise
	Validate(actual, expected interface{}) error

	// ValidateJSON validates JSON outputs with flexible comparison
	ValidateJSON(actual, expected interface{}) error
}

// OutputAdapter adapts agent output to a comparable format
type OutputAdapter interface {
	// Adapt transforms the raw agent output to a normalized format
	Adapt(output interface{}) (interface{}, error)
}

// RunnerFactory creates Runner instances
type RunnerFactory interface {
	// Create creates a new Runner with the given options
	Create(opts *Options) (Runner, error)
}

// ReporterFactory creates Reporter instances
type ReporterFactory interface {
	// Create creates a new Reporter for the given format
	Create(format OutputFormat) (Reporter, error)

	// CreateFromPath creates a Reporter based on output file extension
	CreateFromPath(outputPath string) (Reporter, error)
}

// Hook allows customization of test execution
type Hook interface {
	// BeforeAll is called before any tests run
	BeforeAll(ctx context.Context, cases []*Case) error

	// BeforeEach is called before each test case
	BeforeEach(ctx context.Context, tc *Case) error

	// AfterEach is called after each test case
	AfterEach(ctx context.Context, tc *Case, result *Result) error

	// AfterAll is called after all tests complete
	AfterAll(ctx context.Context, report *Report) error
}

// DefaultHook provides a no-op implementation of Hook
type DefaultHook struct{}

// BeforeAll implements Hook
func (h *DefaultHook) BeforeAll(ctx context.Context, cases []*Case) error {
	return nil
}

// BeforeEach implements Hook
func (h *DefaultHook) BeforeEach(ctx context.Context, tc *Case) error {
	return nil
}

// AfterEach implements Hook
func (h *DefaultHook) AfterEach(ctx context.Context, tc *Case, result *Result) error {
	return nil
}

// AfterAll implements Hook
func (h *DefaultHook) AfterAll(ctx context.Context, report *Report) error {
	return nil
}
