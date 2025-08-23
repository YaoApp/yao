package types

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// Test data for configuration parsing (providers are now loaded from directories)
const testConfigJSON = `{
	"vector": {
		"driver": "qdrant",
		"config": {
			"host": "127.0.0.1",
			"port": 6333
		}
	},
	"graph": {
		"driver": "neo4j",
		"config": {
			"url": "neo4j://127.0.0.1:7686"
		}
	},
	"store": "test_store",
	"pdf": {
		"convert_tool": "pdftoppm",
		"tool_path": "/usr/bin/pdftoppm"
	},
	"ffmpeg": {
		"ffmpeg_path": "/usr/bin/ffmpeg",
		"ffprobe_path": "/usr/bin/ffprobe",
		"enable_gpu": true
	}
}`

const minimalConfigJSON = `{
	"vector": {
		"driver": "qdrant",
		"config": {}
	}
}`

func TestParseConfigFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid full config",
			json:    testConfigJSON,
			wantErr: false,
		},
		{
			name:    "valid minimal config",
			json:    minimalConfigJSON,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{"invalid": json}`,
			wantErr: true,
		},
		{
			name:    "empty json",
			json:    `{}`,
			wantErr: false, // Should parse but with empty fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseConfigFromJSON([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfigFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("ParseConfigFromJSON() returned nil config without error")
			}
		})
	}
}

func TestParseConfigFromFile(t *testing.T) {
	// Create temporary test file
	tmpFile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test config to file
	if _, err := tmpFile.WriteString(testConfigJSON); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	// Test parsing from file
	config, err := ParseConfigFromFile(tmpFile.Name())
	if err != nil {
		t.Errorf("ParseConfigFromFile() error = %v", err)
		return
	}
	if config == nil {
		t.Error("ParseConfigFromFile() returned nil config")
		return
	}

	// Verify basic fields
	if config.Vector.Driver != "qdrant" {
		t.Errorf("Expected vector driver 'qdrant', got '%s'", config.Vector.Driver)
	}
	if config.Store != "test_store" {
		t.Errorf("Expected store 'test_store', got '%s'", config.Store)
	}

	// Test non-existent file
	_, err = ParseConfigFromFile("non_existent_file.json")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestConfig_ToJSON(t *testing.T) {
	// Parse a config first
	config, err := ParseConfigFromJSON([]byte(testConfigJSON))
	if err != nil {
		t.Fatalf("Failed to parse test config: %v", err)
	}

	// Convert back to JSON
	jsonData, err := config.ToJSON()
	if err != nil {
		t.Errorf("ToJSON() error = %v", err)
		return
	}

	// Verify it's valid JSON
	var testObj map[string]interface{}
	if err := json.Unmarshal(jsonData, &testObj); err != nil {
		t.Errorf("ToJSON() produced invalid JSON: %v", err)
	}

	// Verify Features field is not included in JSON output
	jsonStr := string(jsonData)
	if strings.Contains(jsonStr, "features") || strings.Contains(jsonStr, "Features") {
		t.Error("ToJSON() should not include Features field")
	}
}

func TestConfig_UnmarshalJSON(t *testing.T) {
	var config Config
	err := json.Unmarshal([]byte(testConfigJSON), &config)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}

	// Verify basic fields
	if config.Vector.Driver != "qdrant" {
		t.Errorf("Expected vector driver 'qdrant', got '%s'", config.Vector.Driver)
	}

	// Verify that Features are not computed during UnmarshalJSON (they should be computed later)
	// Features will be computed after providers are loaded in the actual Load function

	// But we can manually compute features to test the logic
	config.Features = config.ComputeFeatures()

	// These should be true based on the config content (graph, pdf, ffmpeg are present)
	if !config.Features.GraphDatabase {
		t.Error("Expected GraphDatabase feature to be true")
	}
	if !config.Features.PDFProcessing {
		t.Error("Expected PDFProcessing feature to be true")
	}
	if !config.Features.VideoProcessing {
		t.Error("Expected VideoProcessing feature to be true")
	}
}

func TestConfig_MarshalJSON(t *testing.T) {
	config := &Config{
		Vector: VectorConfig{
			Driver: "qdrant",
			Config: map[string]interface{}{"host": "localhost"},
		},
		Store: "test_store",
		Features: Features{
			GraphDatabase: true, // This should not appear in JSON
		},
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
		return
	}

	// Verify Features field is not included
	jsonStr := string(jsonData)
	if strings.Contains(jsonStr, "features") || strings.Contains(jsonStr, "Features") {
		t.Error("MarshalJSON() should not include Features field")
	}

	// Verify other fields are included
	if !strings.Contains(jsonStr, "qdrant") {
		t.Error("MarshalJSON() should include vector driver")
	}
	if !strings.Contains(jsonStr, "test_store") {
		t.Error("MarshalJSON() should include store")
	}
}

func TestConfig_ComputeFeatures(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected Features
	}{
		{
			name: "full features config",
			config: &Config{
				Graph:  &GraphConfig{Driver: "neo4j"},
				PDF:    &PDFConfig{ConvertTool: "pdftoppm"},
				FFmpeg: &FFmpegConfig{FFmpegPath: "/usr/bin/ffmpeg"},
				Providers: &ProviderConfig{
					Converters: map[string][]*Provider{
						"en": {
							{ID: "__yao.office"},
							{ID: "__yao.ocr"},
							{ID: "__yao.whisper"},
							{ID: "__yao.vision"},
						},
					},
					Extractions: map[string][]*Provider{
						"en": {{ID: "test"}},
					},
					Fetchers: map[string][]*Provider{
						"en": {{ID: "test"}},
					},
					Searchers: map[string][]*Provider{
						"en": {{ID: "test"}},
					},
					Rerankers: map[string][]*Provider{
						"en": {{ID: "test"}},
					},
					Votes: map[string][]*Provider{
						"en": {{ID: "test"}},
					},
					Weights: map[string][]*Provider{
						"en": {{ID: "test"}},
					},
					Scores: map[string][]*Provider{
						"en": {{ID: "test"}},
					},
				},
			},
			expected: Features{
				GraphDatabase:    true,
				PDFProcessing:    true,
				VideoProcessing:  true,
				PlainText:        true,
				OfficeDocuments:  true,
				OCRProcessing:    true,
				AudioTranscript:  true,
				ImageAnalysis:    true,
				EntityExtraction: true,
				WebFetching:      true,
				CustomSearch:     true,
				ResultReranking:  true,
				SegmentVoting:    true,
				SegmentWeighting: true,
				SegmentScoring:   true,
			},
		},
		{
			name: "minimal config",
			config: &Config{
				Graph:     nil,
				PDF:       nil,
				FFmpeg:    nil,
				Providers: nil,
			},
			expected: Features{
				GraphDatabase:   false,
				PDFProcessing:   false,
				VideoProcessing: false,
				PlainText:       true, // Always supported
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ComputeFeatures()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ComputeFeatures() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Parse config from JSON
	originalConfig, err := ParseConfigFromJSON([]byte(testConfigJSON))
	if err != nil {
		t.Fatalf("Failed to parse original config: %v", err)
	}

	// Convert to JSON
	jsonData, err := originalConfig.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert config to JSON: %v", err)
	}

	// Parse again
	roundTripConfig, err := ParseConfigFromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse round-trip config: %v", err)
	}

	// Compare key fields (Features will be recomputed, so they should match)
	if originalConfig.Vector.Driver != roundTripConfig.Vector.Driver {
		t.Errorf("Vector driver mismatch: %s != %s", originalConfig.Vector.Driver, roundTripConfig.Vector.Driver)
	}
	if originalConfig.Store != roundTripConfig.Store {
		t.Errorf("Store mismatch: %s != %s", originalConfig.Store, roundTripConfig.Store)
	}
	if !reflect.DeepEqual(originalConfig.Features, roundTripConfig.Features) {
		t.Errorf("Features mismatch: %+v != %+v", originalConfig.Features, roundTripConfig.Features)
	}
}

func TestConfig_ResolveEnvVars(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_HOST", "localhost")
	os.Setenv("TEST_PORT", "6333")
	defer func() {
		os.Unsetenv("TEST_HOST")
		os.Unsetenv("TEST_PORT")
	}()

	config := &Config{}

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "simple environment variable",
			input: map[string]interface{}{
				"host": "$ENV.TEST_HOST",
				"port": "$ENV.TEST_PORT",
			},
			expected: map[string]interface{}{
				"host": "localhost",
				"port": "6333",
			},
		},
		{
			name: "mixed values",
			input: map[string]interface{}{
				"host":   "$ENV.TEST_HOST",
				"port":   6333,
				"prefix": "test-$ENV.TEST_HOST-suffix",
			},
			expected: map[string]interface{}{
				"host":   "localhost",
				"port":   6333,
				"prefix": "test-localhost-suffix",
			},
		},
		{
			name: "nested configuration",
			input: map[string]interface{}{
				"database": map[string]interface{}{
					"host": "$ENV.TEST_HOST",
					"port": "$ENV.TEST_PORT",
				},
				"name": "test",
			},
			expected: map[string]interface{}{
				"database": map[string]interface{}{
					"host": "localhost",
					"port": "6333",
				},
				"name": "test",
			},
		},
		{
			name: "undefined environment variable",
			input: map[string]interface{}{
				"host": "$ENV.UNDEFINED_VAR",
				"port": "$ENV.TEST_PORT",
			},
			expected: map[string]interface{}{
				"host": "$ENV.UNDEFINED_VAR", // Should remain unchanged
				"port": "6333",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.resolveEnvVars(tt.input)
			if err != nil {
				t.Errorf("resolveEnvVars() error = %v", err)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("resolveEnvVars() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestConfig_ParseEnvVar(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_HOST", "test_value")
	defer os.Unsetenv("TEST_HOST")

	config := &Config{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple env var",
			input:    "$ENV.TEST_HOST",
			expected: "test_value",
		},
		{
			name:     "env var in string",
			input:    "prefix-$ENV.TEST_HOST-suffix",
			expected: "prefix-test_value-suffix",
		},
		{
			name:     "multiple env vars",
			input:    "$ENV.TEST_HOST-$ENV.TEST_HOST",
			expected: "test_value-test_value",
		},
		{
			name:     "undefined env var",
			input:    "$ENV.UNDEFINED_VAR",
			expected: "$ENV.UNDEFINED_VAR", // Should remain unchanged
		},
		{
			name:     "no env var",
			input:    "plain_string",
			expected: "plain_string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.parseEnvVar(tt.input)
			if result != tt.expected {
				t.Errorf("parseEnvVar() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfig_ResolveEnvVarsOnParsing(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_VECTOR_HOST", "test-vector-host")
	os.Setenv("TEST_GRAPH_URL", "neo4j://test-graph:7687")
	os.Setenv("TEST_GRAPH_USER", "test-user")
	os.Setenv("TEST_GRAPH_PASS", "test-pass")
	defer func() {
		os.Unsetenv("TEST_VECTOR_HOST")
		os.Unsetenv("TEST_GRAPH_URL")
		os.Unsetenv("TEST_GRAPH_USER")
		os.Unsetenv("TEST_GRAPH_PASS")
	}()

	configJSON := `{
		"vector": {
			"driver": "qdrant",
			"config": {
				"host": "$ENV.TEST_VECTOR_HOST",
				"port": 6333
			}
		},
		"graph": {
			"driver": "neo4j",
			"config": {
				"url": "$ENV.TEST_GRAPH_URL",
				"username": "$ENV.TEST_GRAPH_USER",
				"password": "$ENV.TEST_GRAPH_PASS"
			}
		}
	}`

	// Parse config from JSON
	config, err := ParseConfigFromJSON([]byte(configJSON))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify that environment variables are resolved immediately after parsing
	if config.Vector.Config["host"] != "test-vector-host" {
		t.Errorf("Expected vector host to be resolved to 'test-vector-host', got '%v'", config.Vector.Config["host"])
	}

	if config.Graph.Config["url"] != "neo4j://test-graph:7687" {
		t.Errorf("Expected graph URL to be resolved to 'neo4j://test-graph:7687', got '%v'", config.Graph.Config["url"])
	}

	if config.Graph.Config["username"] != "test-user" {
		t.Errorf("Expected graph username to be resolved to 'test-user', got '%v'", config.Graph.Config["username"])
	}

	if config.Graph.Config["password"] != "test-pass" {
		t.Errorf("Expected graph password to be resolved to 'test-pass', got '%v'", config.Graph.Config["password"])
	}

	// Verify that numeric values remain unchanged (JSON numbers are parsed as float64)
	if port, ok := config.Vector.Config["port"].(float64); !ok || port != 6333.0 {
		t.Errorf("Expected vector port to remain 6333.0, got %v (type %T)", config.Vector.Config["port"], config.Vector.Config["port"])
	}
}

func TestProviderConfig_GetProviders(t *testing.T) {
	// Create test provider config
	providerConfig := &ProviderConfig{
		Chunkings: map[string][]*Provider{
			"en": {
				{ID: "__yao.structured", Label: "Document Structure", Description: "Split by structure"},
				{ID: "__yao.semantic", Label: "Semantic Split", Description: "AI-powered splitting"},
			},
			"zh-cn": {
				{ID: "__yao.structured", Label: "文档结构", Description: "按结构分割"},
			},
		},
		Embeddings: map[string][]*Provider{
			"en": {
				{ID: "__yao.openai", Label: "OpenAI", Description: "OpenAI embeddings"},
			},
		},
	}

	tests := []struct {
		name         string
		providerType string
		language     string
		expectedLen  int
		expectedIDs  []string
	}{
		{
			name:         "get chunking providers for en",
			providerType: "chunking",
			language:     "en",
			expectedLen:  2,
			expectedIDs:  []string{"__yao.structured", "__yao.semantic"},
		},
		{
			name:         "get chunking providers for zh-cn",
			providerType: "chunking",
			language:     "zh-cn",
			expectedLen:  1,
			expectedIDs:  []string{"__yao.structured"},
		},
		{
			name:         "get embedding providers for en",
			providerType: "embedding",
			language:     "en",
			expectedLen:  1,
			expectedIDs:  []string{"__yao.openai"},
		},
		{
			name:         "fallback to en when language not found",
			providerType: "embedding",
			language:     "fr", // Not available, should fallback to en
			expectedLen:  1,
			expectedIDs:  []string{"__yao.openai"},
		},
		{
			name:         "return empty when provider type not found",
			providerType: "nonexistent",
			language:     "en",
			expectedLen:  0,
			expectedIDs:  []string{},
		},
		{
			name:         "return empty when no providers for language",
			providerType: "converter", // Empty in test config
			language:     "en",
			expectedLen:  0,
			expectedIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers := providerConfig.GetProviders(tt.providerType, tt.language)

			if len(providers) != tt.expectedLen {
				t.Errorf("Expected %d providers, got %d", tt.expectedLen, len(providers))
				return
			}

			// Check provider IDs
			actualIDs := make([]string, len(providers))
			for i, provider := range providers {
				actualIDs[i] = provider.ID
			}

			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, actualID := range actualIDs {
					if actualID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected provider ID '%s' not found in results: %v", expectedID, actualIDs)
				}
			}
		})
	}
}

func TestProviderConfig_GetProvider(t *testing.T) {
	// Create test provider config
	providerConfig := &ProviderConfig{
		Chunkings: map[string][]*Provider{
			"en": {
				{ID: "__yao.structured", Label: "Document Structure", Description: "Split by structure"},
				{ID: "__yao.semantic", Label: "Semantic Split", Description: "AI-powered splitting"},
			},
			"zh-cn": {
				{ID: "__yao.structured", Label: "文档结构", Description: "按结构分割"},
			},
		},
	}

	tests := []struct {
		name         string
		providerType string
		providerID   string
		language     string
		expectError  bool
		expectedID   string
	}{
		{
			name:         "get existing provider in requested language",
			providerType: "chunking",
			providerID:   "__yao.structured",
			language:     "en",
			expectError:  false,
			expectedID:   "__yao.structured",
		},
		{
			name:         "get provider with language fallback",
			providerType: "chunking",
			providerID:   "__yao.semantic", // Only exists in "en"
			language:     "fr",             // Should fallback to "en"
			expectError:  false,
			expectedID:   "__yao.semantic",
		},
		{
			name:         "provider not found",
			providerType: "chunking",
			providerID:   "__yao.nonexistent",
			language:     "en",
			expectError:  true,
		},
		{
			name:         "invalid provider type",
			providerType: "invalid",
			providerID:   "__yao.structured",
			language:     "en",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := providerConfig.GetProvider(tt.providerType, tt.providerID, tt.language)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("Expected provider, got nil")
				return
			}

			if provider.ID != tt.expectedID {
				t.Errorf("Expected provider ID '%s', got '%s'", tt.expectedID, provider.ID)
			}
		})
	}
}
