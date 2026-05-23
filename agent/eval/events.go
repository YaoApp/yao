package eval

// StartEvent is emitted once at the beginning of an eval run (NDJSON type: "start").
type StartEvent struct {
	Type      string     `json:"type"`
	Agent     string     `json:"agent"`
	Connector string     `json:"connector,omitempty"`
	Tai       *TaiStatus `json:"tai,omitempty"`
	GRPC      string     `json:"grpc,omitempty"`
	Total     int        `json:"total"`
}

// ResultEvent is emitted after each test case completes (NDJSON type: "result").
type ResultEvent struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Status     Status            `json:"status"`
	DurationMs int64             `json:"duration_ms"`
	Input      interface{}       `json:"input"`
	Output     string            `json:"output,omitempty"`
	Error      string            `json:"error,omitempty"`
	Assertions []AssertionResult `json:"assertions,omitempty"`
	Trace      *Trace            `json:"trace,omitempty"`
}

// SummaryEvent is emitted once at the end of an eval run (NDJSON type: "summary").
type SummaryEvent struct {
	Type       string   `json:"type"`
	Total      int      `json:"total"`
	Passed     int      `json:"passed"`
	Failed     int      `json:"failed"`
	DurationMs int64    `json:"duration_ms"`
	ExitCode   int      `json:"exit_code"`
	FailedIDs  []string `json:"failed_ids"`
	Suggestion string   `json:"suggestion"`
}

// TaiStatus describes the Tai runtime environment detected during bootstrap.
type TaiStatus struct {
	Bin      string `json:"bin"`
	HostExec bool   `json:"hostexec"`
	Docker   bool   `json:"docker"`
}

// NewStartEvent creates a StartEvent with the "start" type.
func NewStartEvent(agent, connector string, tai *TaiStatus, grpc string, total int) *StartEvent {
	return &StartEvent{
		Type:      "start",
		Agent:     agent,
		Connector: connector,
		Tai:       tai,
		GRPC:      grpc,
		Total:     total,
	}
}

// NewResultEvent creates a ResultEvent from a Result.
func NewResultEvent(r *Result) *ResultEvent {
	ev := &ResultEvent{
		Type:       "result",
		ID:         r.ID,
		Status:     r.Status,
		DurationMs: r.DurationMs,
		Input:      r.Input,
		Error:      r.Error,
		Trace:      r.Trace,
	}
	if s, ok := r.Output.(string); ok {
		ev.Output = s
	}
	return ev
}

// NewSummaryEvent creates a SummaryEvent from a Report.
func NewSummaryEvent(report *Report, suggestion string) *SummaryEvent {
	if report == nil || report.Summary == nil {
		return &SummaryEvent{Type: "summary"}
	}
	s := report.Summary
	exitCode := 0
	if report.HasFailures() {
		exitCode = 1
	}
	var failedIDs []string
	for _, r := range report.Results {
		if r.Status != StatusPassed && r.Status != StatusSkipped {
			failedIDs = append(failedIDs, r.ID)
		}
	}
	return &SummaryEvent{
		Type:       "summary",
		Total:      s.Total,
		Passed:     s.Passed,
		Failed:     s.Failed + s.Errors + s.Timeouts,
		DurationMs: s.DurationMs,
		ExitCode:   exitCode,
		FailedIDs:  failedIDs,
		Suggestion: suggestion,
	}
}

// GenerateSuggestion produces an actionable suggestion string based on the report.
func GenerateSuggestion(report *Report, taiAvailable bool) string {
	if report == nil || report.Summary == nil {
		return ""
	}
	s := report.Summary
	if s.Failed == 0 && s.Errors == 0 && s.Timeouts == 0 {
		return "All tests passed."
	}
	if s.Timeouts > 0 {
		return "Some tests timed out. Consider increasing --timeout or checking agent responsiveness."
	}
	if s.Errors > 0 && !taiAvailable {
		return "Runtime errors detected and Tai is not available. Install Tai or use --tai to specify its path."
	}
	if s.Errors > 0 {
		return "Runtime errors detected. Re-run with --verbose or --json for full trace diagnostics."
	}
	return "Some assertions failed. Re-run with --json to inspect trace details."
}
