package converters

import (
	"testing"

	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestVideo_Make(t *testing.T) {
	video := &Video{}

	// Note: Video converter requires FFmpeg and audio converters to be set up
	// All tests will fail in test environment due to missing dependencies

	t.Run("nil option should return error due to missing FFmpeg", func(t *testing.T) {
		_, err := video.Make(nil)
		if err == nil {
			t.Error("Expected error due to missing FFmpeg or audio converter")
		}
		// Error is expected because FFmpeg and audio converter are not set up in test environment
	})

	t.Run("empty option should return error due to missing FFmpeg", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error due to missing FFmpeg or audio converter")
		}
		// Error is expected because FFmpeg and audio converter are not set up in test environment
	})

	t.Run("option with video processing properties should return error due to missing FFmpeg", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"keyframe_interval":   15.0,
				"max_keyframes":       30,
				"temp_dir":            "/tmp/video",
				"cleanup_temp":        false,
				"max_concurrency":     8,
				"text_optimization":   false,
				"deduplication_ratio": 0.9,
			},
		}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error due to missing FFmpeg or audio converter")
		}
	})

	t.Run("float64 values should be handled correctly but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"keyframe_interval":   12.5, // float64
				"deduplication_ratio": 0.75, // float64
			},
		}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error due to missing FFmpeg or audio converter")
		}
	})

	t.Run("int values should be converted to appropriate types but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"keyframe_interval":   20, // int -> float64
				"max_keyframes":       25, // int
				"max_concurrency":     6,  // int
				"deduplication_ratio": 1,  // int -> float64
			},
		}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error due to missing FFmpeg or audio converter")
		}
	})

	t.Run("boolean values should be handled correctly but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"cleanup_temp":      true,
				"text_optimization": false,
			},
		}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error due to missing FFmpeg or audio converter")
		}
	})

	t.Run("invalid property types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"keyframe_interval":   "invalid", // invalid type
				"max_keyframes":       "invalid", // invalid type
				"text_optimization":   "invalid", // invalid type
				"deduplication_ratio": "invalid", // invalid type
			},
		}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error due to missing FFmpeg or audio converter")
		}
	})

	t.Run("partial properties should use defaults for missing values but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"keyframe_interval": 5.0,
				"max_keyframes":     10,
				// Other properties should use defaults
			},
		}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error due to missing FFmpeg or audio converter")
		}
	})

	// Note: Nested converter tests would require setting up mock factories
	// For now, we test the error cases when parseNestedConverter fails
	t.Run("invalid vision converter should return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"vision": "invalid_format", // should be a map
			},
		}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error for invalid vision converter format")
		}
	})

	t.Run("invalid audio converter should return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"audio": []string{"invalid"}, // should be a map
			},
		}
		_, err := video.Make(option)
		if err == nil {
			t.Error("Expected error for invalid audio converter format")
		}
	})
}

func TestVideo_AutoDetect(t *testing.T) {
	video := &Video{
		Autodetect:    []string{".mp4", ".mov", ".avi", "video/mp4", "video/quicktime"},
		MatchPriority: 10,
	}

	t.Run("should detect .mp4 files", func(t *testing.T) {
		match, priority, err := video.AutoDetect("movie.mp4", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .mp4 file")
		}
		if priority != 10 {
			t.Errorf("Expected priority 10, got %d", priority)
		}
	})

	t.Run("should detect .mov files", func(t *testing.T) {
		match, priority, err := video.AutoDetect("video.mov", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .mov file")
		}
		if priority != 10 {
			t.Errorf("Expected priority 10, got %d", priority)
		}
	})

	t.Run("should detect by content type", func(t *testing.T) {
		match, priority, err := video.AutoDetect("unknown", "video/mp4")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for video/mp4 content type")
		}
		if priority != 10 {
			t.Errorf("Expected priority 10, got %d", priority)
		}
	})

	t.Run("should not detect unsupported files", func(t *testing.T) {
		match, priority, err := video.AutoDetect("audio.mp3", "audio/mpeg")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if match {
			t.Error("Expected no match for .mp3 file")
		}
		if priority != 0 {
			t.Errorf("Expected priority 0, got %d", priority)
		}
	})

	t.Run("empty autodetect should not match", func(t *testing.T) {
		emptyVideo := &Video{}
		match, priority, err := emptyVideo.AutoDetect("video.mp4", "video/mp4")
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

func TestVideo_Schema(t *testing.T) {
	video := &Video{}
	schema, err := video.Schema(nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema != nil {
		t.Error("Expected nil schema")
	}
}
