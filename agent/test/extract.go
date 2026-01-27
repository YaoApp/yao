package test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// ExtractOptions represents options for extracting test results
type ExtractOptions struct {
	// InputFile is the path to the output JSONL file from test run
	InputFile string

	// OutputDir is the directory to write extracted files (default: same as input file)
	OutputDir string

	// Format is the output format: "markdown" (default), "json"
	Format string
}

// Extractor extracts test results to individual files for review
type Extractor struct {
	opts *ExtractOptions
}

// NewExtractor creates a new extractor
func NewExtractor(opts *ExtractOptions) *Extractor {
	if opts.Format == "" {
		opts.Format = "markdown"
	}
	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Dir(opts.InputFile)
	}
	return &Extractor{opts: opts}
}

// Extract reads the test output file and extracts results to individual files
func (e *Extractor) Extract() ([]string, error) {
	// Read the JSONL file
	data, err := os.ReadFile(e.opts.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}

	// Parse the JSON (the output file is a single JSON object, not JSONL)
	var report Report
	if err := jsoniter.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse test report: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(e.opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	var extractedFiles []string

	// Extract each result
	for _, result := range report.Results {
		var filename string
		var content string

		switch e.opts.Format {
		case "markdown":
			filename = filepath.Join(e.opts.OutputDir, result.ID+".md")
			content = e.formatMarkdown(result)
		case "json":
			filename = filepath.Join(e.opts.OutputDir, result.ID+".json")
			jsonBytes, err := jsoniter.MarshalIndent(result, "", "  ")
			if err != nil {
				return extractedFiles, fmt.Errorf("failed to marshal result %s: %w", result.ID, err)
			}
			content = string(jsonBytes)
		default:
			return nil, fmt.Errorf("unsupported format: %s", e.opts.Format)
		}

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return extractedFiles, fmt.Errorf("failed to write file %s: %w", filename, err)
		}

		extractedFiles = append(extractedFiles, filename)
	}

	return extractedFiles, nil
}

// formatMarkdown formats a single test result as Markdown
func (e *Extractor) formatMarkdown(result *Result) string {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("# %s\n\n", result.ID))

	// Status badge
	switch result.Status {
	case StatusPassed:
		sb.WriteString("**Status**: ✅ PASSED\n\n")
	case StatusFailed:
		sb.WriteString("**Status**: ❌ FAILED\n\n")
	case StatusError:
		sb.WriteString("**Status**: ⚠️ ERROR\n\n")
	case StatusTimeout:
		sb.WriteString("**Status**: ⏱️ TIMEOUT\n\n")
	case StatusSkipped:
		sb.WriteString("**Status**: ⏭️ SKIPPED\n\n")
	}

	// Duration
	sb.WriteString(fmt.Sprintf("**Duration**: %dms\n\n", result.DurationMs))

	// Error (if any)
	if result.Error != "" {
		sb.WriteString("## Error\n\n")
		sb.WriteString("```\n")
		sb.WriteString(result.Error)
		sb.WriteString("\n```\n\n")
	}

	// Input
	sb.WriteString("## Input\n\n")
	sb.WriteString("```markdown\n")
	sb.WriteString(formatInputAsString(result.Input))
	sb.WriteString("\n```\n\n")

	// Output
	sb.WriteString("## Output\n\n")
	output := formatOutputAsString(result.Output)
	// Remove markdown code block wrapper if present
	output = strings.TrimPrefix(output, "```markdown\n")
	output = strings.TrimSuffix(output, "\n```")
	output = strings.TrimSuffix(output, "```")
	sb.WriteString(output)
	sb.WriteString("\n")

	return sb.String()
}

// formatInputAsString converts input to string format
func formatInputAsString(input interface{}) string {
	switch v := input.(type) {
	case string:
		return v
	case map[string]interface{}:
		// Single message format
		if content, ok := v["content"].(string); ok {
			return content
		}
		// Fallback to JSON
		jsonBytes, _ := jsoniter.MarshalIndent(v, "", "  ")
		return string(jsonBytes)
	case []interface{}:
		// Array of messages - extract content from last user message
		for i := len(v) - 1; i >= 0; i-- {
			if msg, ok := v[i].(map[string]interface{}); ok {
				if role, ok := msg["role"].(string); ok && role == "user" {
					if content, ok := msg["content"].(string); ok {
						return content
					}
				}
			}
		}
		// Fallback to JSON
		jsonBytes, _ := jsoniter.MarshalIndent(v, "", "  ")
		return string(jsonBytes)
	default:
		jsonBytes, _ := jsoniter.MarshalIndent(input, "", "  ")
		return string(jsonBytes)
	}
}

// formatOutputAsString converts output to string format
func formatOutputAsString(output interface{}) string {
	switch v := output.(type) {
	case string:
		return v
	case map[string]interface{}, []interface{}:
		jsonBytes, _ := jsoniter.MarshalIndent(v, "", "  ")
		return string(jsonBytes)
	default:
		if output == nil {
			return "(no output)"
		}
		return fmt.Sprintf("%v", output)
	}
}
