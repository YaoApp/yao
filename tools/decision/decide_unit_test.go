//go:build unit

package decision_test

import (
	"encoding/json"
	"testing"

	"github.com/yaoapp/gou/llm"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/tools/decision"
	"github.com/yaoapp/yao/tools/image"
)

// ---------------------------------------------------------------------------
// Single-state error paths
// ---------------------------------------------------------------------------

func TestDecideHandler_NoState(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{nil, map[string]interface{}{}, "", "", 60, map[string]interface{}{}},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, hasErr := m["error"]; !hasErr {
		t.Error("expected error when state is nil and no states")
	}
}

func TestDecideHandler_EmptyQuestions(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{"some state", map[string]interface{}{}, "", "", 60, map[string]interface{}{}},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, hasErr := m["error"]; !hasErr {
		t.Error("expected error when questions is empty")
	}
}

// ---------------------------------------------------------------------------
// Questions JSON round-trip
// ---------------------------------------------------------------------------

func TestDecideHandler_QuestionsRoundTrip(t *testing.T) {
	questionsRaw := map[string]interface{}{
		"sentiment": map[string]interface{}{
			"type":         "choice",
			"instructions": "Classify the sentiment",
			"criteria": map[string]interface{}{
				"positive": "Happy or satisfied",
				"negative": "Angry or frustrated",
			},
		},
		"urgency": map[string]interface{}{
			"type":         "noul",
			"instructions": "Is this urgent?",
		},
		"quality": map[string]interface{}{
			"type":         "score",
			"instructions": "Rate quality",
			"criteria":     []interface{}{"Poor", "Average", "Good"},
		},
	}

	raw, err := json.Marshal(questionsRaw)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var questions map[string]llm.DecisionQuestion
	if err := json.Unmarshal(raw, &questions); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}
	if questions["sentiment"].Type != "choice" {
		t.Errorf("sentiment type = %q, want choice", questions["sentiment"].Type)
	}
	if questions["urgency"].Type != "noul" {
		t.Errorf("urgency type = %q, want noul", questions["urgency"].Type)
	}
	if questions["quality"].Type != "score" {
		t.Errorf("quality type = %q, want score", questions["quality"].Type)
	}
	if questions["sentiment"].Instructions == "" {
		t.Error("sentiment instructions should not be empty")
	}
	if questions["sentiment"].Criteria == nil {
		t.Error("sentiment criteria should not be nil")
	}
}

// ---------------------------------------------------------------------------
// Providers — no registry
// ---------------------------------------------------------------------------

func TestProvidersHandler_NoRegistry(t *testing.T) {
	saved := llmprovider.Global
	llmprovider.Global = nil
	t.Cleanup(func() { llmprovider.Global = saved })

	proc := &process.Process{
		Args: []interface{}{"decision"},
	}
	result := decision.ProvidersHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	if _, hasErr := m["error"]; !hasErr {
		t.Error("expected error when llmprovider registry is nil")
	}
}

// ---------------------------------------------------------------------------
// Batch states — mutual exclusion
// ---------------------------------------------------------------------------

func TestDecideHandler_StatesAndStateMutualExclusion(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{"q": map[string]interface{}{"type": "noul", "instructions": "test"}},
			"", "", 60,
			map[string]interface{}{"states": []interface{}{"a", "b"}},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if errStr == "" {
		t.Error("expected error when both state and states are provided")
	}
	if !contains(errStr, "mutually exclusive") {
		t.Errorf("error = %q, want mention of 'mutually exclusive'", errStr)
	}
}

func TestDecideHandler_StatesEmpty(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			nil,
			map[string]interface{}{"q": map[string]interface{}{"type": "noul", "instructions": "test"}},
			"", "", 60,
			map[string]interface{}{"states": []interface{}{}},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if errStr == "" {
		t.Error("expected error when states array is empty")
	}
}

// ---------------------------------------------------------------------------
// SplitModelConnector — via image package export
// ---------------------------------------------------------------------------

func TestSplitModelConnector_Decision(t *testing.T) {
	p, m := image.SplitModelConnector("", "t123.typesafe:jev-latest")
	if p != "t123.typesafe" {
		t.Errorf("provider = %q, want t123.typesafe", p)
	}
	if m != "jev-latest" {
		t.Errorf("model = %q, want jev-latest", m)
	}
}

func TestSplitModelConnector_ProviderSet(t *testing.T) {
	p, m := image.SplitModelConnector("explicit-provider", "t123.typesafe:jev-latest")
	if p != "explicit-provider" {
		t.Errorf("provider = %q, want explicit-provider", p)
	}
	if m != "t123.typesafe:jev-latest" {
		t.Errorf("model = %q, want t123.typesafe:jev-latest (unchanged)", m)
	}
}

func TestSplitModelConnector_CleanModel(t *testing.T) {
	p, m := image.SplitModelConnector("", "jev-latest")
	if p != "" {
		t.Errorf("provider = %q, want empty", p)
	}
	if m != "jev-latest" {
		t.Errorf("model = %q, want jev-latest", m)
	}
}

// ---------------------------------------------------------------------------
// Issue 1: malformed questions — string instead of map
// ---------------------------------------------------------------------------

func TestDecideHandler_QuestionsIsString(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{"some state", "this is not a map", "", "", 60, map[string]interface{}{}},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if errStr == "" {
		t.Fatal("expected error when questions is a string")
	}
	if !contains(errStr, "JSON object") {
		t.Errorf("error should mention JSON object, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue 2: invalid question type
// ---------------------------------------------------------------------------

func TestDecideHandler_InvalidQuestionType(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{"type": "bogus", "instructions": "test"},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if !contains(errStr, "invalid type") {
		t.Errorf("expected 'invalid type' error, got: %q", errStr)
	}
	if !contains(errStr, "bogus") {
		t.Errorf("error should mention the invalid type value, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue 3: noul with criteria should be rejected
// ---------------------------------------------------------------------------

func TestDecideHandler_NoulWithCriteria(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type":         "noul",
					"instructions": "Is this good?",
					"criteria":     map[string]interface{}{"a": "b"},
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if !contains(errStr, "noul") {
		t.Errorf("expected error mentioning noul, got: %q", errStr)
	}
	if !contains(errStr, "criteria") {
		t.Errorf("expected error mentioning criteria, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue 2 continued: choice missing criteria
// ---------------------------------------------------------------------------

func TestDecideHandler_ChoiceNoCriteria(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type":         "choice",
					"instructions": "Pick one",
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if !contains(errStr, "choice") {
		t.Errorf("expected error mentioning choice, got: %q", errStr)
	}
	if !contains(errStr, "criteria") {
		t.Errorf("expected error mentioning criteria, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue 2: score missing criteria
// ---------------------------------------------------------------------------

func TestDecideHandler_ScoreNoCriteria(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type":         "score",
					"instructions": "Rate it",
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if !contains(errStr, "score") {
		t.Errorf("expected error mentioning score, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue 4: empty state in batch
// ---------------------------------------------------------------------------

func TestDecideHandler_BatchEmptyState(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			nil,
			map[string]interface{}{
				"q1": map[string]interface{}{"type": "noul", "instructions": "test"},
			},
			"", "", 60,
			map[string]interface{}{"states": []interface{}{"good order", ""}},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if errStr == "" {
		t.Fatal("expected error when batch contains empty string state")
	}
	if !contains(errStr, "empty string") {
		t.Errorf("error should mention empty string, got: %q", errStr)
	}
}

func TestDecideHandler_BatchNilState(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			nil,
			map[string]interface{}{
				"q1": map[string]interface{}{"type": "noul", "instructions": "test"},
			},
			"", "", 60,
			map[string]interface{}{"states": []interface{}{"ok", nil, "fine"}},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if errStr == "" {
		t.Fatal("expected error when batch contains nil state")
	}
	if !contains(errStr, "null") {
		t.Errorf("error should mention null, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue 2: missing instructions
// ---------------------------------------------------------------------------

func TestDecideHandler_MissingInstructions(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type": "noul",
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	if !contains(errStr, "instructions") {
		t.Errorf("expected error mentioning instructions, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Valid questions pass validation
// ---------------------------------------------------------------------------

func TestDecideHandler_ValidQuestionsPassValidation(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type":         "choice",
					"instructions": "pick one",
					"criteria":     map[string]interface{}{"a": "A desc", "b": "B desc"},
				},
				"q2": map[string]interface{}{
					"type":         "score",
					"instructions": "rate it",
					"criteria":     []interface{}{"Poor", "Good"},
				},
				"q3": map[string]interface{}{
					"type":         "noul",
					"instructions": "is it good?",
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	errStr, _ := m["error"].(string)
	// Should pass validation but fail at connector resolution (no registry)
	if contains(errStr, "invalid type") || contains(errStr, "instructions") || contains(errStr, "criteria") {
		t.Errorf("valid questions should pass validation, got validation error: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue A: choice criteria as array (wrong type)
// ---------------------------------------------------------------------------

func TestDecideHandler_ChoiceCriteriaArray(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type":         "choice",
					"instructions": "pick one",
					"criteria":     []interface{}{"a", "b"},
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m := result.(map[string]interface{})
	errStr, _ := m["error"].(string)
	if !contains(errStr, "map") || !contains(errStr, "not an array") {
		t.Errorf("expected error about criteria type, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue A: score criteria as map (wrong type)
// ---------------------------------------------------------------------------

func TestDecideHandler_ScoreCriteriaMap(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type":         "score",
					"instructions": "rate it",
					"criteria":     map[string]interface{}{"a": "b"},
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m := result.(map[string]interface{})
	errStr, _ := m["error"].(string)
	if !contains(errStr, "array") || !contains(errStr, "not a map") {
		t.Errorf("expected error about criteria type, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue B: empty choice criteria {}
// ---------------------------------------------------------------------------

func TestDecideHandler_ChoiceCriteriaEmpty(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type":         "choice",
					"instructions": "pick one",
					"criteria":     map[string]interface{}{},
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m := result.(map[string]interface{})
	errStr, _ := m["error"].(string)
	if !contains(errStr, "must not be empty") {
		t.Errorf("expected empty criteria error, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue B: empty score criteria []
// ---------------------------------------------------------------------------

func TestDecideHandler_ScoreCriteriaEmpty(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"some state",
			map[string]interface{}{
				"q1": map[string]interface{}{
					"type":         "score",
					"instructions": "rate it",
					"criteria":     []interface{}{},
				},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m := result.(map[string]interface{})
	errStr, _ := m["error"].(string)
	if !contains(errStr, "must not be empty") {
		t.Errorf("expected empty criteria error, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// Issue C: single --state "" (empty string)
// ---------------------------------------------------------------------------

func TestDecideHandler_SingleEmptyState(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"",
			map[string]interface{}{
				"q1": map[string]interface{}{"type": "noul", "instructions": "test"},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m := result.(map[string]interface{})
	errStr, _ := m["error"].(string)
	if !contains(errStr, "empty string") {
		t.Errorf("expected empty string error, got: %q", errStr)
	}
}

func TestDecideHandler_SingleWhitespaceState(t *testing.T) {
	proc := &process.Process{
		Args: []interface{}{
			"   ",
			map[string]interface{}{
				"q1": map[string]interface{}{"type": "noul", "instructions": "test"},
			},
			"", "", 60, map[string]interface{}{},
		},
	}
	result := decision.DecideHandler(proc)
	m := result.(map[string]interface{})
	errStr, _ := m["error"].(string)
	if !contains(errStr, "empty string") {
		t.Errorf("expected empty string error, got: %q", errStr)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
