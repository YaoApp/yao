package doc

import (
	"testing"

	"github.com/yaoapp/gou/doc"
)

const sampleProcessYAML = `
group: test
type: process
entries:
  - name: test.hello
    desc: Greet someone
    args:
      - name: name
        type: string
        required: true
    return:
      type: string
  - name: test.add
    desc: Add two numbers
    args:
      - name: a
        type: number
        required: true
      - name: b
        type: number
        required: true
    return:
      type: number
`

const sampleShortNameYAML = `
group: http
type: process
entries:
  - name: get
    desc: Send HTTP GET
    args:
      - name: url
        type: string
        required: true
    return:
      type: object
  - name: post
    desc: Send HTTP POST
    args:
      - name: url
        type: string
        required: true
    return:
      type: object
`

const sampleSubGroupYAML = `
group: utils
type: process
entries:
  - name: throw.Forbidden
    desc: Throw 403
    return:
      type: void
`

const sampleRuntimeYAML = `
group: objects
type: js_object
entries:
  - name: log
    desc: Logging utility
    methods:
      - name: Info
        desc: Log info
        args:
          - name: msg
            type: string
            required: true
        return:
          type: void
`

const sampleClassYAML = `
group: objects
type: js_class
entries:
  - name: FS
    desc: File system operations
    args:
      - name: name
        type: string
    methods:
      - name: ReadFile
        desc: Read file
        args:
          - name: path
            type: string
            required: true
        return:
          type: string
`

const sampleFunctionYAML = `
group: functions
type: js_function
entries:
  - name: Process
    desc: Execute a Yao process
    args:
      - name: name
        type: string
        required: true
    return:
      type: any
`

func setupTestData(t *testing.T) {
	t.Helper()
	doc.Reset()
	for _, yaml := range []string{
		sampleProcessYAML, sampleShortNameYAML, sampleSubGroupYAML,
		sampleRuntimeYAML, sampleClassYAML, sampleFunctionYAML,
	} {
		if err := doc.LoadYAML([]byte(yaml)); err != nil {
			t.Fatalf("LoadYAML failed: %v", err)
		}
	}
}

func TestNormalisedNames(t *testing.T) {
	setupTestData(t)

	// "test.hello" already has prefix → stays "test.hello"
	e, ok := doc.Get(doc.TypeProcess, "test.hello")
	if !ok {
		t.Fatal("test.hello not found")
	}
	if e.Name != "test.hello" {
		t.Errorf("name = %q, want test.hello", e.Name)
	}

	// short name "get" with group "http" → normalised to "http.get"
	e, ok = doc.Get(doc.TypeProcess, "http.get")
	if !ok {
		t.Fatal("http.get not found — short name was not normalised")
	}
	if e.Name != "http.get" {
		t.Errorf("name = %q, want http.get", e.Name)
	}

	// sub-group name "throw.Forbidden" with group "utils" → "utils.throw.Forbidden"
	e, ok = doc.Get(doc.TypeProcess, "utils.throw.Forbidden")
	if !ok {
		t.Fatal("utils.throw.Forbidden not found")
	}
	if e.Name != "utils.throw.Forbidden" {
		t.Errorf("name = %q, want utils.throw.Forbidden", e.Name)
	}
}

func TestProcessListCommand(t *testing.T) {
	setupTestData(t)

	entries := doc.List(doc.TypeProcess)
	if len(entries) != 5 {
		t.Errorf("expected 5 process entries, got %d", len(entries))
	}
}

func TestProcessListWithGroupFilter(t *testing.T) {
	setupTestData(t)

	entries := doc.List(doc.TypeProcess, doc.ListOption{Group: "http"})
	if len(entries) != 2 {
		t.Errorf("expected 2 http entries, got %d", len(entries))
	}

	entries = doc.List(doc.TypeProcess, doc.ListOption{Group: "nonexistent"})
	if len(entries) != 0 {
		t.Errorf("expected 0, got %d", len(entries))
	}
}

func TestProcessListWithSearch(t *testing.T) {
	setupTestData(t)

	entries := doc.List(doc.TypeProcess, doc.ListOption{Search: "hello"})
	if len(entries) != 1 {
		t.Fatalf("expected 1, got %d", len(entries))
	}
	if entries[0].Name != "test.hello" {
		t.Errorf("expected test.hello, got %s", entries[0].Name)
	}
}

func TestProcessGet_FullName(t *testing.T) {
	setupTestData(t)

	e, ok := doc.Get(doc.TypeProcess, "http.post")
	if !ok {
		t.Fatal("http.post not found")
	}
	if e.Desc != "Send HTTP POST" {
		t.Errorf("desc = %q", e.Desc)
	}
}

func TestProcessValidate_OK(t *testing.T) {
	setupTestData(t)

	r := doc.Validate(doc.TypeProcess, "test.hello")
	if !r.Valid {
		t.Fatal("expected valid")
	}
	if r.Status != "ok" {
		t.Errorf("status = %q", r.Status)
	}
}

func TestProcessValidate_FullName(t *testing.T) {
	setupTestData(t)

	r := doc.Validate(doc.TypeProcess, "http.get")
	if !r.Valid {
		t.Fatal("expected valid for http.get")
	}
}

func TestProcessValidate_NotFound(t *testing.T) {
	setupTestData(t)

	r := doc.Validate(doc.TypeProcess, "does.not.exist")
	if r.Valid {
		t.Fatal("expected not valid")
	}
}

func TestProcessValidate_Suggestions(t *testing.T) {
	setupTestData(t)

	r := doc.Validate(doc.TypeProcess, "test.hell")
	if r.Valid {
		t.Fatal("expected not valid")
	}
	found := false
	for _, s := range r.Suggestion {
		if s == "test.hello" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected test.hello in suggestions: %v", r.Suggestion)
	}
}

func TestRuntimeList(t *testing.T) {
	setupTestData(t)

	objects := doc.List(doc.TypeJSObject)
	if len(objects) != 1 {
		t.Errorf("expected 1 object, got %d", len(objects))
	}

	classes := doc.List(doc.TypeJSClass)
	if len(classes) != 1 {
		t.Errorf("expected 1 class, got %d", len(classes))
	}

	functions := doc.List(doc.TypeJSFunction)
	if len(functions) != 1 {
		t.Errorf("expected 1 function, got %d", len(functions))
	}
}

func TestRuntimeValidate_Object(t *testing.T) {
	setupTestData(t)

	r := doc.Validate(doc.TypeJSObject, "log")
	if !r.Valid {
		t.Fatal("expected valid")
	}
}

func TestRuntimeValidate_Class(t *testing.T) {
	setupTestData(t)

	r := doc.Validate(doc.TypeJSClass, "FS")
	if !r.Valid {
		t.Fatal("expected valid")
	}
}

func TestRuntimeValidate_Function(t *testing.T) {
	setupTestData(t)

	r := doc.Validate(doc.TypeJSFunction, "Process")
	if !r.Valid {
		t.Fatal("expected valid")
	}
}

func TestFindRuntime(t *testing.T) {
	setupTestData(t)

	e := findRuntime("log")
	if e == nil {
		t.Fatal("expected to find log")
	}

	e = findRuntime("FS")
	if e == nil {
		t.Fatal("expected to find FS")
	}

	e = findRuntime("nonexistent")
	if e != nil {
		t.Fatal("expected nil")
	}
}

func TestFormatArgs(t *testing.T) {
	args := []doc.TypeValue{
		{Name: "name", Type: "string", Required: true},
		{Name: "options", Type: "object"},
	}
	s := formatArgs(args)
	if s != "(name string, options object?)" {
		t.Errorf("formatArgs = %q", s)
	}

	s = formatArgs(nil)
	if s != "()" {
		t.Errorf("formatArgs nil = %q", s)
	}
}

func TestFormatReturn(t *testing.T) {
	if formatReturn(nil) != "void" {
		t.Error("nil should return void")
	}
	tv := &doc.TypeValue{Type: "string"}
	if formatReturn(tv) != "string" {
		t.Error("string type mismatch")
	}
}
