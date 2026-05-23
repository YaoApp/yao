package test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// TestingT represents the testing object passed to test functions
// It provides assertion methods and test control flow
type TestingT struct {
	mu      sync.Mutex
	name    string
	failed  bool
	skipped bool
	logs    []string
	errors  []string

	// Assertion failure details (for the first failure)
	assertionInfo *ScriptAssertionInfo
}

// NewTestingT creates a new TestingT instance
func NewTestingT(name string) *TestingT {
	return &TestingT{
		name:   name,
		logs:   make([]string, 0),
		errors: make([]string, 0),
	}
}

// Name returns the test name
func (t *TestingT) Name() string {
	return t.name
}

// Failed returns whether the test has failed
func (t *TestingT) Failed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failed
}

// Skipped returns whether the test was skipped
func (t *TestingT) Skipped() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.skipped
}

// Logs returns all log messages
func (t *TestingT) Logs() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string{}, t.logs...)
}

// Errors returns all error messages
func (t *TestingT) Errors() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string{}, t.errors...)
}

// AssertionInfo returns the first assertion failure info
func (t *TestingT) AssertionInfo() *ScriptAssertionInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.assertionInfo
}

// log adds a log message
func (t *TestingT) log(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.logs = append(t.logs, msg)
}

// fail marks the test as failed with an error message
func (t *TestingT) fail(msg string, info *ScriptAssertionInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failed = true
	t.errors = append(t.errors, msg)
	if t.assertionInfo == nil && info != nil {
		t.assertionInfo = info
	}
}

// skip marks the test as skipped
func (t *TestingT) skip(reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.skipped = true
	if reason != "" {
		t.logs = append(t.logs, "SKIP: "+reason)
	}
}

// NewTestingTObject creates a JavaScript testing.T object for V8
func NewTestingTObject(v8ctx *v8go.Context, t *TestingT) (*v8go.Value, error) {
	iso := v8ctx.Isolate()

	// Create the main testing object
	testObj := v8go.NewObjectTemplate(iso)

	// Set name property
	testObj.Set("name", t.name)

	// Create assert object
	assertObj, err := newAssertObject(v8ctx, t)
	if err != nil {
		return nil, err
	}

	// Set methods
	testObj.Set("log", t.logMethod(iso))
	testObj.Set("error", t.errorMethod(iso))
	testObj.Set("skip", t.skipMethod(iso))
	testObj.Set("fail", t.failMethod(iso))
	testObj.Set("fatal", t.fatalMethod(iso))

	// Create instance
	instance, err := testObj.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	obj, err := instance.Value.AsObject()
	if err != nil {
		return nil, err
	}

	// Set assert object
	obj.Set("assert", assertObj)

	// Set failed getter (dynamic)
	obj.Set("failed", t.failed)

	return instance.Value, nil
}

// logMethod implements t.log(...args)
func (t *TestingT) logMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		parts := make([]string, len(args))
		for i, arg := range args {
			parts[i] = arg.String()
		}
		t.log(strings.Join(parts, " "))
		return v8go.Undefined(iso)
	})
}

// errorMethod implements t.error(...args)
func (t *TestingT) errorMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		parts := make([]string, len(args))
		for i, arg := range args {
			parts[i] = arg.String()
		}
		msg := strings.Join(parts, " ")
		t.fail(msg, nil)
		return v8go.Undefined(iso)
	})
}

// skipMethod implements t.skip(reason?)
func (t *TestingT) skipMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		reason := ""
		if len(info.Args()) > 0 {
			reason = info.Args()[0].String()
		}
		t.skip(reason)
		return v8go.Undefined(iso)
	})
}

// failMethod implements t.fail(reason?)
func (t *TestingT) failMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		reason := "test failed"
		if len(info.Args()) > 0 {
			reason = info.Args()[0].String()
		}
		t.fail(reason, nil)
		return v8go.Undefined(iso)
	})
}

// fatalMethod implements t.fatal(reason?)
// Same as fail but intended to stop execution (in JS, this is handled by throwing)
func (t *TestingT) fatalMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		reason := "fatal error"
		if len(info.Args()) > 0 {
			reason = info.Args()[0].String()
		}
		t.fail(reason, nil)
		// Return exception to stop execution
		return bridge.JsException(v8ctx, reason)
	})
}

// newAssertObject creates the assert object with all assertion methods
func newAssertObject(v8ctx *v8go.Context, t *TestingT) (*v8go.Value, error) {
	iso := v8ctx.Isolate()

	assertObj := v8go.NewObjectTemplate(iso)

	// Boolean assertions
	assertObj.Set("True", assertTrueMethod(iso, t))
	assertObj.Set("False", assertFalseMethod(iso, t))

	// Equality assertions
	assertObj.Set("Equal", assertEqualMethod(iso, t))
	assertObj.Set("NotEqual", assertNotEqualMethod(iso, t))

	// Nil assertions
	assertObj.Set("Nil", assertNilMethod(iso, t))
	assertObj.Set("NotNil", assertNotNilMethod(iso, t))

	// String assertions
	assertObj.Set("Contains", assertContainsMethod(iso, t))
	assertObj.Set("NotContains", assertNotContainsMethod(iso, t))

	// Length assertion
	assertObj.Set("Len", assertLenMethod(iso, t))

	// Comparison assertions
	assertObj.Set("Greater", assertGreaterMethod(iso, t))
	assertObj.Set("GreaterOrEqual", assertGreaterOrEqualMethod(iso, t))
	assertObj.Set("Less", assertLessMethod(iso, t))
	assertObj.Set("LessOrEqual", assertLessOrEqualMethod(iso, t))

	// Error assertions
	assertObj.Set("Error", assertErrorMethod(iso, t))
	assertObj.Set("NoError", assertNoErrorMethod(iso, t))

	// Panic assertions
	assertObj.Set("Panic", assertPanicMethod(iso, t))
	assertObj.Set("NoPanic", assertNoPanicMethod(iso, t))

	// Regex assertions
	assertObj.Set("Match", assertMatchMethod(iso, t))
	assertObj.Set("NotMatch", assertNotMatchMethod(iso, t))

	// Type assertion
	assertObj.Set("Type", assertTypeMethod(iso, t))

	// JSON path assertion
	assertObj.Set("JSONPath", assertJSONPathMethod(iso, t))

	// Agent-driven assertion
	assertObj.Set("Agent", assertAgentMethod(iso, t))

	// Create instance
	instance, err := assertObj.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	return instance.Value, nil
}

// Helper function to get optional message argument
func getMessage(args []*v8go.Value, startIdx int) string {
	if len(args) > startIdx && args[startIdx].IsString() {
		return args[startIdx].String()
	}
	return ""
}

// assertTrueMethod implements assert.True(value, message?)
func assertTrueMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			t.fail("True requires a value argument", &ScriptAssertionInfo{Type: "True"})
			return v8go.Undefined(iso)
		}

		value := args[0].Boolean()
		message := getMessage(args, 1)

		if !value {
			msg := "expected true, got false"
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "True",
				Expected: true,
				Actual:   false,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertFalseMethod implements assert.False(value, message?)
func assertFalseMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			t.fail("False requires a value argument", &ScriptAssertionInfo{Type: "False"})
			return v8go.Undefined(iso)
		}

		value := args[0].Boolean()
		message := getMessage(args, 1)

		if value {
			msg := "expected false, got true"
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "False",
				Expected: false,
				Actual:   true,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertEqualMethod implements assert.Equal(actual, expected, message?)
func assertEqualMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()
		if len(args) < 2 {
			t.fail("Equal requires actual and expected arguments", &ScriptAssertionInfo{Type: "Equal"})
			return v8go.Undefined(iso)
		}

		actual, _ := bridge.GoValue(args[0], v8ctx)
		expected, _ := bridge.GoValue(args[1], v8ctx)
		message := getMessage(args, 2)

		if !deepEqual(actual, expected) {
			msg := fmt.Sprintf("expected %v, got %v", expected, actual)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "Equal",
				Expected: expected,
				Actual:   actual,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertNotEqualMethod implements assert.NotEqual(actual, expected, message?)
func assertNotEqualMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()
		if len(args) < 2 {
			t.fail("NotEqual requires actual and expected arguments", &ScriptAssertionInfo{Type: "NotEqual"})
			return v8go.Undefined(iso)
		}

		actual, _ := bridge.GoValue(args[0], v8ctx)
		expected, _ := bridge.GoValue(args[1], v8ctx)
		message := getMessage(args, 2)

		if deepEqual(actual, expected) {
			msg := fmt.Sprintf("expected values to be different, both are %v", actual)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "NotEqual",
				Expected: expected,
				Actual:   actual,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertNilMethod implements assert.Nil(value, message?)
func assertNilMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			t.fail("Nil requires a value argument", &ScriptAssertionInfo{Type: "Nil"})
			return v8go.Undefined(iso)
		}

		isNil := args[0].IsNull() || args[0].IsUndefined()
		message := getMessage(args, 1)

		if !isNil {
			msg := fmt.Sprintf("expected nil, got %v", args[0].String())
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "Nil",
				Expected: nil,
				Actual:   args[0].String(),
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertNotNilMethod implements assert.NotNil(value, message?)
func assertNotNilMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			t.fail("NotNil requires a value argument", &ScriptAssertionInfo{Type: "NotNil"})
			return v8go.Undefined(iso)
		}

		isNil := args[0].IsNull() || args[0].IsUndefined()
		message := getMessage(args, 1)

		if isNil {
			msg := "expected non-nil value, got nil"
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:    "NotNil",
				Actual:  nil,
				Message: message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertContainsMethod implements assert.Contains(str, substr, message?)
func assertContainsMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("Contains requires str and substr arguments", &ScriptAssertionInfo{Type: "Contains"})
			return v8go.Undefined(iso)
		}

		str := args[0].String()
		substr := args[1].String()
		message := getMessage(args, 2)

		if !strings.Contains(str, substr) {
			msg := fmt.Sprintf("expected '%s' to contain '%s'", str, substr)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "Contains",
				Expected: substr,
				Actual:   str,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertNotContainsMethod implements assert.NotContains(str, substr, message?)
func assertNotContainsMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("NotContains requires str and substr arguments", &ScriptAssertionInfo{Type: "NotContains"})
			return v8go.Undefined(iso)
		}

		str := args[0].String()
		substr := args[1].String()
		message := getMessage(args, 2)

		if strings.Contains(str, substr) {
			msg := fmt.Sprintf("expected '%s' to not contain '%s'", str, substr)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "NotContains",
				Expected: substr,
				Actual:   str,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertLenMethod implements assert.Len(value, length, message?)
func assertLenMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()
		if len(args) < 2 {
			t.fail("Len requires value and length arguments", &ScriptAssertionInfo{Type: "Len"})
			return v8go.Undefined(iso)
		}

		value, _ := bridge.GoValue(args[0], v8ctx)
		expectedLen := int(args[1].Integer())
		message := getMessage(args, 2)

		actualLen := getLength(value)

		if actualLen != expectedLen {
			msg := fmt.Sprintf("expected length %d, got %d", expectedLen, actualLen)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "Len",
				Expected: expectedLen,
				Actual:   actualLen,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertGreaterMethod implements assert.Greater(a, b, message?)
func assertGreaterMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("Greater requires two arguments", &ScriptAssertionInfo{Type: "Greater"})
			return v8go.Undefined(iso)
		}

		a := args[0].Number()
		b := args[1].Number()
		message := getMessage(args, 2)

		if !(a > b) {
			msg := fmt.Sprintf("expected %v > %v", a, b)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "Greater",
				Expected: fmt.Sprintf("> %v", b),
				Actual:   a,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertGreaterOrEqualMethod implements assert.GreaterOrEqual(a, b, message?)
func assertGreaterOrEqualMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("GreaterOrEqual requires two arguments", &ScriptAssertionInfo{Type: "GreaterOrEqual"})
			return v8go.Undefined(iso)
		}

		a := args[0].Number()
		b := args[1].Number()
		message := getMessage(args, 2)

		if !(a >= b) {
			msg := fmt.Sprintf("expected %v >= %v", a, b)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "GreaterOrEqual",
				Expected: fmt.Sprintf(">= %v", b),
				Actual:   a,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertLessMethod implements assert.Less(a, b, message?)
func assertLessMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("Less requires two arguments", &ScriptAssertionInfo{Type: "Less"})
			return v8go.Undefined(iso)
		}

		a := args[0].Number()
		b := args[1].Number()
		message := getMessage(args, 2)

		if !(a < b) {
			msg := fmt.Sprintf("expected %v < %v", a, b)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "Less",
				Expected: fmt.Sprintf("< %v", b),
				Actual:   a,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertLessOrEqualMethod implements assert.LessOrEqual(a, b, message?)
func assertLessOrEqualMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("LessOrEqual requires two arguments", &ScriptAssertionInfo{Type: "LessOrEqual"})
			return v8go.Undefined(iso)
		}

		a := args[0].Number()
		b := args[1].Number()
		message := getMessage(args, 2)

		if !(a <= b) {
			msg := fmt.Sprintf("expected %v <= %v", a, b)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "LessOrEqual",
				Expected: fmt.Sprintf("<= %v", b),
				Actual:   a,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertErrorMethod implements assert.Error(err, message?)
func assertErrorMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			t.fail("Error requires an argument", &ScriptAssertionInfo{Type: "Error"})
			return v8go.Undefined(iso)
		}

		// Check if it's null/undefined (no error)
		isError := !args[0].IsNull() && !args[0].IsUndefined()
		message := getMessage(args, 1)

		if !isError {
			msg := "expected an error, got nil"
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:    "Error",
				Actual:  nil,
				Message: message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertNoErrorMethod implements assert.NoError(err, message?)
func assertNoErrorMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			t.fail("NoError requires an argument", &ScriptAssertionInfo{Type: "NoError"})
			return v8go.Undefined(iso)
		}

		// Check if it's null/undefined (no error)
		isError := !args[0].IsNull() && !args[0].IsUndefined()
		message := getMessage(args, 1)

		if isError {
			msg := fmt.Sprintf("expected no error, got %v", args[0].String())
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:    "NoError",
				Actual:  args[0].String(),
				Message: message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertPanicMethod implements assert.Panic(fn, message?)
func assertPanicMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()
		if len(args) < 1 {
			t.fail("Panic requires a function argument", &ScriptAssertionInfo{Type: "Panic"})
			return v8go.Undefined(iso)
		}

		if !args[0].IsFunction() {
			t.fail("Panic requires a function argument", &ScriptAssertionInfo{Type: "Panic"})
			return v8go.Undefined(iso)
		}

		message := getMessage(args, 1)

		// Try to call the function and check if it throws
		fn, _ := args[0].AsFunction()
		_, err := fn.Call(v8ctx.Global())

		if err == nil {
			msg := "expected function to panic, but it didn't"
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:    "Panic",
				Message: message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertNoPanicMethod implements assert.NoPanic(fn, message?)
func assertNoPanicMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()
		if len(args) < 1 {
			t.fail("NoPanic requires a function argument", &ScriptAssertionInfo{Type: "NoPanic"})
			return v8go.Undefined(iso)
		}

		if !args[0].IsFunction() {
			t.fail("NoPanic requires a function argument", &ScriptAssertionInfo{Type: "NoPanic"})
			return v8go.Undefined(iso)
		}

		message := getMessage(args, 1)

		// Try to call the function and check if it throws
		fn, _ := args[0].AsFunction()
		_, err := fn.Call(v8ctx.Global())

		if err != nil {
			msg := fmt.Sprintf("expected function not to panic, but got: %v", err)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:    "NoPanic",
				Actual:  err.Error(),
				Message: message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertMatchMethod implements assert.Match(value, pattern, message?)
func assertMatchMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("Match requires value and pattern arguments", &ScriptAssertionInfo{Type: "Match"})
			return v8go.Undefined(iso)
		}

		value := args[0].String()
		pattern := args[1].String()
		message := getMessage(args, 2)

		re, err := regexp.Compile(pattern)
		if err != nil {
			t.fail(fmt.Sprintf("invalid regex pattern: %v", err), &ScriptAssertionInfo{
				Type:    "Match",
				Message: message,
			})
			return v8go.Undefined(iso)
		}

		if !re.MatchString(value) {
			msg := fmt.Sprintf("expected '%s' to match pattern '%s'", value, pattern)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "Match",
				Expected: pattern,
				Actual:   value,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertNotMatchMethod implements assert.NotMatch(value, pattern, message?)
func assertNotMatchMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("NotMatch requires value and pattern arguments", &ScriptAssertionInfo{Type: "NotMatch"})
			return v8go.Undefined(iso)
		}

		value := args[0].String()
		pattern := args[1].String()
		message := getMessage(args, 2)

		re, err := regexp.Compile(pattern)
		if err != nil {
			t.fail(fmt.Sprintf("invalid regex pattern: %v", err), &ScriptAssertionInfo{
				Type:    "NotMatch",
				Message: message,
			})
			return v8go.Undefined(iso)
		}

		if re.MatchString(value) {
			msg := fmt.Sprintf("expected '%s' to not match pattern '%s'", value, pattern)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "NotMatch",
				Expected: pattern,
				Actual:   value,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertTypeMethod implements assert.Type(value, typeName, message?)
func assertTypeMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			t.fail("Type requires value and typeName arguments", &ScriptAssertionInfo{Type: "Type"})
			return v8go.Undefined(iso)
		}

		value := args[0]
		expectedType := args[1].String()
		message := getMessage(args, 2)

		actualType := getJsType(value)

		if actualType != expectedType {
			msg := fmt.Sprintf("expected type '%s', got '%s'", expectedType, actualType)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "Type",
				Expected: expectedType,
				Actual:   actualType,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertJSONPathMethod implements assert.JSONPath(obj, path, expected, message?)
func assertJSONPathMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()
		if len(args) < 3 {
			t.fail("JSONPath requires obj, path, and expected arguments", &ScriptAssertionInfo{Type: "JSONPath"})
			return v8go.Undefined(iso)
		}

		obj, _ := bridge.GoValue(args[0], v8ctx)
		path := args[1].String()
		expected, _ := bridge.GoValue(args[2], v8ctx)
		message := getMessage(args, 3)

		// Use the existing extractPath function from assert.go
		asserter := &Asserter{}
		actual := asserter.extractPath(obj, strings.TrimPrefix(path, "$."))

		if !deepEqual(actual, expected) {
			msg := fmt.Sprintf("path '%s': expected %v, got %v", path, expected, actual)
			if message != "" {
				msg = message
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:     "JSONPath",
				Expected: expected,
				Actual:   actual,
				Message:  message,
			})
		}
		return v8go.Undefined(iso)
	})
}

// assertAgentMethod implements assert.Agent(response, agentID, options?)
// Uses a validator agent to check the response
// agentID is the direct agent ID (no "agents:" prefix needed)
func assertAgentMethod(iso *v8go.Isolate, t *TestingT) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()
		if len(args) < 2 {
			t.fail("Agent requires response and agentID arguments", &ScriptAssertionInfo{Type: "Agent"})
			return v8go.Undefined(iso)
		}

		response, _ := bridge.GoValue(args[0], v8ctx)
		agentID := args[1].String()

		// Get options if provided
		var options map[string]interface{}
		if len(args) > 2 && args[2].IsObject() {
			optVal, _ := bridge.GoValue(args[2], v8ctx)
			options, _ = optVal.(map[string]interface{})
		}

		// Build assertion with agents: prefix
		assertion := &Assertion{
			Type: "agent",
			Use:  "agents:" + agentID,
		}

		// Extract criteria and metadata from options
		if options != nil {
			if criteria, ok := options["criteria"]; ok {
				assertion.Value = criteria
			}
			if metadata, ok := options["metadata"].(map[string]interface{}); ok {
				assertion.Options = &AssertionOptions{Metadata: metadata}
			}
			if connector, ok := options["connector"].(string); ok {
				if assertion.Options == nil {
					assertion.Options = &AssertionOptions{}
				}
				assertion.Options.Connector = connector
			}
		}

		// Use the asserter to validate
		asserter := &Asserter{}
		result := asserter.assertAgent(assertion, response, nil)

		if !result.Passed {
			msg := result.Message
			if msg == "" {
				msg = "agent assertion failed"
			}
			t.fail(msg, &ScriptAssertionInfo{
				Type:    "Agent",
				Actual:  response,
				Message: msg,
			})
		}

		return v8go.Undefined(iso)
	})
}

// Helper functions

// deepEqual performs deep equality comparison
func deepEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Try JSON comparison for complex types
	aJSON, errA := json.Marshal(a)
	bJSON, errB := json.Marshal(b)
	if errA == nil && errB == nil {
		return string(aJSON) == string(bJSON)
	}

	// Fall back to reflect.DeepEqual
	return reflect.DeepEqual(a, b)
}

// getLength returns the length of a value (array, string, map)
func getLength(v interface{}) int {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case string:
		return len(val)
	case []interface{}:
		return len(val)
	case map[string]interface{}:
		return len(val)
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
			return rv.Len()
		}
	}
	return 0
}

// getJsType returns the JavaScript type name of a value
func getJsType(v *v8go.Value) string {
	if v.IsNull() {
		return "null"
	}
	if v.IsUndefined() {
		return "undefined"
	}
	if v.IsString() {
		return "string"
	}
	if v.IsNumber() {
		return "number"
	}
	if v.IsBoolean() {
		return "boolean"
	}
	if v.IsArray() {
		return "array"
	}
	if v.IsFunction() {
		return "function"
	}
	if v.IsObject() {
		return "object"
	}
	return "unknown"
}
