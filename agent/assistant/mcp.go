package assistant

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/mcp"
	mcpTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/kun/log"
	agentContext "github.com/yaoapp/yao/agent/context"
)

const (
	// MaxMCPTools maximum number of MCP tools to include (to avoid overwhelming the LLM)
	MaxMCPTools = 20
)

// MCPToolName formats a tool name with MCP server prefix
// Format: server_id__tool_name (double underscore separator)
// Dots in server_id are replaced with single underscores
// Examples:
//   - ("echo", "ping") → "echo__ping"
//   - ("github.enterprise", "search") → "github_enterprise__search"
//
// Naming constraint: MCP server_id MUST NOT contain underscores (_)
// Only dots (.), letters, numbers, and hyphens (-) are allowed in server_id
func MCPToolName(serverID, toolName string) string {
	if serverID == "" || toolName == "" {
		return ""
	}
	// Replace dots with single underscores in server_id
	cleanServerID := strings.ReplaceAll(serverID, ".", "_")
	// Use double underscore as separator
	return fmt.Sprintf("%s__%s", cleanServerID, toolName)
}

// ParseMCPToolName parses a formatted MCP tool name into server ID and tool name
// Splits by double underscore (__), then restores dots in server_id
// Examples:
//   - "echo__ping" → ("echo", "ping")
//   - "github_enterprise__search" → ("github.enterprise", "search")
//
// Returns (serverID, toolName, true) if valid format, ("", "", false) otherwise
func ParseMCPToolName(formattedName string) (string, string, bool) {
	if formattedName == "" {
		return "", "", false
	}

	// Split by double underscore
	parts := strings.Split(formattedName, "__")
	if len(parts) != 2 {
		return "", "", false
	}

	cleanServerID := parts[0]
	toolName := parts[1]

	// Validate that both parts are non-empty
	if cleanServerID == "" || toolName == "" {
		return "", "", false
	}

	// Restore dots in server_id (replace single underscores back to dots)
	serverID := strings.ReplaceAll(cleanServerID, "_", ".")

	return serverID, toolName, true
}

// MCPTool represents a simplified MCP tool for building LLM requests
type MCPTool struct {
	Name        string
	Description string
	Parameters  interface{}
}

// buildMCPTools builds tool definitions and samples system prompt from MCP servers
// Returns (tools, samplesPrompt, error)
func (ast *Assistant) buildMCPTools(ctx *agentContext.Context) ([]MCPTool, string, error) {
	if ast.MCP == nil || len(ast.MCP.Servers) == 0 {
		return nil, "", nil
	}

	mcpCtx := context.Background()
	allTools := make([]MCPTool, 0)
	samplesBuilder := strings.Builder{}
	hasSamples := false

	// Process each MCP server in order
	for _, serverConfig := range ast.MCP.Servers {
		if len(allTools) >= MaxMCPTools {
			log.Warn("[Assistant MCP] Reached maximum tool limit (%d), skipping remaining servers", MaxMCPTools)
			break
		}

		// Get MCP client
		client, err := mcp.Select(serverConfig.ServerID)
		if err != nil {
			log.Warn("[Assistant MCP] Failed to select MCP client '%s': %v", serverConfig.ServerID, err)
			continue
		}

		// Get tools list (filter by serverConfig.Tools if specified)
		toolsResponse, err := client.ListTools(mcpCtx, "")
		if err != nil {
			log.Warn("[Assistant MCP] Failed to list tools for '%s': %v", serverConfig.ServerID, err)
			continue
		}

		// Build tool filter map if specified
		toolFilter := make(map[string]bool)
		if len(serverConfig.Tools) > 0 {
			for _, toolName := range serverConfig.Tools {
				toolFilter[toolName] = true
			}
		}

		// Process each tool
		for _, tool := range toolsResponse.Tools {
			// Check tool limit
			if len(allTools) >= MaxMCPTools {
				break
			}

			// Apply tool filter if specified
			if len(toolFilter) > 0 && !toolFilter[tool.Name] {
				continue
			}

			// Format tool name with server prefix
			formattedName := MCPToolName(serverConfig.ServerID, tool.Name)

			// Convert MCP tool to MCPTool format
			mcpTool := MCPTool{
				Name:        formattedName,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			}

			allTools = append(allTools, mcpTool)

			// Try to get samples for this tool
			samples, err := client.ListSamples(mcpCtx, mcpTypes.SampleTool, tool.Name)
			if err == nil && len(samples.Samples) > 0 {
				if !hasSamples {
					samplesBuilder.WriteString("\n\n## MCP Tool Usage Examples\n\n")
					samplesBuilder.WriteString("The following examples demonstrate how to use MCP tools correctly:\n\n")
					hasSamples = true
				}

				samplesBuilder.WriteString(fmt.Sprintf("### %s\n\n", formattedName))
				if tool.Description != "" {
					samplesBuilder.WriteString(fmt.Sprintf("**Description**: %s\n\n", tool.Description))
				}

				for i, sample := range samples.Samples {
					if i >= 3 { // Limit to 3 examples per tool
						break
					}

					samplesBuilder.WriteString(fmt.Sprintf("**Example %d", i+1))
					if sample.Name != "" {
						samplesBuilder.WriteString(fmt.Sprintf(" - %s", sample.Name))
					}
					samplesBuilder.WriteString("**:\n")

					// Check metadata for description
					if sample.Metadata != nil {
						if desc, ok := sample.Metadata["description"].(string); ok && desc != "" {
							samplesBuilder.WriteString(fmt.Sprintf("- Description: %s\n", desc))
						}
					}

					if sample.Input != nil {
						samplesBuilder.WriteString(fmt.Sprintf("- Input: `%v`\n", sample.Input))
					}

					if sample.Output != nil {
						samplesBuilder.WriteString(fmt.Sprintf("- Output: `%v`\n", sample.Output))
					}

					samplesBuilder.WriteString("\n")
				}
			}
		}

		log.Trace("[Assistant MCP] Loaded %d tools from server '%s'", len(toolsResponse.Tools), serverConfig.ServerID)
	}

	samplesPrompt := ""
	if hasSamples {
		samplesPrompt = samplesBuilder.String()
	}

	log.Trace("[Assistant MCP] Total MCP tools loaded: %d", len(allTools))
	return allTools, samplesPrompt, nil
}
