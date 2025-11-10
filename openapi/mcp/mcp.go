package mcp

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/mcp"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Server represents an MCP server option (from user perspective)
type Server struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Transport   string `json:"transport,omitempty"` // "stdio", "sse", "http"
	Builtin     bool   `json:"builtin"`             // true for system built-in, false for user-defined
}

// Attach attaches the MCP server management handlers to the router with OAuth protection
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {

	// Create servers group with OAuth guard
	group.Use(oauth.Guard)

	// MCP Servers endpoints
	group.GET("/servers", listServers) // GET /servers - List all MCP servers
}

// listServers lists all available MCP servers (loaded clients from user perspective)
func listServers(c *gin.Context) {
	allServers := make([]Server, 0)

	// Get all loaded MCP clients (they are servers from user perspective)
	clientIDs := mcp.ListClients()

	for _, id := range clientIDs {
		client, err := mcp.Select(id)
		if err != nil {
			continue
		}

		// Get metadata
		meta := client.GetMetaInfo()

		label := meta.Label
		if label == "" {
			label = id
		}

		name := id

		// Get transport type (if available from DSL or client info)
		transport := "" // Could extract from client implementation if needed

		allServers = append(allServers, Server{
			Label:       label,
			Value:       id,
			Name:        name,
			Description: meta.Description,
			Transport:   transport,
			Builtin:     meta.Builtin,
		})
	}

	response.RespondWithSuccess(c, response.StatusOK, allServers)
}
