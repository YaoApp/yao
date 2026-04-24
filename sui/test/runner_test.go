package test_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/sui/core"
	suitest "github.com/yaoapp/yao/sui/test"
	"github.com/yaoapp/yao/test"
)

func prepare(t *testing.T) {
	t.Helper()
	test.Prepare(t, config.Conf)
	_, err := engine.Load(config.Conf, engine.LoadOption{Action: "sui.test"})
	require.NoError(t, err)
}

func requireAgentSUI(t *testing.T) {
	t.Helper()
	_, has := core.SUIs["agent"]
	if !has {
		t.Skip("no 'agent' SUI loaded")
	}
}

// --- happy path (dashboard page: 4 pass) ---

func TestDiscoverPageTests(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/dashboard",
		JSON:     true,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Greater(t, report.Summary.Total, 0)
	assert.Greater(t, len(report.Pages), 0)
}

func TestRunnerExecuteTests(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/dashboard",
		JSON:     true,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)

	assert.Equal(t, "sui_backend_test", report.Type)
	assert.False(t, report.Metadata.StartedAt.IsZero())
	assert.False(t, report.Metadata.CompletedAt.IsZero())
	assert.Equal(t, 4, report.Summary.Total)
	assert.Equal(t, 4, report.Summary.Passed)
	assert.Equal(t, 0, report.Summary.Failed)
	assert.False(t, report.HasFailures())
}

func TestRunnerWithRunFilter(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/dashboard",
		Run:      "TestGetDashboard",
		JSON:     true,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Equal(t, 1, report.Summary.Total)
	assert.Equal(t, 1, report.Summary.Passed)
}

func TestRunnerFailFast(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/dashboard",
		FailFast: true,
		JSON:     true,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Equal(t, 0, report.Summary.Failed)
}

// --- errors page: covers fail, skip, setAuthorized, reset ---

func TestRunnerWithFailAndSkip(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/errors",
		JSON:     true,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, 6, report.Summary.Total)
	assert.Equal(t, 2, report.Summary.Failed, "TestAlwaysFail + TestRuntimeError should fail")
	assert.Equal(t, 1, report.Summary.Skipped, "TestAlwaysSkip should be skipped")
	assert.Equal(t, 3, report.Summary.Passed)
	assert.True(t, report.HasFailures())

	page := report.Pages[0]
	for _, r := range page.Results {
		switch r.Name {
		case "TestAlwaysFail":
			assert.Equal(t, "failed", r.Status)
			assert.NotEmpty(t, r.Error)
		case "TestAlwaysSkip":
			assert.Equal(t, "skipped", r.Status)
		case "TestSetAuthorizedAndReset":
			assert.Equal(t, "passed", r.Status)
		case "TestCallWithRequestRender":
			assert.Equal(t, "passed", r.Status)
		case "TestCallNonExistentMethod":
			assert.Equal(t, "passed", r.Status)
		case "TestRuntimeError":
			assert.Equal(t, "error", r.Status)
			assert.Contains(t, r.Error, "deliberate runtime error")
		}
	}
}

func TestRunnerFailFastStopsOnError(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/errors",
		FailFast: true,
		JSON:     true,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Equal(t, 1, report.Summary.Failed)
	// fail-fast: executed only up to the first failure
	executed := report.Summary.Passed + report.Summary.Failed + report.Summary.Skipped
	assert.Less(t, executed, report.Summary.Total, "fail-fast should stop before running all tests")
}

// --- verbose (non-JSON) output: covers printHeader, printTestResult, printSummary ---

func TestRunnerVerboseAllPass(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/dashboard",
		Verbose:  true,
		JSON:     false,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Equal(t, 4, report.Summary.Passed)
}

func TestRunnerVerboseWithFailures(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/errors",
		Verbose:  true,
		JSON:     false,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Greater(t, report.Summary.Failed, 0)
}

func TestRunnerVerboseWithData(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	opts := &suitest.Options{
		SUIID:    "agent",
		Template: "agent",
		Page:     "tests.sui-pages/dashboard",
		Data:     `{"key":"value"}`,
		Verbose:  true,
		JSON:     false,
	}

	runner, err := suitest.NewRunner(opts)
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Equal(t, 4, report.Summary.Passed)
}

// --- error paths ---

func TestNewRunnerInvalidSUI(t *testing.T) {
	prepare(t)
	defer test.Clean()

	_, err := suitest.NewRunner(&suitest.Options{SUIID: "nonexistent", Template: "default"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestNewRunnerInvalidTemplate(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	_, err := suitest.NewRunner(&suitest.Options{SUIID: "agent", Template: "nonexistent-tmpl"})
	assert.Error(t, err)
}

func TestRunnerNoMatchingPage(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	runner, err := suitest.NewRunner(&suitest.Options{
		SUIID: "agent", Template: "agent", Page: "xyz-no-match", JSON: true,
	})
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Equal(t, 0, report.Summary.Total)
}

func TestRunnerNoMatchingRun(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	runner, err := suitest.NewRunner(&suitest.Options{
		SUIID: "agent", Template: "agent", Page: "tests.sui-pages/dashboard",
		Run: "NoSuchFunction", JSON: true,
	})
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Equal(t, 0, report.Summary.Total)
}

func TestRunnerVerboseNoTestFiles(t *testing.T) {
	prepare(t)
	defer test.Clean()
	requireAgentSUI(t)

	runner, err := suitest.NewRunner(&suitest.Options{
		SUIID: "agent", Template: "agent", Page: "xyz-no-match", JSON: false,
	})
	require.NoError(t, err)

	report, err := runner.Run()
	require.NoError(t, err)
	assert.Equal(t, 0, report.Summary.Total)
}

// --- types ---

func TestLoadPageConfig(t *testing.T) {
	prepare(t)
	defer test.Clean()

	cfgFile := "assistants/tests/sui-pages/pages/dashboard/dashboard.config"
	exists, _ := application.App.Exists(cfgFile)
	if !exists {
		t.Skipf("test config %s not found", cfgFile)
	}

	cfg, err := suitest.LoadPageConfig(cfgFile)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "Test Dashboard", cfg.Title)
}

func TestLoadPageConfigNotExist(t *testing.T) {
	prepare(t)
	defer test.Clean()

	cfg, err := suitest.LoadPageConfig("nonexistent/path.config")
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestReportHasFailures(t *testing.T) {
	r := &suitest.Report{Summary: &suitest.TestSummary{Failed: 1}}
	assert.True(t, r.HasFailures())
	r.Summary.Failed = 0
	assert.False(t, r.HasFailures())
}
