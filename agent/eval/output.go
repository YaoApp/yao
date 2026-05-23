package eval

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
)

// OutputWriter handles colored console output for test execution.
// When events is non-nil (JSON stream mode), methods emit structured
// NDJSON events instead of colored text.
type OutputWriter struct {
	verbose bool
	writer  io.Writer
	events  EventWriter
}

// NewOutputWriter creates a new output writer that writes to stdout
func NewOutputWriter(verbose bool) *OutputWriter {
	return &OutputWriter{verbose: verbose, writer: os.Stdout}
}

// NewOutputWriterWithWriter creates a new output writer with a custom writer
func NewOutputWriterWithWriter(verbose bool, w io.Writer, ev EventWriter) *OutputWriter {
	if w == nil {
		w = os.Stdout
	}
	return &OutputWriter{verbose: verbose, writer: w, events: ev}
}

// emitEvent serializes an event as JSON and sends it via EventWriter.
// Returns false if the event could not be sent (stream broken or marshal error).
func (w *OutputWriter) emitEvent(event map[string]interface{}) bool {
	if w.events == nil {
		return false
	}
	data, err := jsoniter.Marshal(event)
	if err != nil {
		return false
	}
	return w.events.WriteEvent(data) == nil
}

// Header prints a header section (skipped in JSON mode)
func (w *OutputWriter) Header(title string) {
	if w.events != nil {
		return
	}
	fmt.Fprintln(w.writer)
	color.New(color.FgCyan, color.Bold).Fprintln(w.writer, "═══════════════════════════════════════════════════════════════")
	color.New(color.FgCyan, color.Bold).Fprintf(w.writer, "  %s\n", title)
	color.New(color.FgCyan, color.Bold).Fprintln(w.writer, "═══════════════════════════════════════════════════════════════")
}

// SubHeader prints a sub-header (skipped in JSON mode)
func (w *OutputWriter) SubHeader(title string) {
	if w.events != nil {
		return
	}
	fmt.Fprintln(w.writer)
	color.New(color.FgWhite, color.Bold).Fprintln(w.writer, "───────────────────────────────────────────────────────────────")
	color.New(color.FgWhite, color.Bold).Fprintf(w.writer, "  %s\n", title)
	color.New(color.FgWhite, color.Bold).Fprintln(w.writer, "───────────────────────────────────────────────────────────────")
}

// Info prints an info message
func (w *OutputWriter) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "log", "level": "info", "message": msg})
		return
	}
	color.New(color.FgBlue).Fprint(w.writer, "ℹ ")
	fmt.Fprintln(w.writer, msg)
}

// Error prints an error message
func (w *OutputWriter) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "log", "level": "error", "message": msg})
		return
	}
	color.New(color.FgRed).Fprint(w.writer, "✗ ")
	fmt.Fprintln(w.writer, msg)
}

// Warning prints a warning message
func (w *OutputWriter) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "log", "level": "warning", "message": msg})
		return
	}
	color.New(color.FgYellow).Fprint(w.writer, "⚠ ")
	fmt.Fprintln(w.writer, msg)
}

// Verbose prints a verbose message (only if verbose mode is enabled)
func (w *OutputWriter) Verbose(format string, args ...interface{}) {
	if !w.verbose {
		return
	}
	msg := fmt.Sprintf(format, args...)
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "log", "level": "verbose", "message": msg})
		return
	}
	color.New(color.FgHiBlack).Fprint(w.writer, "  │ ")
	fmt.Fprintln(w.writer, msg)
}

// TestStart prints test case start
func (w *OutputWriter) TestStart(id string, input string, runNum int) {
	if w.events != nil {
		ev := map[string]interface{}{"type": "test_start", "id": id, "input": truncateString(input, 200)}
		if runNum > 1 {
			ev["run"] = runNum
		}
		w.emitEvent(ev)
		return
	}
	inputPreview := truncateString(input, 50)
	if runNum > 1 {
		color.New(color.FgWhite).Fprintf(w.writer, "► [%s] Run %d: ", id, runNum)
	} else {
		color.New(color.FgWhite).Fprintf(w.writer, "► [%s] ", id)
	}
	color.New(color.FgHiBlack).Fprintf(w.writer, "%s", inputPreview)
	fmt.Fprint(w.writer, " ")
}

// TestResult prints test case result
func (w *OutputWriter) TestResult(id string, status Status, duration time.Duration) {
	if w.events != nil {
		w.emitEvent(map[string]interface{}{
			"type": "test_result", "id": id, "status": string(status),
			"duration_ms": duration.Milliseconds(),
		})
		return
	}
	switch status {
	case StatusPassed:
		color.New(color.FgGreen, color.Bold).Fprint(w.writer, "PASSED")
	case StatusFailed:
		color.New(color.FgRed, color.Bold).Fprint(w.writer, "FAILED")
	case StatusSkipped:
		color.New(color.FgYellow).Fprint(w.writer, "SKIPPED")
	case StatusError:
		color.New(color.FgRed, color.Bold).Fprint(w.writer, "ERROR")
	case StatusTimeout:
		color.New(color.FgRed).Fprint(w.writer, "TIMEOUT")
	}
	color.New(color.FgHiBlack).Fprintf(w.writer, " (%s)\n", formatDuration(duration))
}

// TestError prints test error details
func (w *OutputWriter) TestError(id string, err string) {
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "test_error", "id": id, "error": err})
		return
	}
	color.New(color.FgRed).Fprintf(w.writer, "  └─ %s\n", err)
}

// TestOutput prints test output (verbose mode)
func (w *OutputWriter) TestOutput(output string) {
	if output == "" {
		return
	}
	if w.events != nil {
		if w.verbose {
			w.emitEvent(map[string]interface{}{"type": "test_output", "output": output})
		}
		return
	}
	if w.verbose {
		outputPreview := truncateString(output, 100)
		color.New(color.FgHiBlack).Fprintf(w.writer, "  └─ Output: %s\n", outputPreview)
	}
}

// Summary prints the test summary
func (w *OutputWriter) Summary(summary *Summary, duration time.Duration) {
	if w.events != nil {
		ev := map[string]interface{}{
			"type":        "summary",
			"agent_id":    summary.AgentID,
			"total":       summary.Total,
			"passed":      summary.Passed,
			"failed":      summary.Failed,
			"skipped":     summary.Skipped,
			"errors":      summary.Errors,
			"timeouts":    summary.Timeouts,
			"duration_ms": duration.Milliseconds(),
		}
		if summary.Connector != "" {
			ev["connector"] = summary.Connector
		}
		if summary.RunsPerCase > 1 {
			ev["runs_per_case"] = summary.RunsPerCase
			ev["total_runs"] = summary.TotalRuns
			ev["stable_cases"] = summary.StableCases
			ev["unstable_cases"] = summary.UnstableCases
		}
		w.emitEvent(ev)
		return
	}
	w.SubHeader("Summary")

	// Agent info
	color.New(color.FgWhite).Fprint(w.writer, "  Agent:     ")
	color.New(color.FgCyan).Fprintf(w.writer, "%s\n", summary.AgentID)

	if summary.Connector != "" {
		color.New(color.FgWhite).Fprint(w.writer, "  Connector: ")
		color.New(color.FgCyan).Fprintf(w.writer, "%s\n", summary.Connector)
	}

	// Results
	color.New(color.FgWhite).Fprint(w.writer, "  Total:     ")
	fmt.Fprintf(w.writer, "%d\n", summary.Total)

	color.New(color.FgWhite).Fprint(w.writer, "  Passed:    ")
	if summary.Passed > 0 {
		color.New(color.FgGreen).Fprintf(w.writer, "%d\n", summary.Passed)
	} else {
		fmt.Fprintf(w.writer, "%d\n", summary.Passed)
	}

	color.New(color.FgWhite).Fprint(w.writer, "  Failed:    ")
	if summary.Failed > 0 {
		color.New(color.FgRed).Fprintf(w.writer, "%d\n", summary.Failed)
	} else {
		fmt.Fprintf(w.writer, "%d\n", summary.Failed)
	}

	if summary.Skipped > 0 {
		color.New(color.FgWhite).Fprint(w.writer, "  Skipped:   ")
		color.New(color.FgYellow).Fprintf(w.writer, "%d\n", summary.Skipped)
	}

	if summary.Errors > 0 {
		color.New(color.FgWhite).Fprint(w.writer, "  Errors:    ")
		color.New(color.FgRed).Fprintf(w.writer, "%d\n", summary.Errors)
	}

	if summary.Timeouts > 0 {
		color.New(color.FgWhite).Fprint(w.writer, "  Timeouts:  ")
		color.New(color.FgRed).Fprintf(w.writer, "%d\n", summary.Timeouts)
	}

	// Pass rate
	passRate := float64(0)
	if summary.Total > 0 {
		passRate = float64(summary.Passed) / float64(summary.Total) * 100
	}
	color.New(color.FgWhite).Fprint(w.writer, "  Pass Rate: ")
	if passRate == 100 {
		color.New(color.FgGreen, color.Bold).Fprintf(w.writer, "%.1f%%\n", passRate)
	} else if passRate >= 80 {
		color.New(color.FgYellow).Fprintf(w.writer, "%.1f%%\n", passRate)
	} else {
		color.New(color.FgRed).Fprintf(w.writer, "%.1f%%\n", passRate)
	}

	// Duration
	color.New(color.FgWhite).Fprint(w.writer, "  Duration:  ")
	fmt.Fprintf(w.writer, "%s\n", formatDuration(duration))

	// Stability info (if runs > 1)
	if summary.RunsPerCase > 1 {
		fmt.Fprintln(w.writer)
		color.New(color.FgWhite, color.Bold).Fprintln(w.writer, "  Stability Analysis:")
		color.New(color.FgWhite).Fprintf(w.writer, "    Runs/Case:     %d\n", summary.RunsPerCase)
		color.New(color.FgWhite).Fprintf(w.writer, "    Total Runs:    %d\n", summary.TotalRuns)
		color.New(color.FgWhite).Fprint(w.writer, "    Stable Cases:  ")
		if summary.StableCases == summary.Total {
			color.New(color.FgGreen).Fprintf(w.writer, "%d\n", summary.StableCases)
		} else {
			color.New(color.FgYellow).Fprintf(w.writer, "%d\n", summary.StableCases)
		}
		color.New(color.FgWhite).Fprint(w.writer, "    Unstable:      ")
		if summary.UnstableCases > 0 {
			color.New(color.FgRed).Fprintf(w.writer, "%d\n", summary.UnstableCases)
		} else {
			fmt.Fprintf(w.writer, "%d\n", summary.UnstableCases)
		}
	}
}

// OutputFile prints the output file path
func (w *OutputWriter) OutputFile(path string) {
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "output_file", "path": path})
		return
	}
	fmt.Fprintln(w.writer)
	color.New(color.FgWhite).Fprint(w.writer, "  Output: ")
	color.New(color.FgCyan).Fprintf(w.writer, "%s\n", path)
}

// FinalResult prints the final result banner (skipped in JSON mode, Report is in done chunk)
func (w *OutputWriter) FinalResult(passed bool) {
	if w.events != nil {
		return
	}
	fmt.Fprintln(w.writer)
	if passed {
		color.New(color.FgGreen, color.Bold).Fprintln(w.writer, "═══════════════════════════════════════════════════════════════")
		color.New(color.FgGreen, color.Bold).Fprintln(w.writer, "  ✨ ALL TESTS PASSED ✨")
		color.New(color.FgGreen, color.Bold).Fprintln(w.writer, "═══════════════════════════════════════════════════════════════")
	} else {
		color.New(color.FgRed, color.Bold).Fprintln(w.writer, "═══════════════════════════════════════════════════════════════")
		color.New(color.FgRed, color.Bold).Fprintln(w.writer, "  ❌ TESTS FAILED")
		color.New(color.FgRed, color.Bold).Fprintln(w.writer, "═══════════════════════════════════════════════════════════════")
	}
	fmt.Fprintln(w.writer)
}

// DirectOutput prints the agent output directly (for development mode)
func (w *OutputWriter) DirectOutput(output interface{}) {
	if output == nil {
		return
	}
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "direct_output", "output": output})
		return
	}

	switch v := output.(type) {
	case string:
		fmt.Fprintln(w.writer, v)
	case map[string]interface{}, []interface{}:
		jsonBytes, err := jsoniter.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Fprintf(w.writer, "%v\n", output)
		} else {
			fmt.Fprintln(w.writer, string(jsonBytes))
		}
	default:
		jsonBytes, err := jsoniter.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(w.writer, "%v\n", output)
		} else {
			fmt.Fprintln(w.writer, string(jsonBytes))
		}
	}
}

// DirectOutputJSON outputs a complete JSON object with output, trace and duration.
func (w *OutputWriter) DirectOutputJSON(output interface{}, trace *Trace, duration time.Duration) {
	payload := map[string]interface{}{
		"type":        "direct_output",
		"output":      output,
		"duration_ms": duration.Milliseconds(),
	}
	if trace != nil {
		payload["trace"] = trace
	}
	if w.events != nil {
		w.emitEvent(payload)
		return
	}
	jsonBytes, err := jsoniter.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(w.writer, "%v\n", output)
		return
	}
	fmt.Fprintln(w.writer, string(jsonBytes))
}

// DirectTrace prints a human-readable summary of tool calls from a Trace.
func (w *OutputWriter) DirectTrace(trace *Trace) {
	if trace == nil || len(trace.ToolCalls) == 0 {
		return
	}
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "direct_trace", "trace": trace})
		return
	}

	fmt.Fprintln(w.writer)
	color.New(color.FgHiBlack).Fprintln(w.writer, "--- Tool Calls ---")
	for _, tc := range trace.ToolCalls {
		prefix := tc.Tool
		if tc.Server != "" {
			prefix = tc.Server + "/" + tc.Tool
		}

		status := "OK"
		if tc.Error != "" {
			status = "ERR: " + truncateString(tc.Error, 60)
		}

		argsStr := ""
		if tc.Arguments != nil {
			if b, err := jsoniter.Marshal(tc.Arguments); err == nil {
				argsStr = truncateString(string(b), 80)
			}
		}

		if argsStr != "" {
			color.New(color.FgHiBlack).Fprintf(w.writer, "  %s → %s (args: %s)\n", prefix, status, argsStr)
		} else {
			color.New(color.FgHiBlack).Fprintf(w.writer, "  %s → %s\n", prefix, status)
		}
	}
}

// ScriptOutputJSON outputs the complete script test report as JSON.
func (w *OutputWriter) ScriptOutputJSON(report *ScriptTestReport) {
	jsonBytes, err := jsoniter.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(w.writer, "{\"error\": %q}\n", err.Error())
		return
	}
	fmt.Fprintln(w.writer, string(jsonBytes))
}

// ScriptTestSummary prints the script test summary
func (w *OutputWriter) ScriptTestSummary(summary *ScriptTestSummary, duration time.Duration) {
	if w.events != nil {
		w.emitEvent(map[string]interface{}{
			"type":        "script_summary",
			"total":       summary.Total,
			"passed":      summary.Passed,
			"failed":      summary.Failed,
			"skipped":     summary.Skipped,
			"duration_ms": duration.Milliseconds(),
		})
		return
	}
	w.SubHeader("Summary")

	color.New(color.FgWhite).Fprint(w.writer, "  Total:     ")
	fmt.Fprintf(w.writer, "%d\n", summary.Total)

	color.New(color.FgWhite).Fprint(w.writer, "  Passed:    ")
	if summary.Passed > 0 {
		color.New(color.FgGreen).Fprintf(w.writer, "%d\n", summary.Passed)
	} else {
		fmt.Fprintf(w.writer, "%d\n", summary.Passed)
	}

	color.New(color.FgWhite).Fprint(w.writer, "  Failed:    ")
	if summary.Failed > 0 {
		color.New(color.FgRed).Fprintf(w.writer, "%d\n", summary.Failed)
	} else {
		fmt.Fprintf(w.writer, "%d\n", summary.Failed)
	}

	if summary.Skipped > 0 {
		color.New(color.FgWhite).Fprint(w.writer, "  Skipped:   ")
		color.New(color.FgYellow).Fprintf(w.writer, "%d\n", summary.Skipped)
	}

	passRate := float64(0)
	if summary.Total > 0 {
		passRate = float64(summary.Passed) / float64(summary.Total) * 100
	}
	color.New(color.FgWhite).Fprint(w.writer, "  Pass Rate: ")
	if passRate == 100 {
		color.New(color.FgGreen, color.Bold).Fprintf(w.writer, "%.1f%%\n", passRate)
	} else if passRate >= 80 {
		color.New(color.FgYellow).Fprintf(w.writer, "%.1f%%\n", passRate)
	} else {
		color.New(color.FgRed).Fprintf(w.writer, "%.1f%%\n", passRate)
	}

	color.New(color.FgWhite).Fprint(w.writer, "  Duration:  ")
	fmt.Fprintf(w.writer, "%s\n", formatDuration(duration))
}

// DynamicTestStart outputs the start of a dynamic test
func (w *OutputWriter) DynamicTestStart(id string, checkpointCount int) {
	if w.events != nil {
		w.emitEvent(map[string]interface{}{"type": "dynamic_start", "id": id, "checkpoints": checkpointCount})
		return
	}
	color.New(color.FgWhite).Fprintf(w.writer, "► [%s] ", id)
	color.New(color.FgCyan).Fprintf(w.writer, "(dynamic, %d checkpoints)\n", checkpointCount)
}

// DynamicTestResult outputs the result of a dynamic test
func (w *OutputWriter) DynamicTestResult(status Status, turns int, checkpoints int, duration time.Duration) {
	if w.events != nil {
		w.emitEvent(map[string]interface{}{
			"type": "dynamic_result", "status": string(status),
			"turns": turns, "checkpoints": checkpoints,
			"duration_ms": duration.Milliseconds(),
		})
		return
	}
	color.New(color.FgHiBlack).Fprint(w.writer, "  └─ ")

	switch status {
	case StatusPassed:
		color.New(color.FgGreen).Fprint(w.writer, "PASSED")
	case StatusFailed:
		color.New(color.FgRed).Fprint(w.writer, "FAILED")
	case StatusError:
		color.New(color.FgRed).Fprint(w.writer, "ERROR")
	case StatusTimeout:
		color.New(color.FgRed).Fprint(w.writer, "TIMEOUT")
	}

	color.New(color.FgHiBlack).Fprintf(w.writer, " (%d turns, %d checkpoints, %s)\n", turns, checkpoints, formatDuration(duration))
}

// StabilityResult prints stability analysis result for a test case
func (w *OutputWriter) StabilityResult(sr *StabilityResult) {
	if w.events != nil {
		w.emitEvent(map[string]interface{}{
			"type": "stability", "id": sr.ID,
			"runs": sr.Runs, "passed": sr.Passed,
			"pass_rate": sr.PassRate, "stability_class": string(sr.StabilityClass),
			"avg_duration_ms": sr.AvgDurationMs,
		})
		return
	}
	color.New(color.FgWhite).Fprintf(w.writer, "  [%s] ", sr.ID)

	if sr.PassRate == 100 {
		color.New(color.FgGreen).Fprintf(w.writer, "%.0f%%", sr.PassRate)
	} else if sr.PassRate >= 80 {
		color.New(color.FgYellow).Fprintf(w.writer, "%.0f%%", sr.PassRate)
	} else {
		color.New(color.FgRed).Fprintf(w.writer, "%.0f%%", sr.PassRate)
	}

	color.New(color.FgHiBlack).Fprintf(w.writer, " (%d/%d) ", sr.Passed, sr.Runs)

	switch sr.StabilityClass {
	case StabilityStable:
		color.New(color.FgGreen).Fprint(w.writer, "Stable")
	case StabilityMostlyStable:
		color.New(color.FgYellow).Fprint(w.writer, "Mostly Stable")
	case StabilityUnstable:
		color.New(color.FgRed).Fprint(w.writer, "Unstable")
	case StabilityHighlyUnstable:
		color.New(color.FgRed, color.Bold).Fprint(w.writer, "Highly Unstable")
	}

	color.New(color.FgHiBlack).Fprintf(w.writer, " avg:%.0fms\n", sr.AvgDurationMs)
}

// Helper functions

func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.Join(strings.Fields(s), " ")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// EmitTypedEvent serializes a strong-typed event struct (StartEvent, ResultEvent,
// SummaryEvent) and sends it via the EventWriter. For use by cmd/agent layer.
func (w *OutputWriter) EmitTypedEvent(event interface{}) bool {
	if w.events == nil {
		return false
	}
	data, err := jsoniter.Marshal(event)
	if err != nil {
		return false
	}
	return w.events.WriteEvent(data) == nil
}

// DryRunCase prints a single test case in dry-run mode (list without executing).
func (w *OutputWriter) DryRunCase(id string, input string) {
	if w.events != nil {
		w.emitEvent(map[string]interface{}{
			"type": "dry_run_case", "id": id, "input": truncateString(input, 200),
		})
		return
	}
	color.New(color.FgWhite).Fprintf(w.writer, "  ○ [%s] ", id)
	color.New(color.FgHiBlack).Fprintf(w.writer, "%s\n", truncateString(input, 80))
}
