package adapters

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yaoapp/yao/agent/context"
)

// VisionAdapter handles vision (image) capability
// If model doesn't support vision, it removes or converts image content
type VisionAdapter struct {
	*BaseAdapter
	nativeSupport bool
	format        context.VisionFormat
}

// NewVisionAdapter creates a new vision adapter
func NewVisionAdapter(nativeSupport bool, format context.VisionFormat) *VisionAdapter {
	return &VisionAdapter{
		BaseAdapter:   NewBaseAdapter("VisionAdapter"),
		nativeSupport: nativeSupport,
		format:        format,
	}
}

// PreprocessMessages removes or converts image content if not supported
func (a *VisionAdapter) PreprocessMessages(messages []context.Message) ([]context.Message, error) {
	if !a.nativeSupport {
		// No vision support, remove image content
		return a.removeImageContent(messages), nil
	}

	// Check if we need to convert format
	needsConversion := a.format == context.VisionFormatClaude || a.format == context.VisionFormatBase64

	if !needsConversion {
		// Native support with OpenAI format or default, no preprocessing needed
		return messages, nil
	}

	// Convert image_url format to Claude base64 format
	return a.convertToBase64Format(messages)
}

// removeImageContent removes image content from messages
func (a *VisionAdapter) removeImageContent(messages []context.Message) []context.Message {
	processed := make([]context.Message, 0, len(messages))
	for _, msg := range messages {
		processedMsg := msg

		// Handle multimodal content (array of map)
		if contentParts, ok := msg.Content.([]map[string]interface{}); ok {
			filteredParts := make([]map[string]interface{}, 0)

			for _, part := range contentParts {
				partType, _ := part["type"].(string)
				// Skip image content
				if partType != "image_url" && partType != "image" {
					filteredParts = append(filteredParts, part)
				}
			}

			// If all parts were filtered out, add placeholder text
			if len(filteredParts) == 0 {
				processedMsg.Content = "[Image content not supported by this model]"
			} else if len(filteredParts) == 1 {
				if textVal, ok := filteredParts[0]["text"].(string); ok {
					processedMsg.Content = textVal
				} else {
					processedMsg.Content = filteredParts
				}
			} else {
				processedMsg.Content = filteredParts
			}
		}

		processed = append(processed, processedMsg)
	}

	return processed
}

// convertToBase64Format converts image_url format to Claude base64 format
func (a *VisionAdapter) convertToBase64Format(messages []context.Message) ([]context.Message, error) {
	processed := make([]context.Message, 0, len(messages))

	for _, msg := range messages {
		processedMsg := msg

		// Handle multimodal content
		if contentParts, ok := msg.Content.([]map[string]interface{}); ok {
			convertedParts := make([]map[string]interface{}, 0)

			for _, part := range contentParts {
				partType, _ := part["type"].(string)

				if partType == "image_url" {
					// Convert to base64 format
					convertedPart, err := a.convertImageURLToBase64(part)
					if err != nil {
						// If conversion fails, skip this image
						continue
					}
					convertedParts = append(convertedParts, convertedPart)
				} else {
					// Keep non-image parts as-is
					convertedParts = append(convertedParts, part)
				}
			}

			processedMsg.Content = convertedParts
		}

		processed = append(processed, processedMsg)
	}

	return processed, nil
}

// convertImageURLToBase64 converts OpenAI image_url format to Claude base64 format
func (a *VisionAdapter) convertImageURLToBase64(part map[string]interface{}) (map[string]interface{}, error) {
	// Extract URL from image_url object
	imageURLObj, ok := part["image_url"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid image_url format")
	}

	url, ok := imageURLObj["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("missing or invalid URL in image_url")
	}

	// Check if already base64 data URL
	if strings.HasPrefix(url, "data:") {
		// Extract media type and base64 data from data URL
		// Format: data:image/jpeg;base64,<base64_data>
		parts := strings.SplitN(url, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid data URL format")
		}

		// Extract media type from first part
		mediaParts := strings.Split(parts[0], ";")
		mediaType := strings.TrimPrefix(mediaParts[0], "data:")
		base64Data := parts[1]

		return map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": mediaType,
				"data":       base64Data,
			},
		}, nil
	}

	// Download image from URL and convert to base64
	base64Data, mediaType, err := a.downloadAndEncodeImage(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Return Claude/Anthropic format
	return map[string]interface{}{
		"type": "image",
		"source": map[string]interface{}{
			"type":       "base64",
			"media_type": mediaType,
			"data":       base64Data,
		},
	}, nil
}

// downloadAndEncodeImage downloads an image from URL and returns base64 encoded data
func (a *VisionAdapter) downloadAndEncodeImage(url string) (string, string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Download image
	resp, err := client.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Detect media type from Content-Type header
	mediaType := resp.Header.Get("Content-Type")

	// Normalize media type (remove charset and other parameters)
	if mediaType != "" {
		// Split by semicolon to remove parameters like "; charset=utf-8"
		if idx := strings.Index(mediaType, ";"); idx != -1 {
			mediaType = strings.TrimSpace(mediaType[:idx])
		}
	}

	if mediaType == "" {
		// Fallback to detecting from URL extension or default to jpeg
		urlLower := strings.ToLower(url)
		if strings.HasSuffix(urlLower, ".png") {
			mediaType = "image/png"
		} else if strings.HasSuffix(urlLower, ".gif") {
			mediaType = "image/gif"
		} else if strings.HasSuffix(urlLower, ".webp") {
			mediaType = "image/webp"
		} else if strings.Contains(urlLower, ".jpg") || strings.Contains(urlLower, ".jpeg") {
			mediaType = "image/jpeg"
		} else {
			// Default to jpeg
			mediaType = "image/jpeg"
		}
	}

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	return base64Data, mediaType, nil
}
