package caller

import (
	"sync"

	agentContext "github.com/yaoapp/yao/agent/context"
)

// Orchestrator handles parallel agent calls with different concurrency patterns
// Modeled after JavaScript Promise patterns (all, any, race)
type Orchestrator struct {
	ctx *agentContext.Context
}

// NewOrchestrator creates a new Orchestrator for parallel agent calls
func NewOrchestrator(ctx *agentContext.Context) *Orchestrator {
	return &Orchestrator{ctx: ctx}
}

// callResult is used internally to pass results through channels
type callResult struct {
	idx    int
	result *Result
}

// All executes all agent calls and waits for all to complete (like Promise.all)
// Returns results in the same order as requests, regardless of completion order
// Each call uses a forked context to avoid race conditions on shared state
func (o *Orchestrator) All(reqs []*Request) []*Result {
	if len(reqs) == 0 {
		return []*Result{}
	}

	results := make([]*Result, len(reqs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r *Request) {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					mu.Lock()
					results[idx] = &Result{
						AgentID: r.AgentID,
						Error:   "agent call panic recovered",
					}
					mu.Unlock()
				}
			}()

			// Use forked context to avoid race conditions
			result := o.callAgentWithForkedContext(r)
			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	return results
}

// Any returns as soon as any agent call succeeds (has non-error result) (like Promise.any)
// Other calls continue in background but results are discarded after first success
// Returns all results received so far when first success is found
// Each call uses a forked context to avoid race conditions on shared state
func (o *Orchestrator) Any(reqs []*Request) []*Result {
	if len(reqs) == 0 {
		return []*Result{}
	}

	results := make([]*Result, len(reqs))
	resultChan := make(chan callResult, len(reqs))

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r *Request) {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					// Send panic result through channel
					select {
					case <-done:
					case resultChan <- callResult{idx: idx, result: &Result{
						AgentID: r.AgentID,
						Error:   "agent call panic recovered",
					}}:
					}
				}
			}()

			// Check if done before starting
			select {
			case <-done:
				return
			default:
			}

			// Use forked context to avoid race conditions
			result := o.callAgentWithForkedContext(r)

			// Try to send result
			select {
			case <-done:
				// Already found a successful result
			case resultChan <- callResult{idx: idx, result: result}:
			}
		}(i, req)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results until we find one with success (no error and has content)
	var foundSuccess bool
	for res := range resultChan {
		results[res.idx] = res.result
		// Check if this result is successful (no error)
		if !foundSuccess && res.result != nil && res.result.Error == "" {
			foundSuccess = true
			close(done) // Signal other goroutines to stop
		}
	}

	return results
}

// Race returns as soon as any agent call completes (like Promise.race)
// Returns immediately when first result arrives, regardless of success/failure
// Note: Still waits for all goroutines to complete before returning to avoid resource leaks
// Each call uses a forked context to avoid race conditions on shared state
func (o *Orchestrator) Race(reqs []*Request) []*Result {
	if len(reqs) == 0 {
		return []*Result{}
	}

	results := make([]*Result, len(reqs))
	resultChan := make(chan callResult, len(reqs))

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i, req := range reqs {
		wg.Add(1)
		go func(idx int, r *Request) {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					// Send panic result through channel
					select {
					case <-done:
					case resultChan <- callResult{idx: idx, result: &Result{
						AgentID: r.AgentID,
						Error:   "agent call panic recovered",
					}}:
					}
				}
			}()

			// Check if done before starting
			select {
			case <-done:
				return
			default:
			}

			// Use forked context to avoid race conditions
			result := o.callAgentWithForkedContext(r)

			// Try to send result
			select {
			case <-done:
				// Already got first result
			case resultChan <- callResult{idx: idx, result: result}:
			}
		}(i, req)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Get first result and signal others to stop
	var gotFirst bool
	for res := range resultChan {
		results[res.idx] = res.result
		if !gotFirst {
			gotFirst = true
			close(done) // Signal other goroutines to stop
		}
	}

	return results
}

// callAgent executes a single agent call using the AgentGetterFunc
// This method handles context sharing and result extraction
func (o *Orchestrator) callAgent(req *Request) *Result {
	return o.callAgentWithContext(o.ctx, req)
}

// callAgentWithForkedContext executes a single agent call with a forked context
// This is used by batch operations (All/Any/Race) to avoid race conditions
// when multiple goroutines modify shared context state (Stack, Logger, etc.)
func (o *Orchestrator) callAgentWithForkedContext(req *Request) *Result {
	// Fork the context to get independent Stack and Logger
	forkedCtx := o.ctx.Fork()
	return o.callAgentWithContext(forkedCtx, req)
}

// callAgentWithContext executes a single agent call with the given context
// This is the core implementation used by both callAgent and callAgentWithForkedContext
func (o *Orchestrator) callAgentWithContext(ctx *agentContext.Context, req *Request) *Result {
	if req == nil {
		return &Result{Error: "nil request"}
	}

	result := &Result{
		AgentID: req.AgentID,
	}

	// Get the agent using the getter function
	if AgentGetterFunc == nil {
		result.Error = "agent getter not initialized"
		return result
	}

	agent, err := AgentGetterFunc(req.AgentID)
	if err != nil {
		result.Error = "failed to get agent: " + err.Error()
		return result
	}

	// Mark this as an agent-to-agent fork call for proper source tracking
	// RefererAgentFork distinguishes ctx.agent.Call from delegate calls
	ctx.Referer = agentContext.RefererAgentFork

	// Build context options for the call
	var ctxOpts *agentContext.Options
	if req.Options != nil {
		ctxOpts = req.Options.ToContextOptions()
	} else {
		ctxOpts = &agentContext.Options{}
	}

	// If request has a handler, set OnMessage callback
	if req.Handler != nil {
		if ctxOpts == nil {
			ctxOpts = &agentContext.Options{}
		}
		// Set OnMessage to receive SSE messages
		ctxOpts.OnMessage = req.Handler
	}

	// Execute the agent call with the provided context
	// The agent.Stream method will use the context's Writer for output
	resp, err := agent.Stream(ctx, req.Messages, ctxOpts)
	if err != nil {
		result.Error = "agent call failed: " + err.Error()
		return result
	}

	result.Response = resp

	// Extract content from completion if available
	if resp != nil && resp.Completion != nil {
		result.Content = extractContentFromCompletion(resp.Completion)
	}

	return result
}

// extractContentFromCompletion extracts the text content from a completion response
func extractContentFromCompletion(completion *agentContext.CompletionResponse) string {
	if completion == nil {
		return ""
	}

	// Content can be string or []ContentPart
	switch content := completion.Content.(type) {
	case string:
		return content
	case []interface{}:
		// Handle array of content parts - extract text parts
		var texts []string
		for _, part := range content {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partType, ok := partMap["type"].(string); ok && partType == "text" {
					if text, ok := partMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		if len(texts) > 0 {
			return texts[0] // Return first text content
		}
	}

	return ""
}
