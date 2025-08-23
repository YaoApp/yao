package converters

import (
	"strings"

	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// UTF8 is a converter provider for utf8 files
type UTF8 struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// Make creates a new UTF8 converter
func (utf8 *UTF8) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// UTF8 converter doesn't need any configuration, just return a new instance
	return converter.NewUTF8(), nil
}

// AutoDetect detects the converter based on the filename and content types
func (utf8 *UTF8) AutoDetect(filename, contentTypes string) (bool, int, error) {
	// If autodetect is empty, return false
	if utf8.Autodetect == nil {
		return false, 0, nil
	}

	// Check if the filename matches the autodetect
	for _, autodetect := range utf8.Autodetect {
		if strings.HasSuffix(filename, autodetect) {
			return true, utf8.MatchPriority, nil
		}

		// Check if the content types matches the autodetect
		if strings.Contains(contentTypes, autodetect) {
			return true, utf8.MatchPriority, nil
		}
	}

	return false, 0, nil
}

// Schema returns the schema for the UTF8 converter
func (utf8 *UTF8) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeConverter, "utf8", locale)
}
