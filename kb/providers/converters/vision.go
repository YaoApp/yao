package converters

import (
	"strings"

	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Vision is a converter provider for vision files
type Vision struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// Make creates a new Vision converter
func (vision *Vision) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// Start with default values
	visionOption := converter.VisionOption{
		ConnectorName: "",     // Will be set from option
		Model:         "",     // Will use default from connector
		Prompt:        "",     // Will use default
		CompressSize:  512,    // Default compression size
		Language:      "Auto", // Default language
		Options:       nil,    // Additional options
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		if connector, ok := option.Properties["connector"]; ok {
			if connectorStr, ok := connector.(string); ok {
				visionOption.ConnectorName = connectorStr
			}
		}

		if model, ok := option.Properties["model"]; ok {
			if modelStr, ok := model.(string); ok {
				visionOption.Model = modelStr
			}
		}

		if prompt, ok := option.Properties["prompt"]; ok {
			if promptStr, ok := prompt.(string); ok {
				visionOption.Prompt = promptStr
			}
		}

		if compressSize, ok := option.Properties["compress_size"]; ok {
			if sizeInt, ok := compressSize.(int); ok {
				visionOption.CompressSize = int64(sizeInt)
			} else if sizeFloat, ok := compressSize.(float64); ok {
				visionOption.CompressSize = int64(sizeFloat)
			}
		}

		if language, ok := option.Properties["language"]; ok {
			if langStr, ok := language.(string); ok {
				visionOption.Language = langStr
			}
		}

		if options, ok := option.Properties["options"]; ok {
			if optionsMap, ok := options.(map[string]interface{}); ok {
				visionOption.Options = optionsMap
			}
		}
	}

	return converter.NewVision(visionOption)
}

// AutoDetect detects the converter based on the filename and content types
func (vision *Vision) AutoDetect(filename, contentTypes string) (bool, int, error) {
	// If autodetect is empty, return false
	if vision.Autodetect == nil {
		return false, 0, nil
	}

	// Check if the filename matches the autodetect
	for _, autodetect := range vision.Autodetect {
		if strings.HasSuffix(filename, autodetect) {
			return true, vision.MatchPriority, nil
		}

		// Check if the content types matches the autodetect
		if strings.Contains(contentTypes, autodetect) {
			return true, vision.MatchPriority, nil
		}
	}

	return false, 0, nil
}

// Schema returns the schema for the Vision converter
func (vision *Vision) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeConverter, "vision", locale)
}
