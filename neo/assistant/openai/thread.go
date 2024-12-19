package openai

// Thread the thread struct
type Thread struct {
	ID string `json:"thread_id"`
}

// ThreadList list all threads
func (ast *OpenAI) ThreadList() {}

// ThreadCreate create a new thread
func (ast *OpenAI) ThreadCreate() {}

// ThreadGet get a thread
func (ast *OpenAI) ThreadGet(id string) {}

// ThreadDelete delete a thread
func (ast *OpenAI) ThreadDelete() {}

// ThreadUpdate update a thread
func (ast *OpenAI) ThreadUpdate() {}
