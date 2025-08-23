package converters

import (
	"testing"

	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestUTF8_Make(t *testing.T) {
	utf8 := &UTF8{}

	t.Run("nil option should create UTF8 converter", func(t *testing.T) {
		converter, err := utf8.Make(nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if converter == nil {
			t.Fatal("Expected converter, got nil")
		}
	})

	t.Run("empty option should create UTF8 converter", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		converter, err := utf8.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if converter == nil {
			t.Fatal("Expected converter, got nil")
		}
	})

	t.Run("option with properties should create UTF8 converter", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"some_property": "some_value",
			},
		}
		converter, err := utf8.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if converter == nil {
			t.Fatal("Expected converter, got nil")
		}
	})
}

func TestUTF8_AutoDetect(t *testing.T) {
	utf8 := &UTF8{
		Autodetect:    []string{".txt", ".md", "text/plain"},
		MatchPriority: 100,
	}

	t.Run("should detect .txt files", func(t *testing.T) {
		match, priority, err := utf8.AutoDetect("test.txt", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .txt file")
		}
		if priority != 100 {
			t.Errorf("Expected priority 100, got %d", priority)
		}
	})

	t.Run("should detect .md files", func(t *testing.T) {
		match, priority, err := utf8.AutoDetect("readme.md", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .md file")
		}
		if priority != 100 {
			t.Errorf("Expected priority 100, got %d", priority)
		}
	})

	t.Run("should detect by content type", func(t *testing.T) {
		match, priority, err := utf8.AutoDetect("unknown", "text/plain")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for text/plain content type")
		}
		if priority != 100 {
			t.Errorf("Expected priority 100, got %d", priority)
		}
	})

	t.Run("should not detect unsupported files", func(t *testing.T) {
		match, priority, err := utf8.AutoDetect("test.pdf", "application/pdf")
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
		emptyUTF8 := &UTF8{}
		match, priority, err := emptyUTF8.AutoDetect("test.txt", "text/plain")
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

func TestUTF8_Schema(t *testing.T) {
	utf8 := &UTF8{}
	schema, err := utf8.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}
