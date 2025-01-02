package rag

import (
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/gou/rag"
	"github.com/yaoapp/gou/rag/driver"
)

// RAG the RAG instance
type RAG struct {
	setting    Setting
	engine     driver.Engine
	vectorizer driver.Vectorizer
	fileUpload driver.FileUpload
}

// parseEnvValue parse environment variable if the value starts with $ENV.
func parseEnvValue(value string) string {
	if strings.HasPrefix(value, "$ENV.") {
		envKey := strings.TrimPrefix(value, "$ENV.")
		if envVal := os.Getenv(envKey); envVal != "" {
			return envVal
		}
	}
	return value
}

// convertOptions convert interface{} options map to string map and parse environment variables
func convertOptions(options map[string]interface{}) map[string]string {
	converted := make(map[string]string)
	for k, v := range options {
		if str, ok := v.(string); ok {
			converted[k] = parseEnvValue(str)
		}
	}
	return converted
}

// New create a new RAG instance
func New(setting Setting) (*RAG, error) {
	if setting.Engine.Driver == "" {
		return nil, fmt.Errorf("engine driver is required")
	}

	if setting.Vectorizer.Driver == "" {
		return nil, fmt.Errorf("vectorizer driver is required")
	}

	// Set default values
	if setting.Upload.ChunkSize == 0 {
		setting.Upload.ChunkSize = 1024
	}

	if setting.Upload.ChunkOverlap == 0 {
		setting.Upload.ChunkOverlap = 256
	}

	if setting.IndexPrefix == "" {
		setting.IndexPrefix = "yao_neo_"
	}

	// Convert options map for vectorizer and handle environment variables
	vectorizerOpts := convertOptions(setting.Vectorizer.Options)

	// Create vectorizer
	vectorizer, err := rag.NewVectorizer(setting.Vectorizer.Driver, driver.VectorizeConfig{
		Model:   vectorizerOpts["model"],
		Options: vectorizerOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("create vectorizer: %v", err)
	}

	// Convert options map for engine and handle environment variables
	engineOpts := convertOptions(setting.Engine.Options)

	// Create engine
	engine, err := rag.NewEngine(setting.Engine.Driver, driver.IndexConfig{
		Options: engineOpts,
	}, vectorizer)
	if err != nil {
		return nil, fmt.Errorf("create engine: %v", err)
	}

	// Create file upload
	fileUpload, err := rag.NewFileUpload(setting.Engine.Driver, engine, vectorizer)
	if err != nil {
		return nil, fmt.Errorf("create file upload: %v", err)
	}

	return &RAG{
		setting:    setting,
		engine:     engine,
		vectorizer: vectorizer,
		fileUpload: fileUpload,
	}, nil
}

// Setting get the RAG settings
func (rag *RAG) Setting() Setting {
	return rag.setting
}

// Engine get the vector database engine
func (rag *RAG) Engine() driver.Engine {
	return rag.engine
}

// Vectorizer get the text vectorizer
func (rag *RAG) Vectorizer() driver.Vectorizer {
	return rag.vectorizer
}

// FileUpload get the file upload handler
func (rag *RAG) FileUpload() driver.FileUpload {
	return rag.fileUpload
}
