package test

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
)

// OutputWriter handles colored console output for test execution
type OutputWriter struct {
	verbose bool
}

// NewOutputWriter creates a new output writer
func NewOutputWriter(verbose bool) *OutputWriter {
	return &OutputWriter{verbose: verbose}
}

// Header prints a header section
func (w *OutputWriter) Header(title string) {
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("═══════════════════════════════════════════════════════════════")
	color.New(color.FgCyan, color.Bold).Printf("  %s\n", title)
	color.New(color.FgCyan, color.Bold).Println("═══════════════════════════════════════════════════════════════")
}

// SubHeader prints a sub-header
func (w *OutputWriter) SubHeader(title string) {
	fmt.Println()
	color.New(color.FgWhite, color.Bold).Println("───────────────────────────────────────────────────────────────")
	color.New(color.FgWhite, color.Bold).Printf("  %s\n", title)
	color.New(color.FgWhite, color.Bold).Println("───────────────────────────────────────────────────────────────")
}

// Info prints an info message
func (w *OutputWriter) Info(format string, args ...interface{}) {
	color.New(color.FgBlue).Printf("ℹ ")
	fmt.Printf(format+"\n", args...)
}

// Success prints a success message
func (w *OutputWriter) Success(format string, args ...interface{}) {
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf(format+"\n", args...)
}

// Error prints an error message
func (w *OutputWriter) Error(format string, args ...interface{}) {
	color.New(color.FgRed).Printf("✗ ")
	fmt.Printf(format+"\n", args...)
}

// Warning prints a warning message
func (w *OutputWriter) Warning(format string, args ...interface{}) {
	color.New(color.FgYellow).Printf("⚠ ")
	fmt.Printf(format+"\n", args...)
}

// Skip prints a skip message
func (w *OutputWriter) Skip(format string, args ...interface{}) {
	color.New(color.FgYellow).Printf("○ ")
	fmt.Printf(format+"\n", args...)
}

// Verbose prints a verbose message (only if verbose mode is enabled)
func (w *OutputWriter) Verbose(format string, args ...interface{}) {
	if w.verbose {
		color.New(color.FgHiBlack).Printf("  │ ")
		fmt.Printf(format+"\n", args...)
	}
}

// TestStart prints test case start
func (w *OutputWriter) TestStart(id string, input string, runNum int) {
	inputPreview := truncateString(input, 50)
	if runNum > 1 {
		color.New(color.FgWhite).Printf("► [%s] Run %d: ", id, runNum)
	} else {
		color.New(color.FgWhite).Printf("► [%s] ", id)
	}
	color.New(color.FgHiBlack).Printf("%s", inputPreview)
	fmt.Print(" ")
}

// TestResult prints test case result
func (w *OutputWriter) TestResult(status Status, duration time.Duration) {
	switch status {
	case StatusPassed:
		color.New(color.FgGreen, color.Bold).Printf("PASSED")
	case StatusFailed:
		color.New(color.FgRed, color.Bold).Printf("FAILED")
	case StatusSkipped:
		color.New(color.FgYellow).Printf("SKIPPED")
	case StatusError:
		color.New(color.FgRed, color.Bold).Printf("ERROR")
	case StatusTimeout:
		color.New(color.FgRed).Printf("TIMEOUT")
	}
	color.New(color.FgHiBlack).Printf(" (%s)\n", formatDuration(duration))
}

// TestError prints test error details
func (w *OutputWriter) TestError(err string) {
	color.New(color.FgRed).Printf("  └─ %s\n", err)
}

// TestOutput prints test output (verbose mode)
func (w *OutputWriter) TestOutput(output string) {
	if w.verbose && output != "" {
		outputPreview := truncateString(output, 100)
		color.New(color.FgHiBlack).Printf("  └─ Output: %s\n", outputPreview)
	}
}

// Progress prints progress information
func (w *OutputWriter) Progress(current, total int) {
	percentage := float64(current) / float64(total) * 100
	color.New(color.FgHiBlack).Printf("\r  Progress: %d/%d (%.0f%%)", current, total, percentage)
}

// Summary prints the test summary
func (w *OutputWriter) Summary(summary *Summary, duration time.Duration) {
	w.SubHeader("Summary")

	// Agent info
	color.New(color.FgWhite).Printf("  Agent:     ")
	color.New(color.FgCyan).Printf("%s\n", summary.AgentID)

	if summary.Connector != "" {
		color.New(color.FgWhite).Printf("  Connector: ")
		color.New(color.FgCyan).Printf("%s\n", summary.Connector)
	}

	// Results
	color.New(color.FgWhite).Printf("  Total:     ")
	fmt.Printf("%d\n", summary.Total)

	color.New(color.FgWhite).Printf("  Passed:    ")
	if summary.Passed > 0 {
		color.New(color.FgGreen).Printf("%d\n", summary.Passed)
	} else {
		fmt.Printf("%d\n", summary.Passed)
	}

	color.New(color.FgWhite).Printf("  Failed:    ")
	if summary.Failed > 0 {
		color.New(color.FgRed).Printf("%d\n", summary.Failed)
	} else {
		fmt.Printf("%d\n", summary.Failed)
	}

	if summary.Skipped > 0 {
		color.New(color.FgWhite).Printf("  Skipped:   ")
		color.New(color.FgYellow).Printf("%d\n", summary.Skipped)
	}

	if summary.Errors > 0 {
		color.New(color.FgWhite).Printf("  Errors:    ")
		color.New(color.FgRed).Printf("%d\n", summary.Errors)
	}

	if summary.Timeouts > 0 {
		color.New(color.FgWhite).Printf("  Timeouts:  ")
		color.New(color.FgRed).Printf("%d\n", summary.Timeouts)
	}

	// Pass rate
	passRate := float64(0)
	if summary.Total > 0 {
		passRate = float64(summary.Passed) / float64(summary.Total) * 100
	}
	color.New(color.FgWhite).Printf("  Pass Rate: ")
	if passRate == 100 {
		color.New(color.FgGreen, color.Bold).Printf("%.1f%%\n", passRate)
	} else if passRate >= 80 {
		color.New(color.FgYellow).Printf("%.1f%%\n", passRate)
	} else {
		color.New(color.FgRed).Printf("%.1f%%\n", passRate)
	}

	// Duration
	color.New(color.FgWhite).Printf("  Duration:  ")
	fmt.Printf("%s\n", formatDuration(duration))

	// Stability info (if runs > 1)
	if summary.RunsPerCase > 1 {
		fmt.Println()
		color.New(color.FgWhite, color.Bold).Println("  Stability Analysis:")
		color.New(color.FgWhite).Printf("    Runs/Case:     %d\n", summary.RunsPerCase)
		color.New(color.FgWhite).Printf("    Total Runs:    %d\n", summary.TotalRuns)
		color.New(color.FgWhite).Printf("    Stable Cases:  ")
		if summary.StableCases == summary.Total {
			color.New(color.FgGreen).Printf("%d\n", summary.StableCases)
		} else {
			color.New(color.FgYellow).Printf("%d\n", summary.StableCases)
		}
		color.New(color.FgWhite).Printf("    Unstable:      ")
		if summary.UnstableCases > 0 {
			color.New(color.FgRed).Printf("%d\n", summary.UnstableCases)
		} else {
			fmt.Printf("%d\n", summary.UnstableCases)
		}
	}
}

// OutputFile prints the output file path
func (w *OutputWriter) OutputFile(path string) {
	fmt.Println()
	color.New(color.FgWhite).Printf("  Output: ")
	color.New(color.FgCyan).Printf("%s\n", path)
}

// FinalResult prints the final result banner
func (w *OutputWriter) FinalResult(passed bool) {
	fmt.Println()
	if passed {
		color.New(color.FgGreen, color.Bold).Println("═══════════════════════════════════════════════════════════════")
		color.New(color.FgGreen, color.Bold).Println("  ✨ ALL TESTS PASSED ✨")
		color.New(color.FgGreen, color.Bold).Println("═══════════════════════════════════════════════════════════════")
	} else {
		color.New(color.FgRed, color.Bold).Println("═══════════════════════════════════════════════════════════════")
		color.New(color.FgRed, color.Bold).Println("  ❌ TESTS FAILED")
		color.New(color.FgRed, color.Bold).Println("═══════════════════════════════════════════════════════════════")
	}
	fmt.Println()
}

// DirectOutput prints the agent output directly (for development mode)
func (w *OutputWriter) DirectOutput(output interface{}) {
	if output == nil {
		return
	}

	// Try to format as JSON if it's a complex type
	switch v := output.(type) {
	case string:
		fmt.Println(v)
	case map[string]interface{}, []interface{}:
		// Pretty print JSON
		jsonBytes, err := jsoniter.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Printf("%v\n", output)
		} else {
			fmt.Println(string(jsonBytes))
		}
	default:
		// Try to marshal as JSON
		jsonBytes, err := jsoniter.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Printf("%v\n", output)
		} else {
			fmt.Println(string(jsonBytes))
		}
	}
}

// ScriptTestSummary prints the script test summary
func (w *OutputWriter) ScriptTestSummary(summary *ScriptTestSummary, duration time.Duration) {
	w.SubHeader("Summary")

	// Results
	color.New(color.FgWhite).Printf("  Total:     ")
	fmt.Printf("%d\n", summary.Total)

	color.New(color.FgWhite).Printf("  Passed:    ")
	if summary.Passed > 0 {
		color.New(color.FgGreen).Printf("%d\n", summary.Passed)
	} else {
		fmt.Printf("%d\n", summary.Passed)
	}

	color.New(color.FgWhite).Printf("  Failed:    ")
	if summary.Failed > 0 {
		color.New(color.FgRed).Printf("%d\n", summary.Failed)
	} else {
		fmt.Printf("%d\n", summary.Failed)
	}

	if summary.Skipped > 0 {
		color.New(color.FgWhite).Printf("  Skipped:   ")
		color.New(color.FgYellow).Printf("%d\n", summary.Skipped)
	}

	// Pass rate
	passRate := float64(0)
	if summary.Total > 0 {
		passRate = float64(summary.Passed) / float64(summary.Total) * 100
	}
	color.New(color.FgWhite).Printf("  Pass Rate: ")
	if passRate == 100 {
		color.New(color.FgGreen, color.Bold).Printf("%.1f%%\n", passRate)
	} else if passRate >= 80 {
		color.New(color.FgYellow).Printf("%.1f%%\n", passRate)
	} else {
		color.New(color.FgRed).Printf("%.1f%%\n", passRate)
	}

	// Duration
	color.New(color.FgWhite).Printf("  Duration:  ")
	fmt.Printf("%s\n", formatDuration(duration))
}

// DynamicTestStart outputs the start of a dynamic test
func (w *OutputWriter) DynamicTestStart(id string, checkpointCount int) {
	color.New(color.FgWhite).Printf("► [%s] ", id)
	color.New(color.FgCyan).Printf("(dynamic, %d checkpoints)\n", checkpointCount)
}

// DynamicTurn outputs a single turn in dynamic testing
func (w *OutputWriter) DynamicTurn(turn int, inputSummary string, checkpointsReached, total int) {
	if w.verbose {
		color.New(color.FgHiBlack).Printf("│  ├─ Turn %d: %s ", turn, inputSummary)
		color.New(color.FgCyan).Printf("[%d/%d checkpoints]\n", checkpointsReached, total)
	}
}

// DynamicCheckpoint outputs a checkpoint being reached
func (w *OutputWriter) DynamicCheckpoint(checkpointID string) {
	if w.verbose {
		color.New(color.FgGreen).Printf("│  │  └─ ✓ checkpoint: %s\n", checkpointID)
	}
}

// DynamicTestResult outputs the result of a dynamic test
func (w *OutputWriter) DynamicTestResult(status Status, turns int, checkpoints int, duration time.Duration) {
	color.New(color.FgHiBlack).Printf("  └─ ")

	switch status {
	case StatusPassed:
		color.New(color.FgGreen).Printf("PASSED")
	case StatusFailed:
		color.New(color.FgRed).Printf("FAILED")
	case StatusError:
		color.New(color.FgRed).Printf("ERROR")
	case StatusTimeout:
		color.New(color.FgRed).Printf("TIMEOUT")
	}

	color.New(color.FgHiBlack).Printf(" (%d turns, %d checkpoints, %s)\n", turns, checkpoints, formatDuration(duration))
}

// StabilityResult prints stability analysis result for a test case
func (w *OutputWriter) StabilityResult(sr *StabilityResult) {
	color.New(color.FgWhite).Printf("  [%s] ", sr.ID)

	// Pass rate
	if sr.PassRate == 100 {
		color.New(color.FgGreen).Printf("%.0f%%", sr.PassRate)
	} else if sr.PassRate >= 80 {
		color.New(color.FgYellow).Printf("%.0f%%", sr.PassRate)
	} else {
		color.New(color.FgRed).Printf("%.0f%%", sr.PassRate)
	}

	// Classification
	color.New(color.FgHiBlack).Printf(" (%d/%d) ", sr.Passed, sr.Runs)

	switch sr.StabilityClass {
	case StabilityStable:
		color.New(color.FgGreen).Printf("Stable")
	case StabilityMostlyStable:
		color.New(color.FgYellow).Printf("Mostly Stable")
	case StabilityUnstable:
		color.New(color.FgRed).Printf("Unstable")
	case StabilityHighlyUnstable:
		color.New(color.FgRed, color.Bold).Printf("Highly Unstable")
	}

	// Timing
	color.New(color.FgHiBlack).Printf(" avg:%.0fms\n", sr.AvgDurationMs)
}

// Helper functions

func truncateString(s string, maxLen int) string {
	// Remove newlines and extra spaces
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
