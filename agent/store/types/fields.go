package types

import "github.com/yaoapp/kun/log"

// AssistantAllowedFields defines the whitelist of fields that can be selected for assistants
var AssistantAllowedFields = map[string]bool{
	"id":                     true,
	"assistant_id":           true,
	"type":                   true,
	"name":                   true,
	"avatar":                 true,
	"connector":              true,
	"connector_options":      true,
	"description":            true,
	"path":                   true,
	"sort":                   true,
	"built_in":               true,
	"placeholder":            true,
	"options":                true,
	"prompts":                true,
	"prompt_presets":         true,
	"disable_global_prompts": true,
	"workflow":               true,
	"kb":                     true,
	"mcp":                    true,
	"source":                 true,
	"tags":                   true,
	"readonly":               true,
	"public":                 true,
	"share":                  true,
	"locales":                true,
	"uses":                   true,
	"automated":              true,
	"mentionable":            true,
	"created_at":             true,
	"updated_at":             true,
	"__yao_created_by":       true,
	"__yao_updated_by":       true,
	"__yao_team_id":          true,
	"__yao_tenant_id":        true,
}

// AssistantDefaultFields defines the default fields to select for assistants when no specific fields are requested
// These are lightweight fields suitable for list views and basic information display
var AssistantDefaultFields = []string{
	"assistant_id",
	"type",
	"name",
	"avatar",
	"connector",
	"description",
	"tags", // Tags for categorization (lightweight)
	"sort",
	"built_in",
	"readonly",
	"public",
	"share",
	"automated",
	"mentionable",
	"kb",  // Knowledge base configuration (lightweight)
	"mcp", // MCP servers configuration (lightweight)
	"created_at",
	"updated_at",
	"__yao_created_by", // Permission: creator user ID
	"__yao_updated_by", // Permission: updater user ID
	"__yao_team_id",    // Permission: team ID
	"__yao_tenant_id",  // Permission: tenant ID
}

// AssistantFullFields defines all available fields including complex/large fields
// Use this when you need complete assistant data for backend processing
var AssistantFullFields = []string{
	"assistant_id",
	"type",
	"name",
	"avatar",
	"connector",
	"connector_options",
	"description",
	"path",
	"sort",
	"built_in",
	"placeholder",
	"options",
	"prompts",
	"prompt_presets",
	"disable_global_prompts",
	"workflow",
	"kb",
	"mcp",
	"source",
	"tags",
	"readonly",
	"public",
	"share",
	"locales",
	"uses",
	"automated",
	"mentionable",
	"created_at",
	"updated_at",
	"__yao_created_by",
	"__yao_updated_by",
	"__yao_team_id",
	"__yao_tenant_id",
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
