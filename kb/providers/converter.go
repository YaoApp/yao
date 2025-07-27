package providers

import (
	"strings"

	"github.com/yaoapp/yao/kb/providers/converters"
	"github.com/yaoapp/yao/kb/providers/factory"
)

// Converter is a base converter provider
type Converter struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// AutoDetect detects the converter based on the filename and content types
func (c Converter) AutoDetect(filename, contentTypes string) (bool, int, error) {

	// If autodetect is empty, return false
	if c.Autodetect == nil {
		return false, 0, nil
	}

	// Check if the filename matches the autodetect
	for _, autodetect := range c.Autodetect {
		if strings.HasSuffix(filename, autodetect) {
			return true, c.MatchPriority, nil
		}

		// Check if the content types matches the autodetect
		if strings.Contains(contentTypes, autodetect) {
			return true, c.MatchPriority, nil
		}
	}

	return false, 0, nil
}

// AutoRegister registers the converter providers
func init() {
	factory.Converters["__yao.utf8"] = &converters.UTF8{
		Autodetect:    []string{"text/plain", "text/markdown", ".txt", ".md"},
		MatchPriority: 100,
	}
	factory.Converters["__yao.office"] = &converters.Office{
		Autodetect:    []string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.presentationml.presentation", ".docx", ".pptx"},
		MatchPriority: 10,
	}
	factory.Converters["__yao.ocr"] = &converters.OCR{
		Autodetect:    []string{"application/pdf", "image/jpeg", "image/png", "image/gif", "image/webp", ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".webp"},
		MatchPriority: 10,
	}

	factory.Converters["__yao.video"] = &converters.Video{
		Autodetect:    []string{"video/mp4", "video/mpeg", "video/quicktime", "video/webm", ".mp4", ".mpeg", ".mov", ".webm"},
		MatchPriority: 10,
	}

	factory.Converters["__yao.whisper"] = &converters.Whisper{
		Autodetect:    []string{"audio/mpeg", "audio/wav", "audio/webm", ".mp3", ".wav", ".webm"},
		MatchPriority: 10,
	}

	factory.Converters["__yao.vision"] = &converters.Vision{
		Autodetect:    []string{"image/jpeg", "image/png", "image/gif", "image/webp", ".jpg", ".jpeg", ".png", ".gif", ".webp"},
		MatchPriority: 20,
	}

	factory.Converters["__yao.mcp"] = &converters.MCP{}

}
