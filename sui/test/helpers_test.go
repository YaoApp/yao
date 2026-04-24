package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	"github.com/yaoapp/yao/sui/core"
	yaotest "github.com/yaoapp/yao/test"
)

func prepareInternal(t *testing.T) {
	t.Helper()
	yaotest.Prepare(t, config.Conf)
	_, err := engine.Load(config.Conf, engine.LoadOption{Action: "sui.test"})
	require.NoError(t, err)
}

func TestExtractFuncName(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"export function TestFoo(t, ctx) {", "TestFoo"},
		{"function TestBar(t, ctx) {", "TestBar"},
		{"export function TestBaz() {", "TestBaz"},
		{"// function TestComment(t) {", ""},
		{"const x = 1", ""},
		{"function notTest(t) {", "notTest"},
		{"export function (t) {", ""},
		{"function ", ""},
		{"function TestNoParens", ""},
	}
	for _, tt := range cases {
		got := extractFuncName(tt.line)
		assert.Equal(t, tt.want, got, "extractFuncName(%q)", tt.line)
	}
}

func TestFilterTestsInvalidRegex(t *testing.T) {
	tc := []*TestCase{{Name: "TestFoo"}}
	_, err := filterTests(tc, "[invalid")
	assert.Error(t, err)
}

func TestFilterTestsNoMatch(t *testing.T) {
	tc := []*TestCase{{Name: "TestFoo"}, {Name: "TestBar"}}
	filtered, err := filterTests(tc, "^TestZzz$")
	assert.NoError(t, err)
	assert.Empty(t, filtered)
}

func TestFilterTestsPartialMatch(t *testing.T) {
	tc := []*TestCase{{Name: "TestFoo"}, {Name: "TestBar"}, {Name: "TestFooBar"}}
	filtered, err := filterTests(tc, "Foo")
	assert.NoError(t, err)
	assert.Len(t, filtered, 2)
}

func TestExecuteTestScriptNotLoaded(t *testing.T) {
	r := &Runner{opts: &Options{Timeout: 30 * time.Second}}
	tc := &TestCase{Name: "TestX", Function: "TestX"}

	result := r.executeTest(tc, nil, "nonexistent-script-id", "Api", "test-sid")
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Error, "not loaded")
}

func TestExecuteTestFunctionNotFound(t *testing.T) {
	prepareInternal(t)
	defer yaotest.Clean()

	_, has := core.SUIs["agent"]
	if !has {
		t.Skip("no agent SUI")
	}

	// Load the errors test script
	testFile := "assistants/tests/sui-pages/pages/errors/errors.backend_test.ts"
	testScriptID := "sui-test.helpers-coverage"
	_, err := v8.Load(testFile, testScriptID)
	require.NoError(t, err)

	backendPath := "assistants/tests/sui-pages/pages/errors/errors"
	script, err := core.LoadScript(backendPath, true)
	require.NoError(t, err)
	require.NotNil(t, script)

	r := &Runner{opts: &Options{Timeout: 30 * time.Second}}

	// Call with a function name that doesn't exist in the script
	tc := &TestCase{Name: "NonExistent", Function: "NonExistentFunction"}
	result := r.executeTest(tc, script, testScriptID, "Api", "test-sid")
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Error, "is not a function")
}

func TestLoadPageConfigBadJSON(t *testing.T) {
	prepareInternal(t)
	defer yaotest.Clean()

	// dashboard.html exists but is not valid JSON
	cfg, err := LoadPageConfig("assistants/tests/sui-pages/pages/dashboard/dashboard.html")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}
