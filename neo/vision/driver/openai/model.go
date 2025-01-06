package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/yaoapp/gou/fs"
)

// Model the OpenAI vision model
type Model struct {
	APIKey      string `json:"api_key" yaml:"api_key"`
	Model       string `json:"model" yaml:"model"`
	Compression bool   `json:"compression" yaml:"compression"`
	Prompt      string `json:"prompt" yaml:"prompt"`
}

// New create a new OpenAI vision model
func New(options map[string]interface{}) (*Model, error) {
	model := &Model{
		Model:       "gpt-4-vision-preview",
		Compression: true,
	}

	if apiKey, ok := options["api_key"].(string); ok {
		model.APIKey = apiKey
	}

	if modelName, ok := options["model"].(string); ok {
		model.Model = modelName
	}

	if compression, ok := options["compression"].(bool); ok {
		model.Compression = compression
	}

	if prompt, ok := options["prompt"].(string); ok {
		model.Prompt = prompt
	}

	if model.APIKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	return model, nil
}

// Analyze analyze image using OpenAI vision model
func (model *Model) Analyze(ctx context.Context, fileID string, prompt ...string) (map[string]interface{}, error) {
	if model.APIKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	// Use default prompt if none provided
	userPrompt := model.Prompt
	if len(prompt) > 0 && prompt[0] != "" {
		userPrompt = prompt[0]
	}

	// Check if fileID is a URL or base64 data
	var imageURL string
	if strings.HasPrefix(fileID, "data:image/") {
		// Already a base64 data URL
		imageURL = fileID
	} else if strings.HasPrefix(fileID, "http://") || strings.HasPrefix(fileID, "https://") {
		// Already a URL
		imageURL = fileID
	} else {
		// Try to read the file and convert to base64
		data, err := fs.Get("data")
		if err != nil {
			return nil, fmt.Errorf("failed to get data fs: %w", err)
		}

		reader, err := data.ReadCloser(fileID)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		defer reader.Close()

		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read content: %w", err)
		}

		// Get content type
		contentType := "image/png" // default
		if v, err := data.MimeType(fileID); err == nil {
			contentType = v
		}

		// Convert to base64
		base64Data := base64.StdEncoding.EncodeToString(content)
		imageURL = fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
	}

	// Prepare the request body
	reqBody := map[string]interface{}{
		"model": model.Model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": userPrompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": imageURL,
						},
					},
				},
			},
		},
		"max_tokens": 1000,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", model.APIKey))

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("invalid response format")
	}

	message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// Try to parse content as JSON
	var description map[string]interface{}
	if err := json.Unmarshal([]byte(content), &description); err != nil {
		// If not JSON, use the content as description
		description = map[string]interface{}{
			"description": content,
		}
	}

	return description, nil
}
