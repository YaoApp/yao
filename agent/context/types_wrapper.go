package context

import "strings"

// WrapperType represents the type of wrapper for processing
type WrapperType string

const (
	WrapperTypeAgent WrapperType = "agent" // Use agent for processing
	WrapperTypeMCP   WrapperType = "mcp"   // Use MCP server for processing
)

// ParseWrapper parses a wrapper string and returns the type and ID
// Format: "agent" or "mcp:mcp_server_id"
func ParseWrapper(wrapper string) (WrapperType, string) {
	if wrapper == "" || wrapper == "agent" {
		return WrapperTypeAgent, ""
	}

	if strings.HasPrefix(wrapper, "mcp:") {
		mcpID := strings.TrimPrefix(wrapper, "mcp:")
		return WrapperTypeMCP, mcpID
	}

	// Default to agent if format is unknown
	return WrapperTypeAgent, ""
}

// IsAgentWrapper checks if the wrapper is an agent wrapper
func IsAgentWrapper(wrapper string) bool {
	wrapperType, _ := ParseWrapper(wrapper)
	return wrapperType == WrapperTypeAgent
}

// IsMCPWrapper checks if the wrapper is an MCP wrapper
func IsMCPWrapper(wrapper string) bool {
	wrapperType, _ := ParseWrapper(wrapper)
	return wrapperType == WrapperTypeMCP
}

// GetMCPServerID extracts the MCP server ID from wrapper string
// Returns empty string if not an MCP wrapper
func GetMCPServerID(wrapper string) string {
	wrapperType, id := ParseWrapper(wrapper)
	if wrapperType == WrapperTypeMCP {
		return id
	}
	return ""
}
