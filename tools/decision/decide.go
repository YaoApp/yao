package decision

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/llm"
	"github.com/yaoapp/gou/process"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/tools/image"
)

//go:embed decide_schema.json
var DecideSchemaJSON []byte

// DecideHandler is the tools.decision_decide process handler.
//
// Resolution order for the decision connector (aligned with image_read):
//  1. Explicit provider arg -> ResolveConnector(provider)
//  2. Role "use::decision" (respects system/team/user scope)
//  3. Capability discovery: first provider with "decision" capability
//
// Args layout (must match x-process-args in decide_schema.json):
//
//	[0] state       interface{}
//	[1] questions   map[string]interface{}
//	[2] model       string
//	[3] provider    string
//	[4] timeout     int (seconds, default 60)
//	[5] allArgs     map[string]interface{} (contains states, etc.)
func DecideHandler(proc *process.Process) interface{} {
	// 1. state
	var state interface{}
	if len(proc.Args) > 0 {
		state = proc.Args[0]
	}

	// 2. questions — type-check before ArgsMap to catch malformed input
	if len(proc.Args) > 1 {
		if _, isStr := proc.Args[1].(string); isStr {
			return map[string]interface{}{"error": "questions must be a JSON object, not a string; check that --questions contains valid JSON (e.g. '{\"q\":{\"type\":\"noul\",\"instructions\":\"...\"}}')"}
		}
	}
	questionsRaw := proc.ArgsMap(1)

	// 3. model + provider
	model := proc.ArgsString(2, "")
	provider := proc.ArgsString(3, "")
	provider, model = image.SplitModelConnector(provider, model)

	// 4. timeout
	timeout := proc.ArgsInt(4, 60)
	if timeout <= 0 {
		timeout = 60
	}

	// 5. allArgs — for batch states
	allArgs := proc.ArgsMap(5)

	// 6. auth
	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	// 7. mutual exclusion: state vs states
	statesRaw, hasStates := allArgs["states"]
	if state != nil && hasStates {
		return map[string]interface{}{"error": "state and states are mutually exclusive; provide one or the other"}
	}

	// 8. validate questions
	if len(questionsRaw) == 0 {
		return map[string]interface{}{"error": "questions is required and must not be empty"}
	}
	questionsJSON, err := json.Marshal(questionsRaw)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("marshal questions: %v", err)}
	}
	var questions map[string]llm.DecisionQuestion
	if err := json.Unmarshal(questionsJSON, &questions); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("parse questions: %v", err)}
	}

	// 9. client-side question validation
	if err := validateQuestions(questions); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	// 9b. validate state/states before connector resolution
	if hasStates {
		if err := validateBatchStates(statesRaw); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
	} else {
		if state == nil {
			return map[string]interface{}{"error": "state is required"}
		}
		if str, isStr := state.(string); isStr && strings.TrimSpace(str) == "" {
			return map[string]interface{}{"error": "state must not be an empty string"}
		}
	}

	// 10. resolve connector
	conn, err := resolveDecisionConnector(authInfo, provider)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	dc, ok := conn.(llm.DecisionConnector)
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf("connector %q does not implement DecisionConnector", conn.ID())}
	}

	// 11. dispatch: single state or batch states
	if hasStates {
		return handleBatchStates(dc, statesRaw, questions, model, timeout)
	}

	return handleSingleState(dc, state, questions, model, timeout)
}

// handleSingleState executes a single decision call with timeout.
func handleSingleState(dc llm.DecisionConnector, state interface{}, questions map[string]llm.DecisionQuestion, model string, timeout int) interface{} {
	req := &llm.DecisionRequest{
		State:     state,
		Questions: questions,
		Model:     model,
	}

	type decideResult struct {
		resp *llm.DecisionResponse
		err  error
	}
	ch := make(chan decideResult, 1)
	go func() {
		resp, err := dc.Decide(req)
		ch <- decideResult{resp, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("decision failed: %v", r.err)}
		}
		result := map[string]interface{}{
			"answers": r.resp.Answers,
			"model":   r.resp.Model,
		}
		if r.resp.Usage != nil {
			result["usage"] = r.resp.Usage
		}
		return result
	// TypeSafe http.Client has its own 60s timeout; this outer timeout
	// is a safety net. After select returns, the goroutine will be
	// cleaned up when the HTTP call completes or times out.
	case <-time.After(time.Duration(timeout) * time.Second):
		return map[string]interface{}{"error": fmt.Sprintf("decision timed out after %ds", timeout)}
	}
}

// handleBatchStates executes parallel decision calls for each state in the array.
// Timeout applies to the entire batch operation.
func handleBatchStates(dc llm.DecisionConnector, statesRaw interface{}, questions map[string]llm.DecisionQuestion, model string, timeout int) interface{} {
	statesArr := statesRaw.([]interface{}) // validated by validateBatchStates

	type batchItem struct {
		index  int
		result map[string]interface{}
	}

	results := make([]map[string]interface{}, len(statesArr))
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, 5) // concurrency limit

	for i, s := range statesArr {
		wg.Add(1)
		go func(idx int, state interface{}) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			req := &llm.DecisionRequest{
				State:     state,
				Questions: questions,
				Model:     model,
			}
			resp, err := dc.Decide(req)

			item := map[string]interface{}{"state_index": idx}
			if err != nil {
				item["error"] = err.Error()
			} else {
				item["answers"] = resp.Answers
				item["model"] = resp.Model
				if resp.Usage != nil {
					item["usage"] = resp.Usage
				}
			}
			mu.Lock()
			results[idx] = item
			mu.Unlock()
		}(i, s)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return map[string]interface{}{"results": results}
	case <-time.After(time.Duration(timeout) * time.Second):
		// Collect whatever completed so far
		mu.Lock()
		partial := make([]map[string]interface{}, len(statesArr))
		for i, r := range results {
			if r != nil {
				partial[i] = r
			} else {
				partial[i] = map[string]interface{}{
					"state_index": i,
					"error":       fmt.Sprintf("timed out after %ds", timeout),
				}
			}
		}
		mu.Unlock()
		return map[string]interface{}{
			"results": partial,
			"error":   fmt.Sprintf("batch timed out after %ds; partial results returned", timeout),
		}
	}
}

// resolveDecisionConnector resolves the decision connector in priority order
// (aligned with image_read's provider/role pattern):
//  1. Explicit provider -> ResolveConnector(provider)
//  2. Role "use::decision" (respects system/team/user scope)
//  3. Capability discovery: first provider with "decision" capability
func resolveDecisionConnector(authInfo *oauthTypes.AuthorizedInfo, provider string) (connector.Connector, error) {
	if provider != "" {
		conn, _, err := agentLLM.ResolveConnector(provider, authInfo)
		if err != nil {
			return nil, fmt.Errorf("resolve provider %s: %w", provider, err)
		}
		return conn, nil
	}

	conn, _, err := agentLLM.ResolveConnector("use::decision", authInfo)
	if err == nil {
		return conn, nil
	}

	connectorID := findFirstDecisionConnector(authInfo)
	if connectorID == "" {
		return nil, fmt.Errorf("no decision provider available (user=%s team=%s); configure a 'decision' role or a provider with 'decision' capability",
			authInfo.GetUserID(), authInfo.GetTeamID())
	}

	conn, _, err = agentLLM.ResolveConnector(connectorID, authInfo)
	if err != nil {
		return nil, fmt.Errorf("resolve connector %s: %w", connectorID, err)
	}
	return conn, nil
}

// findFirstDecisionConnector returns the connector ID of the first available
// decision provider, or empty string if none found.
func findFirstDecisionConnector(authInfo *oauthTypes.AuthorizedInfo) string {
	if llmprovider.Global == nil {
		return ""
	}
	providers, err := image.ListProvidersByCapability("decision", authInfo)
	if err != nil || len(providers) == 0 {
		return ""
	}
	if len(providers[0].Models) == 0 {
		return ""
	}
	return providers[0].Models[0].ConnectorID
}

// validateBatchStates checks the states array for type, emptiness, and nil items
// before the connector is resolved.
func validateBatchStates(statesRaw interface{}) error {
	statesArr, ok := statesRaw.([]interface{})
	if !ok {
		return fmt.Errorf("states must be an array")
	}
	if len(statesArr) == 0 {
		return fmt.Errorf("states array must not be empty")
	}
	for i, s := range statesArr {
		if s == nil {
			return fmt.Errorf("states[%d] is null; every batch item must have a value", i)
		}
		if str, isStr := s.(string); isStr && strings.TrimSpace(str) == "" {
			return fmt.Errorf("states[%d] is an empty string; every batch item must have a value", i)
		}
	}
	return nil
}

// validQuestionTypes are the accepted values for DecisionQuestion.Type.
var validQuestionTypes = map[string]bool{
	"choice": true,
	"score":  true,
	"noul":   true,
}

// validateQuestions performs client-side validation before sending to the API.
// Catches invalid type values, mismatched criteria types, and empty criteria
// early with clear messages instead of opaque HTTP 400/422 from the provider.
func validateQuestions(questions map[string]llm.DecisionQuestion) error {
	for id, q := range questions {
		if !validQuestionTypes[q.Type] {
			return fmt.Errorf("question %q: invalid type %q; must be one of: choice, score, noul", id, q.Type)
		}
		if q.Instructions == "" {
			return fmt.Errorf("question %q: instructions is required", id)
		}
		switch q.Type {
		case "noul":
			if q.Criteria != nil {
				return fmt.Errorf("question %q: noul questions must not have criteria (omit the field)", id)
			}
		case "choice":
			if q.Criteria == nil {
				return fmt.Errorf("question %q: choice questions require criteria as {option: description} map", id)
			}
			m, ok := q.Criteria.(map[string]interface{})
			if !ok {
				return fmt.Errorf("question %q: choice criteria must be a {option: description} map, not an array", id)
			}
			if len(m) == 0 {
				return fmt.Errorf("question %q: choice criteria must not be empty", id)
			}
		case "score":
			if q.Criteria == nil {
				return fmt.Errorf("question %q: score questions require criteria as [level_labels] array", id)
			}
			arr, ok := q.Criteria.([]interface{})
			if !ok {
				return fmt.Errorf("question %q: score criteria must be a [level_labels] array, not a map", id)
			}
			if len(arr) == 0 {
				return fmt.Errorf("question %q: score criteria must not be empty", id)
			}
		}
	}
	return nil
}
