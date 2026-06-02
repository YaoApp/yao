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
	"github.com/yaoapp/yao/tools/robot"
	"github.com/yaoapp/yao/tools/secret"
	"github.com/yaoapp/yao/tools/webfetch"
	"github.com/yaoapp/yao/tools/websearch"
	"github.com/yaoapp/yao/tools/workspace"
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

//go:embed mcps/secret.json
var mcpSecretDSL []byte

//go:embed mcps/robot.json
var mcpRobotDSL []byte

//go:embed mcps/workspace.json
var mcpWorkspaceDSL []byte

func init() {
	process.RegisterGroup("tools", map[string]process.Handler{
		"web_search":        websearch.Handler,
		"web_fetch":         webfetch.Handler,
		"process_call":      proc.Handler,
		"process_allowed":   proc.AllowedHandler,
		"doc_list":          docs.ListHandler,
		"doc_inspect":       docs.InspectHandler,
		"doc_validate":      docs.ValidateHandler,
		"image_read":        image.ReadHandler,
		"image_generate":    image.GenerateHandler,
		"image_providers":   image.ProvidersHandler,
		"agent_list":        agent.ListHandler,
		"agent_download":    agent.DownloadHandler,
		"agent_reference":   agent.ReferenceHandler,
		"agent_deploy":      agent.DeployHandler,
		"agent_connectors":  agent.ConnectorsHandler,
		"agent_call":        agent.CallHandler,
		"secret_read":       secret.ReadHandler,
		"secret_list":       secret.ListHandler,
		"secret_connectors": secret.ConnectorsHandler,

		"robot_list":             robot.ListHandler,
		"robot_get":              robot.GetHandler,
		"robot_create":           robot.CreateHandler,
		"robot_update":           robot.UpdateHandler,
		"robot_status":           robot.StatusHandler,
		"robot_execution_list":   robot.ExecutionListHandler,
		"robot_execution_get":    robot.ExecutionGetHandler,
		"robot_execution_create": robot.ExecutionCreateHandler,
		"robot_execution_cancel": robot.ExecutionCancelHandler,
		"robot_result_list":      robot.ResultListHandler,

		"workspace_list":       workspace.ListHandler,
		"workspace_get":        workspace.GetHandler,
		"workspace_file_list":  workspace.FileListHandler,
		"workspace_file_read":  workspace.FileReadHandler,
		"workspace_file_write": workspace.FileWriteHandler,
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
		agent.DeploySchemaJSON, agent.ConnectorsSchemaJSON, agent.CallSchemaJSON)
	registerMCPServer(mcpSecretDSL, "yao-secret",
		secret.ReadSchemaJSON, secret.ListSchemaJSON, secret.ConnectorsSchemaJSON)
	registerMCPServer(mcpRobotDSL, "yao-robot",
		robot.ListSchemaJSON, robot.GetSchemaJSON, robot.CreateSchemaJSON,
		robot.UpdateSchemaJSON, robot.StatusSchemaJSON,
		robot.ExecutionListSchemaJSON, robot.ExecutionGetSchemaJSON,
		robot.ExecutionCreateSchemaJSON, robot.ExecutionCancelSchemaJSON,
		robot.ResultListSchemaJSON)
	registerMCPServer(mcpWorkspaceDSL, "yao-workspace",
		workspace.ListSchemaJSON, workspace.GetSchemaJSON,
		workspace.FileListSchemaJSON, workspace.FileReadSchemaJSON, workspace.FileWriteSchemaJSON)
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
