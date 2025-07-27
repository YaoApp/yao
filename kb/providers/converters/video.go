package converters

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/types"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// Video is a converter provider for video files
type Video struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// Make creates a new Video converter
func (video *Video) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// Start with default values
	videoOption := converter.VideoOption{
		AudioConverter:     nil,  // Will be set from option if provided
		VisionConverter:    nil,  // Will be set from option if provided
		KeyframeInterval:   10.0, // Default 10 seconds
		MaxKeyframes:       20,   // Default max 20 keyframes
		TempDir:            "",   // Use system temp
		CleanupTemp:        true, // Default cleanup
		MaxConcurrency:     4,    // Default 4 concurrent processes
		TextOptimization:   true, // Default enable text optimization
		DeduplicationRatio: 0.8,  // Default deduplication ratio
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		if keyframeInterval, ok := option.Properties["keyframe_interval"]; ok {
			if intervalFloat, ok := keyframeInterval.(float64); ok {
				videoOption.KeyframeInterval = intervalFloat
			} else if intervalInt, ok := keyframeInterval.(int); ok {
				videoOption.KeyframeInterval = float64(intervalInt)
			}
		}

		if maxKeyframes, ok := option.Properties["max_keyframes"]; ok {
			if maxInt, ok := maxKeyframes.(int); ok {
				videoOption.MaxKeyframes = maxInt
			} else if maxFloat, ok := maxKeyframes.(float64); ok {
				videoOption.MaxKeyframes = int(maxFloat)
			}
		}

		if tempDir, ok := option.Properties["temp_dir"]; ok {
			if tempDirStr, ok := tempDir.(string); ok {
				videoOption.TempDir = tempDirStr
			}
		}

		if cleanupTemp, ok := option.Properties["cleanup_temp"]; ok {
			if cleanupBool, ok := cleanupTemp.(bool); ok {
				videoOption.CleanupTemp = cleanupBool
			}
		}

		if maxConcurrency, ok := option.Properties["max_concurrency"]; ok {
			if maxInt, ok := maxConcurrency.(int); ok {
				videoOption.MaxConcurrency = maxInt
			} else if maxFloat, ok := maxConcurrency.(float64); ok {
				videoOption.MaxConcurrency = int(maxFloat)
			}
		}

		if textOptimization, ok := option.Properties["text_optimization"]; ok {
			if optimizationBool, ok := textOptimization.(bool); ok {
				videoOption.TextOptimization = optimizationBool
			}
		}

		if deduplicationRatio, ok := option.Properties["deduplication_ratio"]; ok {
			if ratioFloat, ok := deduplicationRatio.(float64); ok {
				videoOption.DeduplicationRatio = ratioFloat
			} else if ratioInt, ok := deduplicationRatio.(int); ok {
				videoOption.DeduplicationRatio = float64(ratioInt)
			}
		}

		// Handle nested vision converter
		if vision, ok := option.Properties["vision"]; ok {
			visionConverter, err := parseNestedConverter(vision)
			if err != nil {
				return nil, fmt.Errorf("failed to parse vision converter: %w", err)
			}
			videoOption.VisionConverter = visionConverter
		}

		// Handle nested audio converter
		if audio, ok := option.Properties["audio"]; ok {
			audioConverter, err := parseNestedConverter(audio)
			if err != nil {
				return nil, fmt.Errorf("failed to parse audio converter: %w", err)
			}
			videoOption.AudioConverter = audioConverter
		}
	}

	return converter.NewVideo(videoOption)
}

// AutoDetect detects the converter based on the filename and content types
func (video *Video) AutoDetect(filename, contentTypes string) (bool, int, error) {
	// If autodetect is empty, return false
	if video.Autodetect == nil {
		return false, 0, nil
	}

	// Check if the filename matches the autodetect
	for _, autodetect := range video.Autodetect {
		if strings.HasSuffix(filename, autodetect) {
			return true, video.MatchPriority, nil
		}

		// Check if the content types matches the autodetect
		if strings.Contains(contentTypes, autodetect) {
			return true, video.MatchPriority, nil
		}
	}

	return false, 0, nil
}

// Schema returns the schema for the Video converter
func (video *Video) Schema(provider *kbtypes.Provider) (*kbtypes.ProviderSchema, error) {
	return nil, nil
}
