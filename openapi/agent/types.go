package agent

import (
	"fmt"
	"strings"

	agenttypes "github.com/yaoapp/yao/agent/store/types"
)

// Assistant field definitions
var (
	// availableAssistantFields defines all available fields for security filtering
	availableAssistantFields = map[string]bool{
		"id": true, "assistant_id": true, "type": true, "name": true, "avatar": true,
		"connector": true, "description": true, "path": true, "sort": true,
		"built_in": true, "placeholder": true, "options": true, "prompts": true,
		"workflow": true, "kb": true, "mcp": true, "tools": true, "tags": true,
		"readonly": true, "public": true, "share": true, "locales": true,
		"automated": true, "mentionable": true,
		"created_at": true, "updated_at": true, "deleted_at": true,
		"__yao_created_by": true, "__yao_updated_by": true, "__yao_team_id": true,
	}

	// defaultAssistantFields defines the default compact field list
	defaultAssistantFields = []string{
		"assistant_id", "type", "name", "avatar", "connector", "description",
		"sort", "built_in", "tags", "readonly", "public", "share",
		"automated", "mentionable", "created_at", "updated_at",
	}
)

// parseBoolValue parses various string formats into a boolean pointer
// Supports: 1, 0, "1", "0", "true", "false", etc.
func parseBoolValue(value string) *bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "1", "true", "yes", "on":
		v := true
		return &v
	case "0", "false", "no", "off":
		v := false
		return &v
	}
	return nil
}

// AssistantFilterParams represents the parameters for building an AssistantFilter
type AssistantFilterParams struct {
	Page         int
	PageSize     int
	Keywords     string
	Type         string   // Single type filter
	Types        []string // Multiple types filter (IN query)
	Connector    string
	AssistantID  string
	AssistantIDs []string
	Tags         []string
	SelectFields []string
	BuiltIn      *bool
	Mentionable  *bool
	Automated    *bool
	Public       *bool
	Share        string
}

// BuildAssistantFilter builds an AssistantFilter from parameters
func BuildAssistantFilter(params AssistantFilterParams) agenttypes.AssistantFilter {
	filter := agenttypes.AssistantFilter{
		Page:         params.Page,
		PageSize:     params.PageSize,
		Keywords:     params.Keywords,
		Tags:         params.Tags,
		Type:         params.Type,
		Types:        params.Types,
		Connector:    params.Connector,
		AssistantID:  params.AssistantID,
		AssistantIDs: params.AssistantIDs,
		Select:       params.SelectFields,
		BuiltIn:      params.BuiltIn,
		Mentionable:  params.Mentionable,
		Automated:    params.Automated,
	}

	// Set default type if not specified (only when Types is also empty)
	if filter.Type == "" && len(filter.Types) == 0 {
		filter.Type = "assistant"
	}

	// Set default pagination
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	return filter
}

// ValidatePagination validates pagination parameters
func ValidatePagination(page, pagesize int) error {
	if page < 0 {
		return fmt.Errorf("page must be positive")
	}
	if pagesize < 0 {
		return fmt.Errorf("pagesize must be positive")
	}
	if pagesize > 100 {
		return fmt.Errorf("pagesize cannot exceed 100")
	}
	return nil
}
