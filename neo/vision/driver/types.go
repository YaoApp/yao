package driver

import (
	"context"
	"io"
)

// Config the vision configuration
type Config struct {
	Storage StorageConfig `json:"storage" yaml:"storage"`
	Model   ModelConfig   `json:"model" yaml:"model"`
}

// StorageConfig the storage configuration
type StorageConfig struct {
	Driver  string                 `json:"driver" yaml:"driver"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

// ModelConfig the model configuration
type ModelConfig struct {
	Driver  string                 `json:"driver" yaml:"driver"`
	Options map[string]interface{} `json:"options" yaml:"options"`
}

// Storage the storage interface
type Storage interface {
	Upload(ctx context.Context, filename string, reader io.Reader, contentType string) (string, error)
	Download(ctx context.Context, fileID string) (io.ReadCloser, string, error)
	URL(ctx context.Context, fileID string) string
}

// Model the vision model interface
type Model interface {
	// Analyze analyzes an image file
	// If prompt is empty, it will use the default prompt from model.options.prompt
	Analyze(ctx context.Context, fileID string, prompt ...string) (map[string]interface{}, error)
}

// Response the vision response
type Response struct {
	FileID      string                 `json:"file_id" yaml:"file_id"`
	URL         string                 `json:"url" yaml:"url"`
	Description map[string]interface{} `json:"description" yaml:"description"`
}
