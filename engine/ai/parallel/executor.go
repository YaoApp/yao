package parallel

import (
	"sync"
)

// ModelResponse represents the result of a single model call.
type ModelResponse struct {
	Model  string
	Output string
	Error  error
}

// ParallelExecutor runs multiple model calls concurrently.
type ParallelExecutor struct {
	Models []string
}

func (e *ParallelExecutor) Execute(prompt string) []ModelResponse {
	var wg sync.WaitGroup
	responses := make([]ModelResponse, len(e.Models))

	for i, model := range e.Models {
		wg.Add(1)
		go func(idx int, m string) {
			defer wg.Done()
			// Logic to call the specific model API via Yao engine
			responses[idx] = ModelResponse{Model: m, Output: "Parallel Result"}
		}(i, model)
	}

	wg.Wait()
	return responses
}
