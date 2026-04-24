package test

import (
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/sui/core"
)

// Options holds configuration for a SUI backend test run
type Options struct {
	SUIID    string        `json:"sui_id"`
	Template string        `json:"template"`
	Page     string        `json:"page,omitempty"`
	Run      string        `json:"run,omitempty"`
	Data     string        `json:"data,omitempty"`
	Verbose  bool          `json:"verbose,omitempty"`
	JSON     bool          `json:"json,omitempty"`
	FailFast bool          `json:"fail_fast,omitempty"`
	Timeout  time.Duration `json:"timeout,omitempty"`
}

// PageTestInfo describes a page that has backend tests
type PageTestInfo struct {
	Route          string `json:"route"`
	Name           string `json:"name"`
	BackendFile    string `json:"backend_file"`
	TestFile       string `json:"test_file"`
	PageConfigFile string `json:"page_config_file,omitempty"`
	Prefix         string `json:"prefix"`
}

// TestCase represents a single test function discovered in a backend_test.ts file
type TestCase struct {
	Name     string `json:"name"`
	Function string `json:"function"`
}

// TestResult represents the outcome of a single test function
type TestResult struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	DurationMs int64          `json:"duration_ms"`
	Error      string         `json:"error,omitempty"`
	Assertion  *AssertionInfo `json:"assertion,omitempty"`
	Logs       []string       `json:"logs,omitempty"`
}

// AssertionInfo contains details about an assertion failure
type AssertionInfo struct {
	Type     string      `json:"type"`
	Expected interface{} `json:"expected,omitempty"`
	Actual   interface{} `json:"actual,omitempty"`
	Message  string      `json:"message,omitempty"`
}

// TestSummary contains aggregated statistics
type TestSummary struct {
	Total      int   `json:"total"`
	Passed     int   `json:"passed"`
	Failed     int   `json:"failed"`
	Skipped    int   `json:"skipped"`
	DurationMs int64 `json:"duration_ms"`
}

// Report represents the complete test report for yao sui test
type Report struct {
	Type     string        `json:"type"`
	SUIID    string        `json:"sui_id"`
	Template string        `json:"template"`
	Summary  *TestSummary  `json:"summary"`
	Pages    []*PageReport `json:"pages"`
	Metadata *TestMetadata `json:"metadata"`
}

// PageReport contains results for a single page
type PageReport struct {
	Route   string        `json:"route"`
	Results []*TestResult `json:"results"`
}

// TestMetadata contains metadata about the test run
type TestMetadata struct {
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// HasFailures returns true if any tests failed
func (r *Report) HasFailures() bool {
	return r.Summary.Failed > 0
}

// LoadPageConfig reads and parses a .cfg file for a SUI page
func LoadPageConfig(file string) (*core.PageConfig, error) {
	if exist, _ := application.App.Exists(file); !exist {
		return nil, nil
	}

	source, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	cfg := core.PageConfig{}
	if err := jsoniter.Unmarshal(source, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
