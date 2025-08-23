package converters

import (
	"strings"

	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Whisper is a converter provider for audio files
type Whisper struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// Make creates a new Whisper converter
func (whisper *Whisper) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// Start with default values
	whisperOption := converter.WhisperOption{
		ConnectorName:          "",    // Will be set from option
		Model:                  "",    // Will use default from connector
		Language:               "",    // Auto-detect
		ChunkDuration:          30.0,  // Default 30 seconds
		MappingDuration:        5.0,   // Default 5 seconds
		SilenceThreshold:       -40.0, // Default -40dB
		SilenceMinLength:       1.0,   // Default 1 second
		EnableSilenceDetection: true,  // Default enabled
		MaxConcurrency:         4,     // Default 4 concurrent requests
		TempDir:                "",    // Will use system temp
		CleanupTemp:            true,  // Default cleanup
		Options:                nil,   // Additional options
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		if connector, ok := option.Properties["connector"]; ok {
			if connectorStr, ok := connector.(string); ok {
				whisperOption.ConnectorName = connectorStr
			}
		}

		if model, ok := option.Properties["model"]; ok {
			if modelStr, ok := model.(string); ok {
				whisperOption.Model = modelStr
			}
		}

		if language, ok := option.Properties["language"]; ok {
			if langStr, ok := language.(string); ok {
				whisperOption.Language = langStr
			}
		}

		if chunkDuration, ok := option.Properties["chunk_duration"]; ok {
			if durationFloat, ok := chunkDuration.(float64); ok {
				whisperOption.ChunkDuration = durationFloat
			} else if durationInt, ok := chunkDuration.(int); ok {
				whisperOption.ChunkDuration = float64(durationInt)
			}
		}

		if mappingDuration, ok := option.Properties["mapping_duration"]; ok {
			if durationFloat, ok := mappingDuration.(float64); ok {
				whisperOption.MappingDuration = durationFloat
			} else if durationInt, ok := mappingDuration.(int); ok {
				whisperOption.MappingDuration = float64(durationInt)
			}
		}

		if silenceThreshold, ok := option.Properties["silence_threshold"]; ok {
			if thresholdFloat, ok := silenceThreshold.(float64); ok {
				whisperOption.SilenceThreshold = thresholdFloat
			} else if thresholdInt, ok := silenceThreshold.(int); ok {
				whisperOption.SilenceThreshold = float64(thresholdInt)
			}
		}

		if silenceMinLength, ok := option.Properties["silence_min_length"]; ok {
			if lengthFloat, ok := silenceMinLength.(float64); ok {
				whisperOption.SilenceMinLength = lengthFloat
			} else if lengthInt, ok := silenceMinLength.(int); ok {
				whisperOption.SilenceMinLength = float64(lengthInt)
			}
		}

		if enableSilence, ok := option.Properties["enable_silence_detection"]; ok {
			if enableBool, ok := enableSilence.(bool); ok {
				whisperOption.EnableSilenceDetection = enableBool
			}
		}

		if maxConcurrency, ok := option.Properties["max_concurrency"]; ok {
			if maxInt, ok := maxConcurrency.(int); ok {
				whisperOption.MaxConcurrency = maxInt
			} else if maxFloat, ok := maxConcurrency.(float64); ok {
				whisperOption.MaxConcurrency = int(maxFloat)
			}
		}

		if tempDir, ok := option.Properties["temp_dir"]; ok {
			if tempDirStr, ok := tempDir.(string); ok {
				whisperOption.TempDir = tempDirStr
			}
		}

		if cleanupTemp, ok := option.Properties["cleanup_temp"]; ok {
			if cleanupBool, ok := cleanupTemp.(bool); ok {
				whisperOption.CleanupTemp = cleanupBool
			}
		}

		if options, ok := option.Properties["options"]; ok {
			if optionsMap, ok := options.(map[string]interface{}); ok {
				whisperOption.Options = optionsMap
			}
		}
	}

	return converter.NewWhisper(whisperOption)
}

// AutoDetect detects the converter based on the filename and content types
func (whisper *Whisper) AutoDetect(filename, contentTypes string) (bool, int, error) {
	// If autodetect is empty, return false
	if whisper.Autodetect == nil {
		return false, 0, nil
	}

	// Check if the filename matches the autodetect
	for _, autodetect := range whisper.Autodetect {
		if strings.HasSuffix(filename, autodetect) {
			return true, whisper.MatchPriority, nil
		}

		// Check if the content types matches the autodetect
		if strings.Contains(contentTypes, autodetect) {
			return true, whisper.MatchPriority, nil
		}
	}

	return false, 0, nil
}

// Schema returns the schema for the Whisper converter
func (whisper *Whisper) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeConverter, "whisper", locale)
}
