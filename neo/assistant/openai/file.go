package openai

// File the file struct
type File struct {
	ID string `json:"file_id"`
}

// FileLists list all files
func (ast *OpenAI) FileLists() {}

// Upload upload a file to an assistant
func (ast *OpenAI) Upload() {}

// FileDelete delete a file
func (ast *OpenAI) FileDelete() {}

// FileContent get the content of a file
func (ast *OpenAI) FileContent() {}

// FileInfo get the information of a file
func (ast *OpenAI) FileInfo() {}
