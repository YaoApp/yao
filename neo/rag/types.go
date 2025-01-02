package rag

// Setting RAG settings
type Setting struct {
	Engine      Engine     `json:"engine" yaml:"engine"`
	Vectorizer  Vectorizer `json:"vectorizer" yaml:"vectorizer"`
	Upload      Upload     `json:"upload" yaml:"upload"`
	IndexPrefix string     `json:"index_prefix" yaml:"index_prefix"`
}

// Engine the vector database engine settings
type Engine struct {
	Driver  string                 `json:"driver" yaml:"driver"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

// Vectorizer the text vectorizer settings
type Vectorizer struct {
	Driver  string                 `json:"driver" yaml:"driver"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

// Upload the file upload settings
type Upload struct {
	Async        bool     `json:"async" yaml:"async"`
	AllowedTypes []string `json:"allowed_types" yaml:"allowed_types"`
	ChunkSize    int      `json:"chunk_size" yaml:"chunk_size"`
	ChunkOverlap int      `json:"chunk_overlap" yaml:"chunk_overlap"`
}
