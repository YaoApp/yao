package standard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// InputFormatter Integration Tests with Real Data
// These tests use the yao-dev-app environment with real assistants and MCPs
// ============================================================================

func TestFormatAvailableResourcesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	formatter := standard.NewInputFormatter()

	t.Run("formats_real_agents_with_details", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-agents",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Test Robot",
					Duties: []string{"Testing agent formatting"},
				},
				DefaultLocale: "en",
				Resources: &types.Resources{
					// Use real agents from yao-dev-app/assistants
					Agents: []string{
						"experts.data-analyst",
						"experts.text-writer",
						"experts.summarizer",
					},
				},
			},
		}

		result := formatter.FormatAvailableResources(robot)

		// Verify structure
		assert.Contains(t, result, "## Available Resources")
		assert.Contains(t, result, "### Agents")
		assert.Contains(t, result, "These are the AI assistants you can delegate tasks to:")

		// Verify real agent details are included
		// experts.data-analyst should show name and description
		assert.Contains(t, result, "experts.data-analyst")
		assert.Contains(t, result, "Data Analyst Expert") // Name from package.yao

		// experts.text-writer
		assert.Contains(t, result, "experts.text-writer")

		// experts.summarizer
		assert.Contains(t, result, "experts.summarizer")

		// Verify important note is present
		assert.Contains(t, result, "Only plan goals and tasks that can be accomplished")

		t.Logf("Formatted agents result:\n%s", result)
	})

	t.Run("formats_real_mcp_with_tool_details", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-mcp",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Test Robot",
					Duties: []string{"Testing MCP formatting"},
				},
				DefaultLocale: "en",
				Resources: &types.Resources{
					// Use real MCPs from yao-dev-app/mcps
					MCP: []types.MCPConfig{
						{ID: "echo", Tools: []string{"ping", "status"}}, // Specific tools
						{ID: "echo"}, // All tools
					},
				},
			},
		}

		result := formatter.FormatAvailableResources(robot)

		// Verify structure
		assert.Contains(t, result, "## Available Resources")
		assert.Contains(t, result, "### MCP Tools")
		assert.Contains(t, result, "These are the external tools and services you can use:")

		// Verify MCP details
		assert.Contains(t, result, "echo")

		// Verify important note is present
		assert.Contains(t, result, "Only plan goals and tasks that can be accomplished")

		t.Logf("Formatted MCP result:\n%s", result)
	})

	t.Run("formats_combined_resources", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-combined",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Analyst Robot",
					Duties: []string{"Analyze sales data", "Generate reports"},
					Rules:  []string{"Be accurate", "Be concise"},
				},
				DefaultLocale: "en",
				Resources: &types.Resources{
					Agents: []string{
						"experts.data-analyst",
						"experts.summarizer",
					},
					MCP: []types.MCPConfig{
						{ID: "echo", Tools: []string{"ping", "echo"}},
					},
				},
				KB: &types.KB{
					Collections: []string{"sales-policies", "product-catalog"},
				},
				DB: &types.DB{
					Models: []string{"sales", "customers", "orders"},
				},
			},
		}

		result := formatter.FormatAvailableResources(robot)

		// Verify all sections are present
		assert.Contains(t, result, "## Available Resources")
		assert.Contains(t, result, "### Agents")
		assert.Contains(t, result, "### MCP Tools")
		assert.Contains(t, result, "### Knowledge Base")
		assert.Contains(t, result, "### Database")

		// Verify agents
		assert.Contains(t, result, "experts.data-analyst")
		assert.Contains(t, result, "experts.summarizer")

		// Verify MCP
		assert.Contains(t, result, "echo")

		// Verify KB
		assert.Contains(t, result, "sales-policies")
		assert.Contains(t, result, "product-catalog")

		// Verify DB
		assert.Contains(t, result, "sales")
		assert.Contains(t, result, "customers")
		assert.Contains(t, result, "orders")

		t.Logf("Formatted combined resources result:\n%s", result)
	})

	t.Run("handles_locale_zh", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-zh",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "测试机器人",
					Duties: []string{"测试国际化"},
				},
				DefaultLocale: "zh-cn",
				Resources: &types.Resources{
					// Use agents that have zh-cn locales
					Agents: []string{
						"hello", // This agent has locales/zh-cn.yml
						"mohe",  // This agent also has locales/zh-cn.yml
					},
				},
			},
		}

		result := formatter.FormatAvailableResourcesWithLocale(robot, "zh-cn")

		// Verify structure
		assert.Contains(t, result, "## Available Resources")
		assert.Contains(t, result, "### Agents")

		// Verify agents are listed
		assert.Contains(t, result, "hello")
		assert.Contains(t, result, "mohe")

		t.Logf("Formatted zh-cn result:\n%s", result)
	})

	t.Run("gracefully_handles_missing_agents", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-missing",
			Config: &types.Config{
				Identity: &types.Identity{
					Role: "Test Robot",
				},
				Resources: &types.Resources{
					Agents: []string{
						"non-existent-agent",
						"experts.data-analyst", // This one exists
						"another-missing-agent",
					},
				},
			},
		}

		result := formatter.FormatAvailableResources(robot)

		// Should not panic, should include fallback for missing agents
		assert.Contains(t, result, "## Available Resources")
		assert.Contains(t, result, "### Agents")

		// Missing agents should still be listed with just ID
		assert.Contains(t, result, "non-existent-agent")
		assert.Contains(t, result, "another-missing-agent")

		// Existing agent should have full details
		assert.Contains(t, result, "experts.data-analyst")
		assert.Contains(t, result, "Data Analyst Expert")

		t.Logf("Formatted with missing agents:\n%s", result)
	})

	t.Run("gracefully_handles_missing_mcp", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-missing-mcp",
			Config: &types.Config{
				Identity: &types.Identity{
					Role: "Test Robot",
				},
				Resources: &types.Resources{
					MCP: []types.MCPConfig{
						{ID: "non-existent-mcp", Tools: []string{"tool1", "tool2"}},
						{ID: "echo"}, // This one exists
					},
				},
			},
		}

		result := formatter.FormatAvailableResources(robot)

		// Should not panic, should include fallback for missing MCP
		assert.Contains(t, result, "## Available Resources")
		assert.Contains(t, result, "### MCP Tools")

		// Missing MCP should still be listed with fallback
		assert.Contains(t, result, "non-existent-mcp")
		assert.Contains(t, result, "tool1, tool2")

		// Existing MCP should have details
		assert.Contains(t, result, "echo")

		t.Logf("Formatted with missing MCP:\n%s", result)
	})
}

func TestFormatAvailableResourcesTableFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	formatter := standard.NewInputFormatter()

	t.Run("mcp_tools_in_table_format", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-table",
			Config: &types.Config{
				Identity: &types.Identity{
					Role: "Test Robot",
				},
				Resources: &types.Resources{
					MCP: []types.MCPConfig{
						{ID: "echo"}, // All tools - should show table
					},
				},
			},
		}

		result := formatter.FormatAvailableResources(robot)

		// Check if table format is used when tools are available
		// Table headers: | Tool | Description |
		if strings.Contains(result, "| Tool | Description |") {
			assert.Contains(t, result, "|------|-------------|")
			t.Logf("MCP tools displayed in table format:\n%s", result)
		} else {
			// Fallback format
			t.Logf("MCP tools displayed in fallback format:\n%s", result)
		}
	})
}

func TestFormatClockContextWithRobotIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	formatter := standard.NewInputFormatter()

	t.Run("full_context_for_inspiration", func(t *testing.T) {
		// Create a realistic robot configuration
		robot := &types.Robot{
			MemberID:       "sales-robot-001",
			TeamID:         "team-001",
			DisplayName:    "Sales Analyst Robot",
			AutonomousMode: true,
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Analyst",
					Duties: []string{"Monitor sales performance", "Generate daily reports", "Alert on anomalies"},
					Rules:  []string{"Only use approved data sources", "Maintain confidentiality"},
				},
				DefaultLocale: "en",
				Resources: &types.Resources{
					Agents: []string{
						"experts.data-analyst",
						"experts.summarizer",
					},
					MCP: []types.MCPConfig{
						{ID: "echo", Tools: []string{"ping"}},
					},
				},
			},
		}

		// Create clock context
		clock := types.NewClockContext(time.Now(), "UTC")

		// Format clock context (includes robot identity)
		clockContent := formatter.FormatClockContext(clock, robot)

		// Format available resources
		resourcesContent := formatter.FormatAvailableResources(robot)

		// Combine for full context (as done in inspiration.go)
		fullContext := clockContent + "\n\n" + resourcesContent

		// Verify full context contains all necessary information
		require.NotEmpty(t, fullContext)

		// Time context
		assert.Contains(t, fullContext, "## Current Time Context")
		assert.Contains(t, fullContext, "### Time Markers")

		// Robot identity
		assert.Contains(t, fullContext, "## Robot Identity")
		assert.Contains(t, fullContext, "Sales Analyst")
		assert.Contains(t, fullContext, "Monitor sales performance")
		assert.Contains(t, fullContext, "Only use approved data sources")

		// Available resources
		assert.Contains(t, fullContext, "## Available Resources")
		assert.Contains(t, fullContext, "### Agents")
		assert.Contains(t, fullContext, "experts.data-analyst")
		assert.Contains(t, fullContext, "### MCP Tools")

		t.Logf("Full context for inspiration:\n%s", fullContext)
	})
}
