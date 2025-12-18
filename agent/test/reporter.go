package test

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/agent/context"
)

// JSONLReporter generates JSONL format reports (default)
type JSONLReporter struct{}

// NewJSONLReporter creates a new JSONL reporter
func NewJSONLReporter() *JSONLReporter {
	return &JSONLReporter{}
}

// Generate generates a JSONL report (writes to stdout or file)
func (r *JSONLReporter) Generate(report *Report) error {
	return nil // JSONL is written during test execution
}

// Write writes the report in JSONL format
func (r *JSONLReporter) Write(report *Report, w io.Writer) error {
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	// Start event
	startEvent := map[string]interface{}{
		"type":        "start",
		"timestamp":   report.Metadata.StartedAt.Format(time.RFC3339),
		"agent_id":    report.Summary.AgentID,
		"total_cases": report.Summary.Total,
	}
	if err := writeJSONLineToWriter(writer, startEvent); err != nil {
		return err
	}

	// Result events
	if report.Results != nil {
		for _, result := range report.Results {
			resultEvent := map[string]interface{}{
				"type":        "result",
				"id":          result.ID,
				"status":      result.Status,
				"duration_ms": result.DurationMs,
			}
			if result.Output != nil {
				resultEvent["output"] = result.Output
			}
			if result.Error != "" {
				resultEvent["error"] = result.Error
			}
			if err := writeJSONLineToWriter(writer, resultEvent); err != nil {
				return err
			}
		}
	}

	// Stability results
	if report.StabilityResults != nil {
		for _, sr := range report.StabilityResults {
			stabilityEvent := map[string]interface{}{
				"type":            "stability",
				"id":              sr.ID,
				"runs":            sr.Runs,
				"passed":          sr.Passed,
				"failed":          sr.Failed,
				"pass_rate":       sr.PassRate,
				"stable":          sr.Stable,
				"stability_class": sr.StabilityClass,
				"avg_duration_ms": sr.AvgDurationMs,
			}
			if err := writeJSONLineToWriter(writer, stabilityEvent); err != nil {
				return err
			}
		}
	}

	// Summary event
	summaryEvent := map[string]interface{}{
		"type":        "summary",
		"total":       report.Summary.Total,
		"passed":      report.Summary.Passed,
		"failed":      report.Summary.Failed,
		"skipped":     report.Summary.Skipped,
		"errors":      report.Summary.Errors,
		"timeouts":    report.Summary.Timeouts,
		"duration_ms": report.Summary.DurationMs,
	}
	if report.Summary.RunsPerCase > 1 {
		summaryEvent["runs_per_case"] = report.Summary.RunsPerCase
		summaryEvent["total_runs"] = report.Summary.TotalRuns
		summaryEvent["overall_pass_rate"] = report.Summary.OverallPassRate
		summaryEvent["stable_cases"] = report.Summary.StableCases
		summaryEvent["unstable_cases"] = report.Summary.UnstableCases
	}
	return writeJSONLineToWriter(writer, summaryEvent)
}

// writeJSONLineToWriter writes a JSON line to the writer
func writeJSONLineToWriter(writer *bufio.Writer, data interface{}) error {
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

// JSONReporter generates full JSON format reports
type JSONReporter struct{}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{}
}

// Generate generates a JSON report
func (r *JSONReporter) Generate(report *Report) error {
	return nil
}

// Write writes the report in JSON format
func (r *JSONReporter) Write(report *Report, w io.Writer) error {
	encoder := jsoniter.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// MarkdownReporter generates Markdown format reports
type MarkdownReporter struct{}

// NewMarkdownReporter creates a new Markdown reporter
func NewMarkdownReporter() *MarkdownReporter {
	return &MarkdownReporter{}
}

// Generate generates a Markdown report
func (r *MarkdownReporter) Generate(report *Report) error {
	return nil
}

// Write writes the report in Markdown format
func (r *MarkdownReporter) Write(report *Report, w io.Writer) error {
	var sb strings.Builder

	// Header
	sb.WriteString("# Agent Test Report\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("| ------ | ----- |\n")
	sb.WriteString(fmt.Sprintf("| Agent | %s |\n", report.Summary.AgentID))
	if report.Summary.Connector != "" {
		sb.WriteString(fmt.Sprintf("| Connector | %s |\n", report.Summary.Connector))
	}
	sb.WriteString(fmt.Sprintf("| Total | %d |\n", report.Summary.Total))
	sb.WriteString(fmt.Sprintf("| Passed | %d |\n", report.Summary.Passed))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", report.Summary.Failed))
	if report.Summary.Skipped > 0 {
		sb.WriteString(fmt.Sprintf("| Skipped | %d |\n", report.Summary.Skipped))
	}
	if report.Summary.Errors > 0 {
		sb.WriteString(fmt.Sprintf("| Errors | %d |\n", report.Summary.Errors))
	}
	if report.Summary.Timeouts > 0 {
		sb.WriteString(fmt.Sprintf("| Timeouts | %d |\n", report.Summary.Timeouts))
	}

	passRate := float64(0)
	if report.Summary.Total > 0 {
		passRate = float64(report.Summary.Passed) / float64(report.Summary.Total) * 100
	}
	sb.WriteString(fmt.Sprintf("| Pass Rate | %.1f%% |\n", passRate))
	sb.WriteString(fmt.Sprintf("| Duration | %dms |\n", report.Summary.DurationMs))
	sb.WriteString("\n")

	// Environment
	if report.Environment != nil {
		sb.WriteString("## Environment\n\n")
		sb.WriteString("| Setting | Value |\n")
		sb.WriteString("| ------- | ----- |\n")
		sb.WriteString(fmt.Sprintf("| User | %s |\n", report.Environment.UserID))
		sb.WriteString(fmt.Sprintf("| Team | %s |\n", report.Environment.TeamID))
		sb.WriteString(fmt.Sprintf("| Locale | %s |\n", report.Environment.Locale))
		sb.WriteString("\n")
	}

	// Results
	sb.WriteString("## Results\n\n")

	if report.Results != nil {
		for _, result := range report.Results {
			statusIcon := "✅"
			switch result.Status {
			case StatusFailed:
				statusIcon = "❌"
			case StatusError:
				statusIcon = "💥"
			case StatusTimeout:
				statusIcon = "⏱️"
			case StatusSkipped:
				statusIcon = "⏭️"
			}

			sb.WriteString(fmt.Sprintf("### %s %s - %s (%dms)\n\n", statusIcon, result.ID, result.Status, result.DurationMs))

			if result.Error != "" {
				sb.WriteString(fmt.Sprintf("**Error:** %s\n\n", result.Error))
			}
		}
	}

	// Stability results
	if report.StabilityResults != nil {
		sb.WriteString("## Stability Analysis\n\n")
		sb.WriteString("| ID | Pass Rate | Runs | Status | Avg Duration |\n")
		sb.WriteString("| -- | --------- | ---- | ------ | ------------ |\n")

		for _, sr := range report.StabilityResults {
			status := string(sr.StabilityClass)
			sb.WriteString(fmt.Sprintf("| %s | %.0f%% | %d/%d | %s | %.0fms |\n",
				sr.ID, sr.PassRate, sr.Passed, sr.Runs, status, sr.AvgDurationMs))
		}
		sb.WriteString("\n")
	}

	// Metadata
	sb.WriteString("## Metadata\n\n")
	sb.WriteString(fmt.Sprintf("- **Started:** %s\n", report.Metadata.StartedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Completed:** %s\n", report.Metadata.CompletedAt.Format(time.RFC3339)))
	if report.Metadata.InputFile != "" {
		sb.WriteString(fmt.Sprintf("- **Input File:** %s\n", report.Metadata.InputFile))
	}
	if report.Metadata.OutputFile != "" {
		sb.WriteString(fmt.Sprintf("- **Output File:** %s\n", report.Metadata.OutputFile))
	}

	_, err := w.Write([]byte(sb.String()))
	return err
}

// HTMLReporter generates HTML format reports
type HTMLReporter struct{}

// NewHTMLReporter creates a new HTML reporter
func NewHTMLReporter() *HTMLReporter {
	return &HTMLReporter{}
}

// Generate generates an HTML report
func (r *HTMLReporter) Generate(report *Report) error {
	return nil
}

// Write writes the report in HTML format
func (r *HTMLReporter) Write(report *Report, w io.Writer) error {
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Calculate pass rate
	passRate := float64(0)
	if report.Summary.Total > 0 {
		passRate = float64(report.Summary.Passed) / float64(report.Summary.Total) * 100
	}

	data := map[string]interface{}{
		"Report":   report,
		"PassRate": passRate,
	}

	return tmpl.Execute(w, data)
}

// HTML template for reports
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Agent Test Report - {{.Report.Summary.AgentID}}</title>
    <style>
        :root {
            --bg-primary: #0d1117;
            --bg-secondary: #161b22;
            --bg-tertiary: #21262d;
            --text-primary: #c9d1d9;
            --text-secondary: #8b949e;
            --accent-green: #3fb950;
            --accent-red: #f85149;
            --accent-yellow: #d29922;
            --accent-blue: #58a6ff;
            --border-color: #30363d;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
            padding: 2rem;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        
        h1 {
            font-size: 2rem;
            margin-bottom: 0.5rem;
            color: var(--text-primary);
        }
        
        h2 {
            font-size: 1.25rem;
            margin: 2rem 0 1rem;
            color: var(--text-primary);
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 0.5rem;
        }
        
        .subtitle {
            color: var(--text-secondary);
            font-size: 0.9rem;
            margin-bottom: 2rem;
        }
        
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        
        .summary-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            padding: 1rem;
            text-align: center;
        }
        
        .summary-card .value {
            font-size: 2rem;
            font-weight: 600;
        }
        
        .summary-card .label {
            color: var(--text-secondary);
            font-size: 0.85rem;
            margin-top: 0.25rem;
        }
        
        .summary-card.passed .value { color: var(--accent-green); }
        .summary-card.failed .value { color: var(--accent-red); }
        .summary-card.rate .value { color: var(--accent-blue); }
        
        .results-table {
            width: 100%;
            border-collapse: collapse;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            overflow: hidden;
        }
        
        .results-table th,
        .results-table td {
            padding: 0.75rem 1rem;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
        }
        
        .results-table th {
            background: var(--bg-tertiary);
            font-weight: 600;
            color: var(--text-primary);
        }
        
        .results-table tr:last-child td {
            border-bottom: none;
        }
        
        .status {
            display: inline-block;
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
            font-size: 0.8rem;
            font-weight: 500;
        }
        
        .status.passed { background: rgba(63, 185, 80, 0.2); color: var(--accent-green); }
        .status.failed { background: rgba(248, 81, 73, 0.2); color: var(--accent-red); }
        .status.error { background: rgba(248, 81, 73, 0.2); color: var(--accent-red); }
        .status.timeout { background: rgba(210, 153, 34, 0.2); color: var(--accent-yellow); }
        .status.skipped { background: rgba(139, 148, 158, 0.2); color: var(--text-secondary); }
        
        .error-msg {
            color: var(--accent-red);
            font-size: 0.85rem;
            margin-top: 0.25rem;
        }
        
        .metadata {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            padding: 1rem;
            font-size: 0.85rem;
            color: var(--text-secondary);
        }
        
        .metadata dt {
            font-weight: 600;
            color: var(--text-primary);
            display: inline;
        }
        
        .metadata dd {
            display: inline;
            margin: 0 1rem 0 0.5rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Agent Test Report</h1>
        <p class="subtitle">{{.Report.Summary.AgentID}} {{if .Report.Summary.Connector}}• {{.Report.Summary.Connector}}{{end}}</p>
        
        <div class="summary-grid">
            <div class="summary-card">
                <div class="value">{{.Report.Summary.Total}}</div>
                <div class="label">Total Tests</div>
            </div>
            <div class="summary-card passed">
                <div class="value">{{.Report.Summary.Passed}}</div>
                <div class="label">Passed</div>
            </div>
            <div class="summary-card failed">
                <div class="value">{{.Report.Summary.Failed}}</div>
                <div class="label">Failed</div>
            </div>
            <div class="summary-card rate">
                <div class="value">{{printf "%.1f" .PassRate}}%</div>
                <div class="label">Pass Rate</div>
            </div>
            <div class="summary-card">
                <div class="value">{{.Report.Summary.DurationMs}}ms</div>
                <div class="label">Duration</div>
            </div>
        </div>
        
        <h2>Test Results</h2>
        <table class="results-table">
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Status</th>
                    <th>Duration</th>
                    <th>Details</th>
                </tr>
            </thead>
            <tbody>
                {{range .Report.Results}}
                <tr>
                    <td>{{.ID}}</td>
                    <td><span class="status {{.Status}}">{{.Status}}</span></td>
                    <td>{{.DurationMs}}ms</td>
                    <td>
                        {{if .Error}}<div class="error-msg">{{.Error}}</div>{{end}}
                    </td>
                </tr>
                {{end}}
                {{range .Report.StabilityResults}}
                <tr>
                    <td>{{.ID}}</td>
                    <td><span class="status {{if .Stable}}passed{{else}}failed{{end}}">{{.StabilityClass}}</span></td>
                    <td>{{printf "%.0f" .AvgDurationMs}}ms avg</td>
                    <td>{{.Passed}}/{{.Runs}} passed ({{printf "%.0f" .PassRate}}%)</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        
        <h2>Metadata</h2>
        <dl class="metadata">
            <dt>Started:</dt><dd>{{.Report.Metadata.StartedAt}}</dd>
            <dt>Completed:</dt><dd>{{.Report.Metadata.CompletedAt}}</dd>
            {{if .Report.Metadata.InputFile}}<dt>Input:</dt><dd>{{.Report.Metadata.InputFile}}</dd>{{end}}
            {{if .Report.Metadata.OutputFile}}<dt>Output:</dt><dd>{{.Report.Metadata.OutputFile}}</dd>{{end}}
        </dl>
    </div>
</body>
</html>`

// AgentReporter uses a custom agent to generate reports
type AgentReporter struct {
	agentID string
	format  string
	verbose bool
	ctx     *context.Context // Test context for agent call
}

// NewAgentReporter creates a new agent-based reporter
func NewAgentReporter(agentID, format string, verbose bool) *AgentReporter {
	return &AgentReporter{
		agentID: agentID,
		format:  format,
		verbose: verbose,
	}
}

// SetContext sets the context for agent calls
func (r *AgentReporter) SetContext(ctx *context.Context) {
	r.ctx = ctx
}

// Generate generates a report using the agent
func (r *AgentReporter) Generate(report *Report) error {
	return nil
}

// Write writes the report using the agent
func (r *AgentReporter) Write(report *Report, w io.Writer) error {
	// Check if AgentGetterFunc is initialized
	if caller.AgentGetterFunc == nil {
		return fmt.Errorf("AgentGetterFunc not initialized, cannot call reporter agent")
	}

	// Get the reporter agent
	agent, err := caller.AgentGetterFunc(r.agentID)
	if err != nil {
		return fmt.Errorf("failed to get reporter agent %s: %w", r.agentID, err)
	}

	// Build input for the reporter agent
	input := &ReporterInput{
		Report: report,
		Format: r.format,
		Options: &ReporterOptions{
			Verbose:        r.verbose,
			IncludeOutputs: r.verbose,
			IncludeInputs:  r.verbose,
		},
	}

	// Convert input to JSON for the agent
	inputJSON, err := jsoniter.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal reporter input: %w", err)
	}

	// Create message for the agent
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: string(inputJSON),
		},
	}

	// Create context if not provided
	ctx := r.ctx
	if ctx == nil {
		// Create a minimal context for the reporter agent call
		ctx = NewTestContext("reporter", r.agentID, NewEnvironment("", ""))
		defer ctx.Release()
	}

	// Call the agent with skip options (no history, no output)
	options := &context.Options{
		Skip: &context.Skip{
			History: true,
			Output:  true,
		},
	}

	response, err := agent.Stream(ctx, messages, options)
	if err != nil {
		return fmt.Errorf("reporter agent call failed: %w", err)
	}

	// Extract content from response
	content, err := r.extractContent(response)
	if err != nil {
		return fmt.Errorf("failed to extract report content: %w", err)
	}

	// Write the content to output
	_, err = w.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

// extractContent extracts the report content from the agent's *context.Response
// Now that agent.Stream() returns *context.Response directly,
// we can access fields without type assertions.
func (r *AgentReporter) extractContent(response *context.Response) (string, error) {
	if response == nil {
		return "", fmt.Errorf("agent returned nil response")
	}

	// Priority 1: Check Next field (custom hook data)
	if response.Next != nil {
		return r.contentToString(response.Next)
	}

	// Priority 2: Extract from completion content
	if response.Completion != nil && response.Completion.Content != nil {
		return r.contentToString(response.Completion.Content)
	}

	return "", fmt.Errorf("no content in response")
}

// contentToString converts various content types to string
func (r *AgentReporter) contentToString(content interface{}) (string, error) {
	switch v := content.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		jsonBytes, err := jsoniter.Marshal(content)
		if err != nil {
			return fmt.Sprintf("%v", content), nil
		}
		return string(jsonBytes), nil
	}
}

// GetReporter returns a reporter based on output format
func GetReporter(format OutputFormat) Reporter {
	switch format {
	case FormatJSON:
		return NewJSONReporter()
	case FormatHTML:
		return NewHTMLReporter()
	case FormatMarkdown:
		return NewMarkdownReporter()
	default:
		return NewJSONLReporter()
	}
}

// GetReporterFromPath returns a reporter based on file extension
func GetReporterFromPath(outputPath string) Reporter {
	format := GetOutputFormat(outputPath)
	return GetReporter(format)
}

// GetReporterWithAgent returns an agent-based reporter if agentID is specified,
// otherwise returns a built-in reporter based on output format
func GetReporterWithAgent(agentID, outputPath string, verbose bool) Reporter {
	if agentID != "" {
		format := GetOutputFormat(outputPath)
		return NewAgentReporter(agentID, string(format), verbose)
	}
	return GetReporterFromPath(outputPath)
}
