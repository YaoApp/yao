package test

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
)

// JSONLLoader loads test cases from JSONL files
type JSONLLoader struct{}

// NewLoader creates a new JSONL loader
func NewLoader() Loader {
	return &JSONLLoader{}
}

// Load loads test cases from the default input source
// This is a placeholder - actual implementation would use configured path
func (l *JSONLLoader) Load() ([]*Case, error) {
	return nil, fmt.Errorf("Load() requires explicit path, use LoadFile() instead")
}

// LoadFile loads test cases from a JSONL file
// If path is relative and YAO_ROOT is set, resolves relative to YAO_ROOT
func (l *JSONLLoader) LoadFile(path string) ([]*Case, error) {
	// Resolve path relative to YAO_ROOT if it's a relative path
	resolvedPath := ResolvePathWithYaoRoot(path)

	file, err := os.Open(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	var cases []*Case
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Increase buffer size for long lines
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comments (lines starting with //)
		if strings.HasPrefix(line, "//") {
			continue
		}

		var tc Case
		if err := jsoniter.UnmarshalFromString(line, &tc); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}

		// Validate required fields
		if tc.ID == "" {
			return nil, fmt.Errorf("line %d: missing required field 'id'", lineNum)
		}
		if tc.Input == nil {
			return nil, fmt.Errorf("line %d (id=%s): missing required field 'input'", lineNum, tc.ID)
		}

		cases = append(cases, &tc)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	if len(cases) == 0 {
		return nil, fmt.Errorf("no test cases found in %s", path)
	}

	return cases, nil
}

// ValidateTestCases validates a slice of test cases
func ValidateTestCases(cases []*Case) error {
	ids := make(map[string]bool)

	for i, tc := range cases {
		// Check for duplicate IDs
		if ids[tc.ID] {
			return fmt.Errorf("duplicate test case ID: %s", tc.ID)
		}
		ids[tc.ID] = true

		// Validate input can be parsed
		if _, err := tc.GetMessages(); err != nil {
			return fmt.Errorf("test case %s (index %d): invalid input: %w", tc.ID, i, err)
		}

		// Validate timeout format if specified
		if tc.Timeout != "" {
			// GetTimeout returns a duration, parsing error would return default
			// We validate by checking if the string is parseable
			if _, err := time.ParseDuration(tc.Timeout); err != nil {
				return fmt.Errorf("test case %s: invalid timeout format: %s", tc.ID, tc.Timeout)
			}
		}
	}

	return nil
}

// FilterTestCases filters test cases based on criteria
func FilterTestCases(cases []*Case, filter func(*Case) bool) []*Case {
	var result []*Case
	for _, tc := range cases {
		if filter(tc) {
			result = append(result, tc)
		}
	}
	return result
}

// FilterSkipped returns test cases that are not skipped
func FilterSkipped(cases []*Case) []*Case {
	return FilterTestCases(cases, func(tc *Case) bool {
		return !tc.Skip
	})
}

// FilterByIDs returns test cases matching the given IDs
func FilterByIDs(cases []*Case, ids []string) []*Case {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	return FilterTestCases(cases, func(tc *Case) bool {
		return idSet[tc.ID]
	})
}

// FilterByPattern returns test cases whose ID matches the given regex pattern
func FilterByPattern(cases []*Case, pattern *regexp.Regexp) []*Case {
	return FilterTestCases(cases, func(tc *Case) bool {
		return pattern.MatchString(tc.ID)
	})
}

// LoadFromAgent generates test cases using a generator agent
func (l *JSONLLoader) LoadFromAgent(agentID string, targetInfo *TargetAgentInfo, params map[string]interface{}) ([]*Case, error) {
	return GenerateTestCases(agentID, targetInfo, params)
}

// LoadFromScript generates test cases using a script
// scriptRef format: "module.FunctionName" (e.g., "tests.gen.Generate")
func (l *JSONLLoader) LoadFromScript(scriptRef string, targetInfo *TargetAgentInfo) ([]*Case, error) {
	// Parse script reference
	parts := strings.Split(scriptRef, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid script reference format: %s (expected 'module.Function')", scriptRef)
	}

	// Build process name: scripts.module.Function
	processName := "scripts." + scriptRef

	// Execute via process
	p, err := process.Of(processName, targetInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create process %s: %w", processName, err)
	}

	result, err := p.Exec()
	if err != nil {
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	// Parse result as test cases
	return convertToCases(result)
}
