package tools

import (
	_ "embed"
	"encoding/json"

	"github.com/yaoapp/gou/mcp"
	mcpTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/tools/agent"
	"github.com/yaoapp/yao/tools/docs"
	"github.com/yaoapp/yao/tools/image"
	"github.com/yaoapp/yao/tools/proc"
	"github.com/yaoapp/yao/tools/webfetch"
	"github.com/yaoapp/yao/tools/websearch"
)

//go:embed mcps/web.json
var mcpWebDSL []byte

//go:embed mcps/process.json
var mcpProcessDSL []byte

//go:embed mcps/doc.json
var mcpDocDSL []byte

//go:embed mcps/image.json
var mcpImageDSL []byte

//go:embed mcps/agent.json
var mcpAgentDSL []byte

func init() {
	process.RegisterGroup("tools", map[string]process.Handler{
		"web_search":       websearch.Handler,
		"web_fetch":        webfetch.Handler,
		"process_call":     proc.Handler,
		"process_allowed":  proc.AllowedHandler,
		"doc_list":         docs.ListHandler,
		"doc_inspect":      docs.InspectHandler,
		"doc_validate":     docs.ValidateHandler,
		"image_read":       image.ReadHandler,
		"image_generate":   image.GenerateHandler,
		"image_providers":  image.ProvidersHandler,
		"agent_list":       agent.ListHandler,
		"agent_download":   agent.DownloadHandler,
		"agent_reference":  agent.ReferenceHandler,
		"agent_deploy":     agent.DeployHandler,
		"agent_connectors": agent.ConnectorsHandler,
	})

	registerMCPServer(mcpWebDSL, "yao-web",
		websearch.SchemaJSON, webfetch.SchemaJSON)
	registerMCPServer(mcpProcessDSL, "yao-process",
		proc.SchemaJSON, proc.AllowedSchemaJSON)
	registerMCPServer(mcpDocDSL, "yao-doc",
		docs.ListSchemaJSON, docs.InspectSchemaJSON, docs.ValidateSchemaJSON)
	registerMCPServer(mcpImageDSL, "yao-image",
		image.ReadSchemaJSON, image.GenerateSchemaJSON, image.ProvidersSchemaJSON)
	registerMCPServer(mcpAgentDSL, "yao-agent",
		agent.ListSchemaJSON, agent.DownloadSchemaJSON, agent.ReferenceSchemaJSON,
		agent.DeploySchemaJSON, agent.ConnectorsSchemaJSON)
}

func registerMCPServer(dsl []byte, id string, schemas ...[]byte) {
	mapping := &mcpTypes.MappingData{
		Tools:     map[string]*mcpTypes.ToolSchema{},
		Resources: map[string]*mcpTypes.ResourceSchema{},
		Prompts:   map[string]*mcpTypes.PromptSchema{},
	}
	for _, raw := range schemas {
		var s mcpTypes.ToolSchema
		if err := json.Unmarshal(raw, &s); err != nil {
			log.Error("[tools] failed to parse schema: %s", err.Error())
			continue
		}
		mapping.Tools[s.Name] = &s
	}
	if _, err := mcp.LoadClientSourceWithType(string(dsl), id, "", mapping); err != nil {
		log.Error("[tools] failed to register MCP server %s: %s", id, err.Error())
	}
}
