package test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/context"
)

func TestParseInput_String(t *testing.T) {
	input := "Hello world"
	messages, err := ParseInput(input)
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != context.RoleUser {
		t.Errorf("Expected role 'user', got '%s'", messages[0].Role)
	}

	content, ok := messages[0].Content.(string)
	if !ok {
		t.Fatalf("Expected string content, got %T", messages[0].Content)
	}
	if content != "Hello world" {
		t.Errorf("Expected content 'Hello world', got '%s'", content)
	}
}

func TestParseInput_MessageMap(t *testing.T) {
	input := map[string]interface{}{
		"role":    "user",
		"content": "Test message",
	}

	messages, err := ParseInput(input)
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != context.RoleUser {
		t.Errorf("Expected role 'user', got '%s'", messages[0].Role)
	}
}

func TestParseInput_MessageArray(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"role": "user", "content": "Hello"},
		map[string]interface{}{"role": "assistant", "content": "Hi there"},
		map[string]interface{}{"role": "user", "content": "Follow-up"},
	}

	messages, err := ParseInput(input)
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	if messages[0].Role != context.RoleUser {
		t.Errorf("Expected first message role 'user', got '%s'", messages[0].Role)
	}
	if messages[1].Role != context.RoleAssistant {
		t.Errorf("Expected second message role 'assistant', got '%s'", messages[1].Role)
	}
}

func TestParseInput_ContentParts(t *testing.T) {
	input := map[string]interface{}{
		"role": "user",
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": "Analyze this"},
			map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{
				"url":    "https://example.com/image.jpg",
				"detail": "high",
			}},
		},
	}

	messages, err := ParseInput(input)
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	parts, ok := messages[0].Content.([]context.ContentPart)
	if !ok {
		t.Fatalf("Expected []ContentPart, got %T", messages[0].Content)
	}

	if len(parts) != 2 {
		t.Fatalf("Expected 2 content parts, got %d", len(parts))
	}

	if parts[0].Type != context.ContentText {
		t.Errorf("Expected first part type 'text', got '%s'", parts[0].Type)
	}
	if parts[0].Text != "Analyze this" {
		t.Errorf("Expected text 'Analyze this', got '%s'", parts[0].Text)
	}

	if parts[1].Type != context.ContentImageURL {
		t.Errorf("Expected second part type 'image_url', got '%s'", parts[1].Type)
	}
	if parts[1].ImageURL == nil {
		t.Fatal("Expected ImageURL to be set")
	}
	if parts[1].ImageURL.URL != "https://example.com/image.jpg" {
		t.Errorf("Expected URL 'https://example.com/image.jpg', got '%s'", parts[1].ImageURL.URL)
	}
	if parts[1].ImageURL.Detail != context.DetailHigh {
		t.Errorf("Expected detail 'high', got '%s'", parts[1].ImageURL.Detail)
	}
}

func TestParseInputWithOptions_FileProtocol_Image(t *testing.T) {
	// Create a temporary test image file
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.png")

	// Create a minimal PNG file (1x1 pixel, red)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xFE,
		0xD4, 0xEF, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, // IEND chunk
		0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(imgPath, pngData, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	input := map[string]interface{}{
		"role": "user",
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": "Analyze this image"},
			map[string]interface{}{"type": "image", "source": "file://test.png"},
		},
	}

	opts := &InputOptions{BaseDir: tmpDir}
	messages, err := ParseInputWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ParseInputWithOptions failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	parts, ok := messages[0].Content.([]context.ContentPart)
	if !ok {
		t.Fatalf("Expected []ContentPart, got %T", messages[0].Content)
	}

	if len(parts) != 2 {
		t.Fatalf("Expected 2 content parts, got %d", len(parts))
	}

	// Check image part
	imgPart := parts[1]
	if imgPart.Type != context.ContentImageURL {
		t.Errorf("Expected type 'image_url', got '%s'", imgPart.Type)
	}
	if imgPart.ImageURL == nil {
		t.Fatal("Expected ImageURL to be set")
	}
	if !strings.HasPrefix(imgPart.ImageURL.URL, "data:image/png;base64,") {
		t.Errorf("Expected base64 data URL, got '%s'", imgPart.ImageURL.URL[:50])
	}

	// Verify the base64 content
	b64Part := strings.TrimPrefix(imgPart.ImageURL.URL, "data:image/png;base64,")
	decoded, err := base64.StdEncoding.DecodeString(b64Part)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	if len(decoded) != len(pngData) {
		t.Errorf("Decoded data length mismatch: expected %d, got %d", len(pngData), len(decoded))
	}
}

func TestParseInputWithOptions_FileProtocol_Audio(t *testing.T) {
	// Create a temporary test audio file
	tmpDir := t.TempDir()
	audioPath := filepath.Join(tmpDir, "test.wav")

	// Create a minimal WAV file header
	wavData := []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x24, 0x00, 0x00, 0x00, // File size - 8
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Subchunk1Size (16 for PCM)
		0x01, 0x00, // AudioFormat (1 = PCM)
		0x01, 0x00, // NumChannels (1 = mono)
		0x44, 0xAC, 0x00, 0x00, // SampleRate (44100)
		0x88, 0x58, 0x01, 0x00, // ByteRate
		0x02, 0x00, // BlockAlign
		0x10, 0x00, // BitsPerSample (16)
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x00, 0x00, 0x00, // Subchunk2Size (0 = no data)
	}
	if err := os.WriteFile(audioPath, wavData, 0644); err != nil {
		t.Fatalf("Failed to create test audio: %v", err)
	}

	input := map[string]interface{}{
		"role": "user",
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": "Transcribe this"},
			map[string]interface{}{"type": "audio", "source": "file://test.wav"},
		},
	}

	opts := &InputOptions{BaseDir: tmpDir}
	messages, err := ParseInputWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ParseInputWithOptions failed: %v", err)
	}

	parts, ok := messages[0].Content.([]context.ContentPart)
	if !ok {
		t.Fatalf("Expected []ContentPart, got %T", messages[0].Content)
	}

	// Check audio part
	audioPart := parts[1]
	if audioPart.Type != context.ContentInputAudio {
		t.Errorf("Expected type 'input_audio', got '%s'", audioPart.Type)
	}
	if audioPart.InputAudio == nil {
		t.Fatal("Expected InputAudio to be set")
	}
	if audioPart.InputAudio.Format != "wav" {
		t.Errorf("Expected format 'wav', got '%s'", audioPart.InputAudio.Format)
	}
	if audioPart.InputAudio.Data == "" {
		t.Error("Expected base64 data to be set")
	}
}

func TestParseInputWithOptions_FileProtocol_File(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "document.pdf")

	// Create a minimal PDF file
	pdfData := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\ntrailer\n<<>>\n%%EOF")
	if err := os.WriteFile(pdfPath, pdfData, 0644); err != nil {
		t.Fatalf("Failed to create test PDF: %v", err)
	}

	input := map[string]interface{}{
		"role": "user",
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": "Analyze this document"},
			map[string]interface{}{"type": "file", "source": "file://document.pdf", "name": "my_doc.pdf"},
		},
	}

	opts := &InputOptions{BaseDir: tmpDir}
	messages, err := ParseInputWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ParseInputWithOptions failed: %v", err)
	}

	parts, ok := messages[0].Content.([]context.ContentPart)
	if !ok {
		t.Fatalf("Expected []ContentPart, got %T", messages[0].Content)
	}

	// Check file part
	filePart := parts[1]
	if filePart.Type != context.ContentFile {
		t.Errorf("Expected type 'file', got '%s'", filePart.Type)
	}
	if filePart.File == nil {
		t.Fatal("Expected File to be set")
	}
	if filePart.File.Filename != "my_doc.pdf" {
		t.Errorf("Expected filename 'my_doc.pdf', got '%s'", filePart.File.Filename)
	}
	if !strings.HasPrefix(filePart.File.URL, "data:application/pdf;base64,") {
		t.Errorf("Expected base64 data URL with PDF mime type, got '%s'", filePart.File.URL[:40])
	}
}

func TestParseInputWithOptions_FileProtocol_AbsolutePath(t *testing.T) {
	// Create a temporary test image file
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "absolute.png")

	// Create a minimal PNG file
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xFE,
		0xD4, 0xEF, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
		0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(imgPath, pngData, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Use absolute path
	input := map[string]interface{}{
		"role": "user",
		"content": []interface{}{
			map[string]interface{}{"type": "image", "source": "file://" + imgPath},
		},
	}

	// BaseDir should be ignored for absolute paths
	opts := &InputOptions{BaseDir: "/some/other/dir"}
	messages, err := ParseInputWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ParseInputWithOptions failed: %v", err)
	}

	parts, ok := messages[0].Content.([]context.ContentPart)
	if !ok {
		t.Fatalf("Expected []ContentPart, got %T", messages[0].Content)
	}

	if parts[0].Type != context.ContentImageURL {
		t.Errorf("Expected type 'image_url', got '%s'", parts[0].Type)
	}
}

func TestParseInputWithOptions_FileNotFound(t *testing.T) {
	input := map[string]interface{}{
		"role": "user",
		"content": []interface{}{
			map[string]interface{}{"type": "image", "source": "file://nonexistent.png"},
		},
	}

	opts := &InputOptions{BaseDir: t.TempDir()}
	_, err := ParseInputWithOptions(input, opts)
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to read image file") {
		t.Errorf("Expected 'failed to read image file' error, got: %v", err)
	}
}

func TestResolveFilePath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		baseDir  string
		expected string
	}{
		{
			name:     "relative path with base dir",
			filePath: "fixtures/image.png",
			baseDir:  "/app/tests",
			expected: "/app/tests/fixtures/image.png",
		},
		{
			name:     "relative path without base dir",
			filePath: "fixtures/image.png",
			baseDir:  "",
			expected: "fixtures/image.png",
		},
		{
			name:     "absolute path ignores base dir",
			filePath: "/absolute/path/image.png",
			baseDir:  "/app/tests",
			expected: "/absolute/path/image.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InputOptions{BaseDir: tt.baseDir}
			result := resolveFilePath(tt.filePath, opts)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name     string
		content  interface{}
		expected string
	}{
		{
			name:     "string content",
			content:  "Hello world",
			expected: "Hello world",
		},
		{
			name: "content parts array",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": "First"},
				map[string]interface{}{"type": "image", "source": "file://test.png"},
				map[string]interface{}{"type": "text", "text": "Second"},
			},
			expected: "First\nSecond",
		},
		{
			name:     "nil content",
			content:  nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTextContent(tt.content)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSummarizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "long string truncated",
			input:    "This is a very long message that should be truncated",
			maxLen:   20,
			expected: "This is a very lo...",
		},
		{
			name: "message array - last user message",
			input: []interface{}{
				map[string]interface{}{"role": "user", "content": "First"},
				map[string]interface{}{"role": "assistant", "content": "Response"},
				map[string]interface{}{"role": "user", "content": "Last user message"},
			},
			maxLen:   50,
			expected: "Last user message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SummarizeInput(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
