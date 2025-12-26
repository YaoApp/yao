package test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// PathResolver resolves agent information from file paths
type PathResolver struct{}

// NewResolver creates a new path resolver
func NewResolver() Resolver {
	return &PathResolver{}
}

// Resolve resolves the agent from options
// Priority: explicit AgentID > path-based detection (from input file or cwd)
func (r *PathResolver) Resolve(opts *Options) (*AgentInfo, error) {
	// If explicit agent ID is provided, use it
	if opts.AgentID != "" {
		return r.ResolveByID(opts.AgentID)
	}

	// For file mode, resolve from input file path
	if opts.InputMode == InputModeFile {
		if opts.Input == "" {
			return nil, fmt.Errorf("no agent ID or input file specified")
		}
		return r.ResolveFromPath(opts.Input)
	}

	// For message mode, try to resolve from current working directory
	return r.ResolveFromCwd()
}

// ResolveFromCwd resolves the agent from the current working directory
// It looks for package.yao in the current directory or parent directories
func (r *PathResolver) ResolveFromCwd() (*AgentInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	info, err := r.ResolveFromPath(cwd)
	if err != nil {
		return nil, fmt.Errorf("no agent found in current directory. Use -n to specify agent explicitly")
	}
	return info, nil
}

// ResolveFromPath resolves the agent by traversing up from the input file path
// It looks for package.yao in parent directories
// If YAO_ROOT is set, it also considers paths relative to YAO_ROOT
func (r *PathResolver) ResolveFromPath(inputPath string) (*AgentInfo, error) {
	// Get absolute path
	absPath, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Start from the directory containing the input file
	dir := filepath.Dir(absPath)

	// Traverse up to find package.yao
	for {
		packagePath := filepath.Join(dir, "package.yao")
		if _, err := os.Stat(packagePath); err == nil {
			// Found package.yao
			return r.loadAgentFromPath(dir, packagePath)
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, no package.yao found
			break
		}
		dir = parent
	}

	// If YAO_ROOT is set, try resolving relative to it
	yaoRoot := os.Getenv("YAO_ROOT")
	if yaoRoot != "" {
		// Try the input path relative to YAO_ROOT
		relPath := inputPath
		// If inputPath is absolute, try to make it relative
		if filepath.IsAbs(inputPath) {
			// Check if inputPath is under YAO_ROOT
			if rel, err := filepath.Rel(yaoRoot, inputPath); err == nil && !strings.HasPrefix(rel, "..") {
				relPath = rel
			}
		}

		// Traverse up from YAO_ROOT + relPath
		dir = filepath.Join(yaoRoot, filepath.Dir(relPath))
		for {
			packagePath := filepath.Join(dir, "package.yao")
			if _, err := os.Stat(packagePath); err == nil {
				// Found package.yao
				return r.loadAgentFromPath(dir, packagePath)
			}

			// Move to parent directory, but don't go above YAO_ROOT
			parent := filepath.Dir(dir)
			if parent == dir || !strings.HasPrefix(parent, yaoRoot) {
				break
			}
			dir = parent
		}
	}

	return nil, fmt.Errorf("no package.yao found in path hierarchy of %s", inputPath)
}

// ResolveByID resolves an agent by its ID
// This would integrate with the assistant loading system
func (r *PathResolver) ResolveByID(agentID string) (*AgentInfo, error) {
	// This is a placeholder - actual implementation would use assistant.Get()
	// For now, return basic info
	return &AgentInfo{
		ID:   agentID,
		Name: agentID,
	}, nil
}

// loadAgentFromPath loads agent information from a package.yao file
func (r *PathResolver) loadAgentFromPath(agentDir, packagePath string) (*AgentInfo, error) {
	// Read package.yao
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.yao: %w", err)
	}

	// Parse package.yao
	var pkg PackageYao
	if err := jsoniter.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.yao: %w", err)
	}

	// Derive agent ID from directory path
	agentID := deriveAgentID(agentDir)

	return &AgentInfo{
		ID:          agentID,
		Name:        pkg.Name,
		Description: pkg.Description,
		Path:        agentDir,
		Connector:   pkg.Connector,
		Type:        pkg.Type,
	}, nil
}

// PackageYao represents the structure of package.yao
type PackageYao struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Connector   string                 `json:"connector,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Uses        map[string]interface{} `json:"uses,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// deriveAgentID derives an agent ID from the directory path
// e.g., /app/assistants/workers/system/keyword -> workers.system.keyword
func deriveAgentID(dir string) string {
	// Find "assistants" in path and use everything after it
	parts := strings.Split(filepath.ToSlash(dir), "/")

	// Look for "assistants" marker
	startIdx := -1
	for i, part := range parts {
		if part == "assistants" {
			startIdx = i + 1
			break
		}
	}

	if startIdx == -1 || startIdx >= len(parts) {
		// No "assistants" found, use the last directory name
		return filepath.Base(dir)
	}

	// Join remaining parts with dots
	return strings.Join(parts[startIdx:], ".")
}

// GetOutputFormat determines the output format from file extension
func GetOutputFormat(outputPath string) OutputFormat {
	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".json":
		return FormatJSON
	case ".html", ".htm":
		return FormatHTML
	case ".md", ".markdown":
		return FormatMarkdown
	default:
		return FormatJSON // Default to JSON
	}
}

// ValidateOptions validates test options
func ValidateOptions(opts *Options) error {
	if opts.Input == "" {
		return fmt.Errorf("input is required (-i flag)")
	}

	// For file mode, check input file exists
	if opts.InputMode == InputModeFile {
		resolvedPath := ResolvePathWithYaoRoot(opts.Input)
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			return fmt.Errorf("input file not found: %s", opts.Input)
		}
	}

	// Note: For message mode, agent can be resolved from cwd, so no validation here
	// The resolver will return an error if agent cannot be found

	// Validate timeout
	if opts.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	// Validate parallel
	if opts.Parallel < 0 {
		return fmt.Errorf("parallel cannot be negative")
	}

	return nil
}

// DefaultOptions returns options with default values
func DefaultOptions() *Options {
	return &Options{
		Timeout:  120 * time.Second, // 2 minutes default timeout
		Parallel: 1,
		Runs:     1,
		Verbose:  false,
		FailFast: false,
	}
}

// DetectInputMode detects the input mode from the input string
// Returns:
//   - InputModeScript: if input starts with "scripts."
//   - InputModeFile: if input ends with ".jsonl" or is an existing file
//   - InputModeMessage: otherwise (direct message mode)
func DetectInputMode(input string) InputMode {
	// Check for script test prefix
	if strings.HasPrefix(input, "scripts.") {
		return InputModeScript
	}

	// If input ends with .jsonl or .json, treat as file
	if strings.HasSuffix(input, ".jsonl") || strings.HasSuffix(input, ".json") {
		return InputModeFile
	}

	// If input contains path separator, check if file exists
	if strings.Contains(input, string(filepath.Separator)) || strings.Contains(input, "/") {
		if _, err := os.Stat(input); err == nil {
			return InputModeFile
		}
	}

	// Otherwise treat as direct message
	return InputModeMessage
}

// MergeOptions merges user options with defaults
func MergeOptions(opts *Options, defaults *Options) *Options {
	result := *defaults

	if opts.Input != "" {
		result.Input = opts.Input
		result.InputMode = DetectInputMode(opts.Input)
	}
	if opts.OutputFile != "" {
		result.OutputFile = opts.OutputFile
	}
	if opts.AgentID != "" {
		result.AgentID = opts.AgentID
	}
	if opts.Connector != "" {
		result.Connector = opts.Connector
	}
	if opts.UserID != "" {
		result.UserID = opts.UserID
	}
	if opts.TeamID != "" {
		result.TeamID = opts.TeamID
	}
	if opts.Locale != "" {
		result.Locale = opts.Locale
	}
	if opts.Timeout > 0 {
		result.Timeout = opts.Timeout
	}
	if opts.Parallel > 0 {
		result.Parallel = opts.Parallel
	}
	if opts.Runs > 0 {
		result.Runs = opts.Runs
	}
	if opts.ReporterID != "" {
		result.ReporterID = opts.ReporterID
	}
	if opts.ContextFile != "" {
		result.ContextFile = opts.ContextFile
	}
	if opts.Run != "" {
		result.Run = opts.Run
	}
	if opts.Verbose {
		result.Verbose = opts.Verbose
	}
	if opts.FailFast {
		result.FailFast = opts.FailFast
	}

	return &result
}

// GenerateDefaultOutputPath generates the default output path based on input file
// Format: {input_directory}/output-{timestamp}.jsonl
// Timestamp format: YYYYMMDDHHMMSS
func GenerateDefaultOutputPath(inputPath string) string {
	// Resolve input path considering YAO_ROOT
	resolvedPath := ResolvePathWithYaoRoot(inputPath)
	dir := filepath.Dir(resolvedPath)
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("output-%s.jsonl", timestamp)
	return filepath.Join(dir, filename)
}

// ResolveOutputPath resolves the output path based on input mode
// - File mode: generate default path in same directory as input
// - Message mode: return empty string (output to stdout)
// If outputPath is explicitly specified, always use it
func ResolveOutputPath(opts *Options) string {
	if opts.OutputFile != "" {
		return opts.OutputFile
	}

	// For file mode, generate default output path
	if opts.InputMode == InputModeFile {
		return GenerateDefaultOutputPath(opts.Input)
	}

	// For message mode, output to stdout (empty string)
	return ""
}

// CreateTestCaseFromMessage creates a single test case from a direct message
func CreateTestCaseFromMessage(message string) *Case {
	return &Case{
		ID:    "T001",
		Input: message,
	}
}

// ResolvePathWithYaoRoot resolves a file path relative to current directory
// No fallback to YAO_ROOT - paths are always resolved from current working directory
func ResolvePathWithYaoRoot(path string) string {
	// If path is absolute, return as-is
	if filepath.IsAbs(path) {
		return path
	}

	// Resolve relative to current directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}
