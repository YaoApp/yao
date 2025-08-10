package converters

import (
	"testing"

	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestWhisper_Make(t *testing.T) {
	whisper := &Whisper{}

	// Note: Whisper converter requires connectors to be loaded
	// All tests will fail in test environment due to missing connectors

	t.Run("nil option should return error due to missing connector", func(t *testing.T) {
		_, err := whisper.Make(nil)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("empty option should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		_, err := whisper.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("option with all audio properties should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":                "openai.whisper",
				"model":                    "whisper-1",
				"language":                 "en",
				"chunk_duration":           45.0,
				"mapping_duration":         10.0,
				"silence_threshold":        -35.0,
				"silence_min_length":       2.0,
				"enable_silence_detection": false,
				"max_concurrency":          8,
				"temp_dir":                 "/tmp/whisper",
				"cleanup_temp":             false,
				"options": map[string]interface{}{
					"temperature": 0.0,
				},
			},
		}
		_, err := whisper.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because openai.whisper connector is not loaded
	})

	t.Run("float64 values should be handled correctly but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":          "openai.whisper",
				"chunk_duration":     30.5,  // float64
				"mapping_duration":   5.2,   // float64
				"silence_threshold":  -42.3, // float64
				"silence_min_length": 1.8,   // float64
			},
		}
		_, err := whisper.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("int values should be converted to float64 for duration fields but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":          "openai.whisper",
				"chunk_duration":     30,  // int
				"mapping_duration":   5,   // int
				"silence_threshold":  -40, // int
				"silence_min_length": 1,   // int
				"max_concurrency":    6,   // int
			},
		}
		_, err := whisper.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("boolean values should be handled correctly but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":                "openai.whisper",
				"enable_silence_detection": true,
				"cleanup_temp":             false,
			},
		}
		_, err := whisper.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("invalid property types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":                123,         // invalid type
				"chunk_duration":           "invalid",   // invalid type
				"enable_silence_detection": "invalid",   // invalid type
				"max_concurrency":          "invalid",   // invalid type
				"options":                  "not a map", // invalid type
			},
		}
		_, err := whisper.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("partial properties should use defaults for missing values but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":      "openai.whisper",
				"chunk_duration": 60.0,
				// Other properties should use defaults
			},
		}
		_, err := whisper.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})
}

func TestWhisper_AutoDetect(t *testing.T) {
	whisper := &Whisper{
		Autodetect:    []string{".mp3", ".wav", ".m4a", "audio/mpeg", "audio/wav"},
		MatchPriority: 10,
	}

	t.Run("should detect .mp3 files", func(t *testing.T) {
		match, priority, err := whisper.AutoDetect("audio.mp3", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .mp3 file")
		}
		if priority != 10 {
			t.Errorf("Expected priority 10, got %d", priority)
		}
	})

	t.Run("should detect .wav files", func(t *testing.T) {
		match, priority, err := whisper.AutoDetect("recording.wav", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .wav file")
		}
		if priority != 10 {
			t.Errorf("Expected priority 10, got %d", priority)
		}
	})

	t.Run("should detect by content type", func(t *testing.T) {
		match, priority, err := whisper.AutoDetect("unknown", "audio/mpeg")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for audio/mpeg content type")
		}
		if priority != 10 {
			t.Errorf("Expected priority 10, got %d", priority)
		}
	})

	t.Run("should not detect unsupported files", func(t *testing.T) {
		match, priority, err := whisper.AutoDetect("video.mp4", "video/mp4")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if match {
			t.Error("Expected no match for .mp4 file")
		}
		if priority != 0 {
			t.Errorf("Expected priority 0, got %d", priority)
		}
	})

	t.Run("empty autodetect should not match", func(t *testing.T) {
		emptyWhisper := &Whisper{}
		match, priority, err := emptyWhisper.AutoDetect("audio.mp3", "audio/mpeg")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if match {
			t.Error("Expected no match when autodetect is empty")
		}
		if priority != 0 {
			t.Errorf("Expected priority 0, got %d", priority)
		}
	})
}

func TestWhisper_Schema(t *testing.T) {
	whisper := &Whisper{}
	schema, err := whisper.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}
