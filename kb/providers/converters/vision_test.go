package converters

import (
	"testing"

	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestVision_Make(t *testing.T) {
	vision := &Vision{}

	// Note: Vision converter requires connectors to be loaded
	// All tests will fail in test environment due to missing connectors

	t.Run("nil option should return error due to missing connector", func(t *testing.T) {
		_, err := vision.Make(nil)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("empty option should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		_, err := vision.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("option with non-existent connector should return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "non-existent.connector",
			},
		}
		_, err := vision.Make(option)
		if err == nil {
			t.Error("Expected error for non-existent connector")
		}
		// Error is expected because non-existent.connector is not loaded
	})

	t.Run("option with all properties but non-existent connector should return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":     "fake.vision.connector",
				"model":         "gpt-4o",
				"prompt":        "Describe this image",
				"compress_size": 1024,
				"language":      "English",
				"options": map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  500,
				},
			},
		}
		_, err := vision.Make(option)
		if err == nil {
			t.Error("Expected error for non-existent connector")
		}
		// Error is expected because fake.vision.connector is not loaded
	})

	t.Run("compress_size as float64 should be converted to int64 but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":     "invalid.test.connector",
				"compress_size": 512.0, // float64
			},
		}
		_, err := vision.Make(option)
		if err == nil {
			t.Error("Expected error for invalid connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("invalid property types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":     123,       // invalid type
				"compress_size": "invalid", // invalid type
				"language":      true,      // invalid type
			},
		}
		_, err := vision.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("missing connector should still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"model":         "gpt-4o-mini",
				"compress_size": 256,
			},
		}
		_, err := vision.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because no connector is specified
	})
}

func TestVision_AutoDetect(t *testing.T) {
	vision := &Vision{
		Autodetect:    []string{".jpg", ".png", ".gif", "image/jpeg", "image/png"},
		MatchPriority: 20,
	}

	t.Run("should detect .jpg files", func(t *testing.T) {
		match, priority, err := vision.AutoDetect("photo.jpg", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .jpg file")
		}
		if priority != 20 {
			t.Errorf("Expected priority 20, got %d", priority)
		}
	})

	t.Run("should detect .png files", func(t *testing.T) {
		match, priority, err := vision.AutoDetect("image.png", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .png file")
		}
		if priority != 20 {
			t.Errorf("Expected priority 20, got %d", priority)
		}
	})

	t.Run("should detect by content type", func(t *testing.T) {
		match, priority, err := vision.AutoDetect("unknown", "image/jpeg")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for image/jpeg content type")
		}
		if priority != 20 {
			t.Errorf("Expected priority 20, got %d", priority)
		}
	})

	t.Run("should not detect unsupported files", func(t *testing.T) {
		match, priority, err := vision.AutoDetect("document.pdf", "application/pdf")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if match {
			t.Error("Expected no match for .pdf file")
		}
		if priority != 0 {
			t.Errorf("Expected priority 0, got %d", priority)
		}
	})

	t.Run("empty autodetect should not match", func(t *testing.T) {
		emptyVision := &Vision{}
		match, priority, err := emptyVision.AutoDetect("image.jpg", "image/jpeg")
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

func TestVision_Schema(t *testing.T) {
	vision := &Vision{}
	schema, err := vision.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}
