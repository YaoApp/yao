package converters

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Office is a converter provider for office files, support docx, pptx.
type Office struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// Make creates a new Office converter
func (office *Office) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// Start with default values
	officeOption := converter.OfficeOption{
		VisionConverter:  nil,  // Will be set from option
		VideoConverter:   nil,  // Optional, will be set from option if provided
		WhisperConverter: nil,  // Optional, will be set from option if provided
		MaxConcurrency:   4,    // Default 4 concurrent processes
		TempDir:          "",   // Use system temp
		CleanupTemp:      true, // Default cleanup
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		if maxConcurrency, ok := option.Properties["max_concurrency"]; ok {
			if maxInt, ok := maxConcurrency.(int); ok {
				officeOption.MaxConcurrency = maxInt
			} else if maxFloat, ok := maxConcurrency.(float64); ok {
				officeOption.MaxConcurrency = int(maxFloat)
			}
		}

		if tempDir, ok := option.Properties["temp_dir"]; ok {
			if tempDirStr, ok := tempDir.(string); ok {
				officeOption.TempDir = tempDirStr
			}
		}

		if cleanupTemp, ok := option.Properties["cleanup_temp"]; ok {
			if cleanupBool, ok := cleanupTemp.(bool); ok {
				officeOption.CleanupTemp = cleanupBool
			}
		}

		// Handle nested vision converter (required)
		if vision, ok := option.Properties["vision"]; ok {
			visionConverter, err := parseNestedConverter(vision)
			if err != nil {
				return nil, fmt.Errorf("failed to parse vision converter: %w", err)
			}
			officeOption.VisionConverter = visionConverter
		}

		// Handle nested video converter (optional)
		if video, ok := option.Properties["video"]; ok {
			videoConverter, err := parseNestedConverter(video)
			if err != nil {
				return nil, fmt.Errorf("failed to parse video converter: %w", err)
			}
			officeOption.VideoConverter = videoConverter
		}

		// Handle nested audio/whisper converter (optional)
		if audio, ok := option.Properties["audio"]; ok {
			audioConverter, err := parseNestedConverter(audio)
			if err != nil {
				return nil, fmt.Errorf("failed to parse audio converter: %w", err)
			}
			officeOption.WhisperConverter = audioConverter
		}
	}

	// Vision converter is required for office processing
	if officeOption.VisionConverter == nil {
		return nil, fmt.Errorf("vision converter is required for office document processing")
	}

	return converter.NewOffice(officeOption)
}

// AutoDetect detects the converter based on the filename and content types
func (office *Office) AutoDetect(filename, contentTypes string) (bool, int, error) {
	// If autodetect is empty, return false
	if office.Autodetect == nil {
		return false, 0, nil
	}

	// Check if the filename matches the autodetect
	for _, autodetect := range office.Autodetect {
		if strings.HasSuffix(filename, autodetect) {
			return true, office.MatchPriority, nil
		}

		// Check if the content types matches the autodetect
		if strings.Contains(contentTypes, autodetect) {
			return true, office.MatchPriority, nil
		}
	}

	return false, 0, nil
}

// Schema returns the schema for the Office converter
func (office *Office) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeConverter, "office", locale)
}
