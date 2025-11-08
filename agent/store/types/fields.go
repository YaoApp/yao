package types

import "github.com/yaoapp/kun/log"

// AssistantAllowedFields defines the whitelist of fields that can be selected for assistants
var AssistantAllowedFields = map[string]bool{
	"id":               true,
	"assistant_id":     true,
	"type":             true,
	"name":             true,
	"avatar":           true,
	"connector":        true,
	"description":      true,
	"path":             true,
	"sort":             true,
	"built_in":         true,
	"placeholder":      true,
	"options":          true,
	"prompts":          true,
	"workflow":         true,
	"kb":               true,
	"mcp":              true,
	"tools":            true,
	"tags":             true,
	"readonly":         true,
	"public":           true,
	"share":            true,
	"locales":          true,
	"automated":        true,
	"mentionable":      true,
	"created_at":       true,
	"updated_at":       true,
	"__yao_created_by": true,
	"__yao_updated_by": true,
	"__yao_team_id":    true,
	"__yao_tenant_id":  true,
}

// AssistantDefaultFields defines the default fields to select for assistants when no specific fields are requested
var AssistantDefaultFields = []string{
	"assistant_id",
	"type",
	"name",
	"avatar",
	"connector",
	"description",
	"sort",
	"built_in",
	"readonly",
	"public",
	"share",
	"automated",
	"mentionable",
	"created_at",
	"updated_at",
}

// ValidateAssistantFields validates and filters assistant select fields against the whitelist
// Returns the filtered fields. If input is empty, returns empty slice (meaning no restriction).
// If all fields are invalid, returns default fields as fallback.
func ValidateAssistantFields(fields []string) []string {
	// If no fields specified, return empty slice (no restriction)
	if len(fields) == 0 {
		return []string{}
	}

	// Filter out any fields not in the whitelist
	sanitized := make([]string, 0, len(fields))
	for _, field := range fields {
		if AssistantAllowedFields[field] {
			sanitized = append(sanitized, field)
		} else {
			log.Warn("Ignoring invalid assistant select field: %s", field)
		}
	}

	// If all fields were filtered out, return default fields as fallback
	if len(sanitized) == 0 {
		log.Warn("All assistant select fields were invalid, using default fields")
		return AssistantDefaultFields
	}

	return sanitized
}
