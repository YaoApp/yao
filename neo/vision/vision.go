package vision

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yaoapp/yao/neo/vision/driver"
	"github.com/yaoapp/yao/neo/vision/driver/local"
	"github.com/yaoapp/yao/neo/vision/driver/openai"
	"github.com/yaoapp/yao/neo/vision/driver/s3"
)

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
func convertOptions(options map[string]interface{}) map[string]interface{} {
	converted := make(map[string]interface{})
	for k, v := range options {
		if str, ok := v.(string); ok {
			converted[k] = parseEnvValue(str)
		} else {
			converted[k] = v
		}
	}
	return converted
}

// Vision the vision service
type Vision struct {
	storage driver.Storage
	model   driver.Model
}

// New create a new vision service
func New(cfg *driver.Config) (*Vision, error) {

	// Parse environment variables in options
	storageOptions := convertOptions(cfg.Storage.Options)
	modelOptions := convertOptions(cfg.Model.Options)

	// Create storage driver
	var storage driver.Storage
	var err error
	switch cfg.Storage.Driver {
	case "local":
		storage, err = local.New(storageOptions)
	case "s3":
		// Convert expiration string to duration if present
		if exp, ok := storageOptions["expiration"].(string); ok {
			if duration, err := time.ParseDuration(exp); err == nil {
				storageOptions["expiration"] = duration
			}
		}
		storage, err = s3.New(storageOptions)
	default:
		return nil, fmt.Errorf("storage driver %s not supported", cfg.Storage.Driver)
	}
	if err != nil {
		return nil, fmt.Errorf("create storage driver error: %s", err.Error())
	}

	// Create model driver
	var model driver.Model
	switch cfg.Model.Driver {
	case "openai":
		model, err = openai.New(modelOptions)
	default:
		return nil, fmt.Errorf("model driver %s not supported", cfg.Model.Driver)
	}
	if err != nil {
		return nil, fmt.Errorf("create model driver error: %s", err.Error())
	}

	return &Vision{
		storage: storage,
		model:   model,
	}, nil
}

// Upload upload file
func (v *Vision) Upload(ctx context.Context, filename string, reader io.Reader, contentType string) (*driver.Response, error) {
	fileID, err := v.storage.Upload(ctx, filename, reader, contentType)
	if err != nil {
		return nil, err
	}

	return &driver.Response{
		FileID: fileID,
		URL:    v.storage.URL(ctx, fileID),
	}, nil
}

// Analyze analyze image using vision model
func (v *Vision) Analyze(ctx context.Context, fileID string, prompt ...string) (*driver.Response, error) {
	if v.model == nil {
		return nil, fmt.Errorf("model is required")
	}

	var url string
	// If the input is already a base64 data URL or a HTTP(S) URL, use it directly
	if strings.HasPrefix(fileID, "data:image/") || strings.HasPrefix(fileID, "http://") || strings.HasPrefix(fileID, "https://") {
		url = fileID
	} else {
		// Otherwise, try to get the URL from storage
		url = v.storage.URL(ctx, fileID)
		if url == "" {
			return nil, fmt.Errorf("failed to get URL for file %s", fileID)
		}
	}

	result, err := v.model.Analyze(ctx, url, prompt...)
	if err != nil {
		return nil, err
	}

	return &driver.Response{
		FileID:      fileID,
		URL:         url,
		Description: result,
	}, nil
}

// Download download file
func (v *Vision) Download(ctx context.Context, fileID string) (io.ReadCloser, string, error) {
	return v.storage.Download(ctx, fileID)
}
