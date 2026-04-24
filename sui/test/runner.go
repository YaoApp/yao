package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	agenttest "github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/sui/core"
)

// Runner discovers and executes SUI backend tests
type Runner struct {
	opts *Options
	sui  core.SUI
	tmpl core.ITemplate
}

// NewRunner creates a new SUI test runner
func NewRunner(opts *Options) (*Runner, error) {
	sui, has := core.SUIs[opts.SUIID]
	if !has {
		return nil, fmt.Errorf("SUI %q not found", opts.SUIID)
	}

	sid := uuid.New().String()
	sui.WithSid(sid)

	tmpl, err := sui.GetTemplate(opts.Template)
	if err != nil {
		return nil, fmt.Errorf("template %q: %w", opts.Template, err)
	}

	return &Runner{opts: opts, sui: sui, tmpl: tmpl}, nil
}

// Run discovers and executes all matching backend tests, returning a report
func (r *Runner) Run() (*Report, error) {
	startTime := time.Now()
	sid := r.sui.GetSid()

	if !r.opts.JSON {
		r.printHeader(sid)
	}

	pageInfos, err := r.discoverPageTests()
	if err != nil {
		return nil, err
	}

	if !r.opts.JSON {
		fmt.Printf("Found: %d test files\n\n", len(pageInfos))
	}

	report := &Report{
		Type:     "sui_backend_test",
		SUIID:    r.opts.SUIID,
		Template: r.opts.Template,
		Summary:  &TestSummary{},
		Pages:    make([]*PageReport, 0, len(pageInfos)),
		Metadata: &TestMetadata{StartedAt: startTime},
	}

	for _, pi := range pageInfos {
		pageReport, stop := r.runPageTests(pi, sid, report.Summary)
		if pageReport != nil {
			report.Pages = append(report.Pages, pageReport)
		}
		if stop {
			break
		}
	}

	report.Summary.DurationMs = time.Since(startTime).Milliseconds()
	report.Metadata.CompletedAt = time.Now()

	if r.opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(report)
	} else {
		r.printSummary(report.Summary, time.Since(startTime))
	}

	return report, nil
}

// discoverPageTests walks template pages and finds those with backend_test.ts files
func (r *Runner) discoverPageTests() ([]*PageTestInfo, error) {
	pages, err := r.tmpl.Pages()
	if err != nil {
		return nil, fmt.Errorf("listing pages: %w", err)
	}

	var infos []*PageTestInfo
	for _, page := range pages {
		pg := page.Get()
		if pg == nil {
			continue
		}

		if r.opts.Page != "" && !strings.Contains(pg.Route, r.opts.Page) {
			continue
		}

		dir := pg.Path
		testFile := filepath.Join(dir, pg.Name+".backend_test.ts")

		exists, _ := application.App.Exists(testFile)
		if !exists {
			testFile = filepath.Join(dir, pg.Name+".backend_test.js")
			exists, _ = application.App.Exists(testFile)
		}
		if !exists {
			continue
		}

		backendFile := filepath.Join(dir, pg.Name+".backend.ts")
		if ex, _ := application.App.Exists(backendFile); !ex {
			backendFile = filepath.Join(dir, pg.Name+".backend.js")
		}

		prefix := "Api"
		cfgFile := filepath.Join(dir, pg.Name+".config")
		cfg, _ := LoadPageConfig(cfgFile)
		if cfg == nil {
			cfgFile = filepath.Join(dir, pg.Name+".cfg")
			cfg, _ = LoadPageConfig(cfgFile)
		}
		if cfg != nil && cfg.API != nil && cfg.API.Prefix != "" {
			prefix = cfg.API.Prefix
		}

		infos = append(infos, &PageTestInfo{
			Route:       pg.Route,
			Name:        pg.Name,
			BackendFile: backendFile,
			TestFile:    testFile,
			Prefix:      prefix,
		})
	}

	return infos, nil
}

// discoverTests scans a test file for exported Test* functions
func discoverTests(testFile string) ([]*TestCase, error) {
	content, err := application.App.Read(testFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", testFile, err)
	}

	var tests []*TestCase
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "function Test") {
			continue
		}
		name := extractFuncName(line)
		if name != "" && strings.HasPrefix(name, "Test") {
			tests = append(tests, &TestCase{Name: name, Function: name})
		}
	}
	return tests, nil
}

func extractFuncName(line string) string {
	line = strings.TrimPrefix(line, "export ")
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "function ") {
		return ""
	}
	line = strings.TrimPrefix(line, "function ")
	idx := strings.Index(line, "(")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(line[:idx])
}

// filterTests applies --run regex filter
func filterTests(tests []*TestCase, pattern string) ([]*TestCase, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	var filtered []*TestCase
	for _, tc := range tests {
		if re.MatchString(tc.Name) {
			filtered = append(filtered, tc)
		}
	}
	return filtered, nil
}

// runPageTests loads and executes all tests for one page
func (r *Runner) runPageTests(pi *PageTestInfo, sid string, summary *TestSummary) (*PageReport, bool) {
	tests, err := discoverTests(pi.TestFile)
	if err != nil {
		if !r.opts.JSON {
			color.Red("  Error discovering tests in %s: %v\n", pi.TestFile, err)
		}
		return nil, false
	}

	if r.opts.Run != "" {
		tests, err = filterTests(tests, r.opts.Run)
		if err != nil {
			if !r.opts.JSON {
				color.Red("  Invalid --run pattern: %v\n", err)
			}
			return nil, false
		}
	}

	if len(tests) == 0 {
		return nil, false
	}

	if !r.opts.JSON {
		color.New(color.FgWhite, color.Bold).Printf("--- %s ---\n", pi.Route)
	}

	// The backend script is loaded via core.LoadScript which finds *.backend.ts
	// from the page .sui path. We derive the .sui path from the backend file path.
	suiPath := strings.TrimSuffix(strings.TrimSuffix(pi.BackendFile, ".backend.ts"), ".backend.js")
	script, err := core.LoadScript(suiPath, true)
	if err != nil {
		if !r.opts.JSON {
			color.Red("  Failed to load backend script: %v\n", err)
		}
		return nil, false
	}
	if script == nil {
		if !r.opts.JSON {
			color.Yellow("  No backend script found for %s\n", pi.Route)
		}
		return nil, false
	}

	// Load the test file into V8
	testScriptID := "sui-test." + strings.ReplaceAll(strings.Trim(pi.Route, "/"), "/", ".")
	_, err = v8.Load(pi.TestFile, testScriptID)
	if err != nil {
		if !r.opts.JSON {
			color.Red("  Failed to load test script: %v\n", err)
		}
		return nil, false
	}

	// Only count tests after scripts are successfully loaded
	summary.Total += len(tests)

	pageReport := &PageReport{Route: pi.Route, Results: make([]*TestResult, 0, len(tests))}
	stop := false

	for _, tc := range tests {
		result := r.executeTest(tc, script, testScriptID, pi.Prefix, sid)
		pageReport.Results = append(pageReport.Results, result)

		switch result.Status {
		case "passed":
			summary.Passed++
		case "failed", "error":
			summary.Failed++
		case "skipped":
			summary.Skipped++
		}

		if !r.opts.JSON {
			r.printTestResult(tc.Name, result)
		}

		if r.opts.FailFast && (result.Status == "failed" || result.Status == "error") {
			stop = true
			break
		}
	}

	if !r.opts.JSON {
		fmt.Println()
	}

	return pageReport, stop
}

// executeTest runs a single Test* function in V8
func (r *Runner) executeTest(tc *TestCase, script *core.Script, testScriptID, prefix, sid string) (result *TestResult) {
	startTime := time.Now()
	result = &TestResult{Name: tc.Name, Status: "passed"}

	defer func() {
		if rec := recover(); rec != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("panic: %v", rec)
		}
		result.DurationMs = time.Since(startTime).Milliseconds()
	}()

	testScript, ok := v8.Scripts[testScriptID]
	if !ok {
		result.Status = "error"
		result.Error = fmt.Sprintf("test script %q not loaded", testScriptID)
		return
	}

	scriptCtx, err := testScript.NewContext(sid, nil)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("create context: %v", err)
		return
	}
	defer scriptCtx.Close()

	v8ctx := scriptCtx.Context

	// Set share data for Process calls within tests
	err = bridge.SetShareData(v8ctx, v8ctx.Global(), &bridge.Share{
		Sid:    sid,
		Root:   false,
		Global: nil,
	})
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("set share data: %v", err)
		return
	}

	testingT := agenttest.NewTestingT(tc.Name)
	tObj, err := agenttest.NewTestingTObject(v8ctx, testingT)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("create testing.T: %v", err)
		return
	}

	suiCtx := NewSUITestContext(script, prefix, sid)
	ctxObj, err := NewSUITestContextObject(v8ctx, suiCtx)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("create SUIContext: %v", err)
		return
	}

	global := v8ctx.Global()
	fnValue, err := global.Get(tc.Function)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("get function %s: %v", tc.Function, err)
		return
	}

	if !fnValue.IsFunction() {
		result.Status = "error"
		result.Error = fmt.Sprintf("%s is not a function", tc.Function)
		return
	}

	fn, err := fnValue.AsFunction()
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("as function: %v", err)
		return
	}

	_, err = fn.Call(global, tObj, ctxObj)
	if err != nil {
		if testingT.Failed() {
			goto collectResult
		}
		result.Status = "error"
		result.Error = fmt.Sprintf("call error: %v", err)
		return
	}

collectResult:
	result.Logs = testingT.Logs()

	if testingT.Skipped() {
		result.Status = "skipped"
		return
	}

	if testingT.Failed() {
		result.Status = "failed"
		errors := testingT.Errors()
		if len(errors) > 0 {
			result.Error = errors[0]
		}
		if info := testingT.AssertionInfo(); info != nil {
			result.Assertion = &AssertionInfo{
				Type:     info.Type,
				Expected: info.Expected,
				Actual:   info.Actual,
				Message:  info.Message,
			}
		}
		return
	}

	return
}

func (r *Runner) printHeader(sid string) {
	fmt.Println(color.WhiteString("-----------------------"))
	fmt.Println(color.WhiteString("SUI Backend Test"))
	fmt.Printf(color.WhiteString("  SUI:      %s\n"), r.opts.SUIID)
	fmt.Printf(color.WhiteString("  Template: %s\n"), r.opts.Template)
	if r.opts.Data != "" {
		fmt.Printf(color.WhiteString("  Session:  %s\n"), r.opts.Data)
	}
	fmt.Println(color.WhiteString("-----------------------"))
}

func (r *Runner) printTestResult(name string, result *TestResult) {
	switch result.Status {
	case "passed":
		color.Green("  %-40s PASS  (%dms)\n", name, result.DurationMs)
	case "failed":
		color.Red("  %-40s FAIL  (%dms)\n", name, result.DurationMs)
		if result.Error != "" {
			color.Red("    %s\n", result.Error)
		}
	case "error":
		color.Red("  %-40s ERROR (%dms)\n", name, result.DurationMs)
		if result.Error != "" {
			color.Red("    %s\n", result.Error)
		}
	case "skipped":
		color.Yellow("  %-40s SKIP  (%dms)\n", name, result.DurationMs)
	}
}

func (r *Runner) printSummary(s *TestSummary, elapsed time.Duration) {
	passColor := color.New(color.FgGreen)
	failColor := color.New(color.FgRed)

	fmt.Print("RESULTS: ")
	passColor.Printf("%d passed", s.Passed)
	fmt.Print(", ")
	if s.Failed > 0 {
		failColor.Printf("%d failed", s.Failed)
	} else {
		fmt.Printf("%d failed", s.Failed)
	}
	fmt.Printf(", %d skipped (%s)\n", s.Skipped, elapsed.Truncate(time.Millisecond))
}
