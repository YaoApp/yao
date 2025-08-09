package factory

import (
	"fmt"
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/data"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// GetSchemaFromBindata reads the schema from bindata
func GetSchemaFromBindata(typ ProviderType, name string, locale string) (*kbtypes.ProviderSchema, error) {
	local := strings.ToLower(locale)
	if local == "" {
		local = "en"
	}

	// Read the schema from bindata
	raw, err := data.Asset("yao/data/kb/providers/" + string(typ) + "/" + name + "/" + local + ".json")
	if err != nil {
		// fallback to en
		raw, err = data.Asset("yao/data/kb/providers/" + string(typ) + "/" + name + "/en.json")
		if err != nil {
			return nil, err
		}
	}

	// Replace the {{ $limit... }} with the actual limit values
	raw, err = replaceVars(raw, map[string]interface{}{
		"limit.max_concurrent":      10,
		"limit.task.max_concurrent": 10,
	})
	if err != nil {
		return nil, err
	}

	schema := &kbtypes.ProviderSchema{}
	if err := jsoniter.Unmarshal(raw, schema); err != nil {
		return nil, err
	}

	return schema, nil
}

// ReplaceVars replaces the variables in the raw data {{ $... }}
func replaceVars(raw []byte, vars map[string]interface{}) ([]byte, error) {
	result := string(raw)

	// Regular expressions to match quoted and unquoted variables
	// Match "{{ $variable }}" (quoted) or {{ $variable }} (unquoted)
	var regQuoted = regexp.MustCompile(`"\{\{\s*\$([a-zA-Z_][a-zA-Z0-9_.]*)\s*\}\}"`)
	var regUnquoted = regexp.MustCompile(`\{\{\s*\$([a-zA-Z_][a-zA-Z0-9_.]*)\s*\}\}`)

	// First, process quoted variables
	quotedMatches := regQuoted.FindAllStringSubmatch(result, -1)
	for _, match := range quotedMatches {
		fullMatch := match[0] // Full match, e.g. "{{ $limit.max_concurrent }}"
		varName := match[1]   // Variable name, e.g. "limit.max_concurrent"

		if value, exists := vars[varName]; exists {
			// For quoted variables, replace the entire quoted part with JSON-encoded value
			valueBytes, err := jsoniter.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal variable %s: %v", varName, err)
			}
			replacement := string(valueBytes)
			result = strings.ReplaceAll(result, fullMatch, replacement)
		}
	}

	// Then, process unquoted variables
	unquotedMatches := regUnquoted.FindAllStringSubmatch(result, -1)
	for _, match := range unquotedMatches {
		fullMatch := match[0] // Full match, e.g. {{ $limit.max_concurrent }}
		varName := match[1]   // Variable name, e.g. "limit.max_concurrent"

		if value, exists := vars[varName]; exists {
			// For unquoted variables, also use JSON encoding
			valueBytes, err := jsoniter.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal variable %s: %v", varName, err)
			}
			replacement := string(valueBytes)
			result = strings.ReplaceAll(result, fullMatch, replacement)
		}
	}

	return []byte(result), nil
}
