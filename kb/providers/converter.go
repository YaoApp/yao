package providers

import (
	"strings"

	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Converter is a base converter provider
type Converter struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// UTF8 is a converter provider for utf8 files
type UTF8 struct{ Converter }

// Office is a converter provider for office files, support docx, pptx.
type Office struct{ Converter }

// OCR is a converter provider for ocr files, support pdf, image.
type OCR struct{ Converter }

// Video is a converter provider for video files
type Video struct{ Converter }

// Whisper is a converter provider for audio files
type Whisper struct{ Converter }

// Vision is a converter provider for vision files
type Vision struct{ Converter }

// MCP is a converter provider for mcp files
type MCP struct{ Converter }

// AutoRegister registers the converter providers
func init() {
	factory.Converters["__yao.utf8"] = &UTF8{
		Converter: Converter{
			Autodetect:    []string{"text/plain", "text/markdown", ".txt", ".md"},
			MatchPriority: 100,
		},
	}
	factory.Converters["__yao.office"] = &Office{
		Converter: Converter{
			Autodetect:    []string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.presentationml.presentation", ".docx", ".pptx"},
			MatchPriority: 10,
		},
	}
	factory.Converters["__yao.ocr"] = &OCR{
		Converter: Converter{
			Autodetect:    []string{"application/pdf", "image/jpeg", "image/png", "image/gif", "image/webp", ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".webp"},
			MatchPriority: 10,
		},
	}

	factory.Converters["__yao.video"] = &Video{
		Converter: Converter{
			Autodetect:    []string{"video/mp4", "video/mpeg", "video/quicktime", "video/webm", ".mp4", ".mpeg", ".mov", ".webm"},
			MatchPriority: 10,
		},
	}

	factory.Converters["__yao.whisper"] = &Whisper{
		Converter: Converter{
			Autodetect:    []string{"audio/mpeg", "audio/wav", "audio/webm", ".mp3", ".wav", ".webm"},
			MatchPriority: 10,
		},
	}

	factory.Converters["__yao.vision"] = &Vision{
		Converter: Converter{
			Autodetect:    []string{"image/jpeg", "image/png", "image/gif", "image/webp", ".jpg", ".jpeg", ".png", ".gif", ".webp"},
			MatchPriority: 20,
		},
	}

	factory.Converters["__yao.mcp"] = &MCP{Converter: Converter{}}

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

// === UTF8 ===

// Make creates a new UTF8 converter
func (utf8 *UTF8) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	return converter.NewUTF8(), nil
}

// Schema returns the schema for the UTF8 converter
func (utf8 *UTF8) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === Office ===

// Make creates a new Office converter
func (office *Office) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// TODO: Map kbtypes.ProviderOption to converter.OfficeOption
	officeOption := converter.OfficeOption{
		// VisionConverter:  nil, // TODO: Get vision converter from option
		// VideoConverter:   nil, // TODO: Get video converter from option
		// WhisperConverter: nil, // TODO: Get whisper converter from option
		// MaxConcurrency:   0,   // Will use default
		// TempDir:          "",  // Will use default
		// CleanupTemp:      false,
	}
	return converter.NewOffice(officeOption)
}

// Schema returns the schema for the Office converter
func (office *Office) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === OCR ===

// Make creates a new OCR converter
func (ocr *OCR) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// TODO: Map kbtypes.ProviderOption to converter.OCROption
	ocrOption := converter.OCROption{
		// Vision:         nil, // TODO: Get vision converter from option
		// Mode:           "", // Will use default
		// MaxConcurrency: 0,  // Will use default
		// CompressSize:   0,  // Will use default
		// ForceImageMode: false,
	}
	return converter.NewOCR(ocrOption)
}

// Schema returns the schema for the OCR converter
func (ocr *OCR) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === Video ===

// Make creates a new Video converter
func (video *Video) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// TODO: Map kbtypes.ProviderOption to converter.VideoOption
	videoOption := converter.VideoOption{
		// AudioConverter:     nil, // TODO: Get audio converter from option
		// VisionConverter:    nil, // TODO: Get vision converter from option
		// KeyframeInterval:   0,   // Will use default
		// MaxKeyframes:       0,   // Will use default
		// TempDir:            "",  // Will use default
		// CleanupTemp:        false,
		// MaxConcurrency:     0,   // Will use default
		// TextOptimization:   false,
		// DeduplicationRatio: 0,   // Will use default
	}
	return converter.NewVideo(videoOption)
}

// Schema returns the schema for the Video converter
func (video *Video) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === Whisper ===

// Make creates a new Whisper converter
func (whisper *Whisper) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// TODO: Map kbtypes.ProviderOption to converter.WhisperOption
	whisperOption := converter.WhisperOption{
		// ConnectorName:          "", // TODO: Get connector name from option
		// Model:                  "", // Will use default
		// Options:                nil,
		// Language:               "", // Will use default
		// ChunkDuration:          0,  // Will use default
		// MappingDuration:        0,  // Will use default
		// SilenceThreshold:       0,  // Will use default
		// SilenceMinLength:       0,  // Will use default
		// EnableSilenceDetection: false,
		// MaxConcurrency:         0,  // Will use default
		// TempDir:                "", // Will use default
		// CleanupTemp:            false,
	}
	return converter.NewWhisper(whisperOption)
}

// Schema returns the schema for the Whisper converter
func (whisper *Whisper) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === Vision ===

// Make creates a new Vision converter
func (vision *Vision) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// TODO: Map kbtypes.ProviderOption to converter.VisionOption
	visionOption := converter.VisionOption{
		// ConnectorName: "", // TODO: Get connector name from option
		// Model:         "", // Will use default
		// Prompt:        "", // Will use default
		// Options:       nil,
		// CompressSize:  0,  // Will use default
		// Language:      "", // Will use default
	}
	return converter.NewVision(visionOption)
}

// Schema returns the schema for the Vision converter
func (vision *Vision) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}

// === MCP ===

// Make creates a new MCP converter
func (mcp *MCP) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// TODO: Map kbtypes.ProviderOption to converter.MCPOptions
	mcpOptions := &converter.MCPOptions{
		// ID:                  "", // TODO: Get ID from option
		// Tool:                "", // TODO: Get tool from option
		// ArgumentsMapping:    nil, // TODO: Get arguments mapping from option
		// ResultMapping:       nil, // TODO: Get result mapping from option
		// NotificationMapping: nil, // TODO: Get notification mapping from option
	}
	return converter.NewMCP(mcpOptions)
}

// Schema returns the schema for the MCP converter
func (mcp *MCP) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}
