package types

import (
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/i18n"
)

// ToKnowledgeBase converts various types to KnowledgeBase
func ToKnowledgeBase(v interface{}) (*KnowledgeBase, error) {
	if v == nil {
		return nil, nil
	}

	switch kb := v.(type) {
	case *KnowledgeBase:
		return kb, nil

	case KnowledgeBase:
		return &kb, nil

	case []string:
		return &KnowledgeBase{Collections: kb}, nil

	case []interface{}:
		var collections []string
		for _, item := range kb {
			collections = append(collections, cast.ToString(item))
		}
		return &KnowledgeBase{Collections: collections}, nil

	default:
		raw, err := jsoniter.Marshal(kb)
		if err != nil {
			return nil, fmt.Errorf("kb format error: %s", err.Error())
		}

		var knowledgeBase KnowledgeBase
		err = jsoniter.Unmarshal(raw, &knowledgeBase)
		if err != nil {
			return nil, fmt.Errorf("kb format error: %s", err.Error())
		}
		return &knowledgeBase, nil
	}
}

// ToMCPServers converts various types to MCPServers
func ToMCPServers(v interface{}) (*MCPServers, error) {
	if v == nil {
		return nil, nil
	}

	switch mcp := v.(type) {
	case *MCPServers:
		return mcp, nil

	case MCPServers:
		return &mcp, nil

	case []string:
		return &MCPServers{Servers: mcp}, nil

	case []interface{}:
		var servers []string
		for _, item := range mcp {
			servers = append(servers, cast.ToString(item))
		}
		return &MCPServers{Servers: servers}, nil

	default:
		raw, err := jsoniter.Marshal(mcp)
		if err != nil {
			return nil, fmt.Errorf("mcp format error: %s", err.Error())
		}

		var mcpServers MCPServers
		err = jsoniter.Unmarshal(raw, &mcpServers)
		if err != nil {
			return nil, fmt.Errorf("mcp format error: %s", err.Error())
		}
		return &mcpServers, nil
	}
}

// ToWorkflow converts various types to Workflow
func ToWorkflow(v interface{}) (*Workflow, error) {
	if v == nil {
		return nil, nil
	}

	switch workflow := v.(type) {
	case *Workflow:
		return workflow, nil

	case Workflow:
		return &workflow, nil

	case []string:
		return &Workflow{Workflows: workflow}, nil

	case []interface{}:
		var workflows []string
		for _, item := range workflow {
			workflows = append(workflows, cast.ToString(item))
		}
		return &Workflow{Workflows: workflows}, nil

	default:
		raw, err := jsoniter.Marshal(workflow)
		if err != nil {
			return nil, fmt.Errorf("workflow format error: %s", err.Error())
		}

		var wf Workflow
		err = jsoniter.Unmarshal(raw, &wf)
		if err != nil {
			return nil, fmt.Errorf("workflow format error: %s", err.Error())
		}
		return &wf, nil
	}
}

// ToMySQLTime converts various types to MySQL datetime format
func ToMySQLTime(v interface{}) string {
	switch val := v.(type) {
	case int64:
		if val == 0 {
			return "0000-00-00 00:00:00"
		}
		return time.Unix(val/1e9, val%1e9).Format("2006-01-02 15:04:05")

	case int:
		if val == 0 {
			return "0000-00-00 00:00:00"
		}
		return time.Unix(int64(val)/1e9, int64(val)%1e9).Format("2006-01-02 15:04:05")

	case string:
		// If already in MySQL format, return as-is
		if _, err := time.Parse("2006-01-02 15:04:05", val); err == nil {
			return val
		}
		// Try RFC3339 format
		if ts, err := time.Parse(time.RFC3339, val); err == nil {
			return ts.Format("2006-01-02 15:04:05")
		}
		// Try parsing as Unix timestamp
		if ts, err := cast.ToInt64E(val); err == nil {
			if ts == 0 {
				return "0000-00-00 00:00:00"
			}
			return time.Unix(ts/1e9, ts%1e9).Format("2006-01-02 15:04:05")
		}
		return val

	case time.Time:
		if val.IsZero() {
			return "0000-00-00 00:00:00"
		}
		return val.Format("2006-01-02 15:04:05")

	case nil:
		return "0000-00-00 00:00:00"

	default:
		return "0000-00-00 00:00:00"
	}
}

// ToAssistantModel converts various types to AssistantModel
func ToAssistantModel(v interface{}) (*AssistantModel, error) {
	if v == nil {
		return nil, nil
	}

	// If already an AssistantModel, return it
	switch model := v.(type) {
	case *AssistantModel:
		return model, nil
	case AssistantModel:
		return &model, nil
	}

	// Convert to map first if needed
	var data map[string]interface{}
	switch v := v.(type) {
	case map[string]interface{}:
		data = v
	default:
		// Try to marshal and unmarshal
		raw, err := jsoniter.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal to AssistantModel: %w", err)
		}
		err = jsoniter.Unmarshal(raw, &data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
		}
	}

	model := &AssistantModel{}

	// Basic string fields
	if id, ok := data["assistant_id"].(string); ok {
		model.ID = id
	}
	if typ, ok := data["type"].(string); ok {
		model.Type = typ
	}
	if name, ok := data["name"].(string); ok {
		model.Name = name
	}
	if avatar, ok := data["avatar"].(string); ok {
		model.Avatar = avatar
	}
	if connector, ok := data["connector"].(string); ok {
		model.Connector = connector
	}
	if path, ok := data["path"].(string); ok {
		model.Path = path
	}
	if description, ok := data["description"].(string); ok {
		model.Description = description
	}
	if share, ok := data["share"].(string); ok {
		model.Share = share
	}

	// Boolean fields (handle both bool and int types from database)
	model.BuiltIn = getBoolValue(data, "built_in")
	model.Readonly = getBoolValue(data, "readonly")
	model.Public = getBoolValue(data, "public")
	model.Mentionable = getBoolValue(data, "mentionable")
	model.Automated = getBoolValue(data, "automated")

	// Integer fields
	if sort, ok := data["sort"].(int); ok {
		model.Sort = sort
	} else if sort, ok := data["sort"].(float64); ok {
		model.Sort = int(sort)
	}

	if createdAt, ok := data["created_at"].(int64); ok {
		model.CreatedAt = createdAt
	} else if createdAt, ok := data["created_at"].(float64); ok {
		model.CreatedAt = int64(createdAt)
	}

	if updatedAt, ok := data["updated_at"].(int64); ok {
		model.UpdatedAt = updatedAt
	} else if updatedAt, ok := data["updated_at"].(float64); ok {
		model.UpdatedAt = int64(updatedAt)
	}

	// Tags (string array)
	if tags, ok := data["tags"]; ok && tags != nil {
		raw, err := jsoniter.Marshal(tags)
		if err == nil {
			var t []string
			if err := jsoniter.Unmarshal(raw, &t); err == nil {
				model.Tags = t
			}
		}
	}

	// Options (map)
	if options, ok := data["options"].(map[string]interface{}); ok {
		model.Options = options
	}

	// Prompts
	if prompts, ok := data["prompts"]; ok && prompts != nil {
		raw, err := jsoniter.Marshal(prompts)
		if err == nil {
			var p []Prompt
			if err := jsoniter.Unmarshal(raw, &p); err == nil {
				model.Prompts = p
			}
		}
	}

	// KB
	if kb, ok := data["kb"]; ok && kb != nil {
		kbConverted, err := ToKnowledgeBase(kb)
		if err == nil {
			model.KB = kbConverted
		}
	}

	// MCP
	if mcp, ok := data["mcp"]; ok && mcp != nil {
		mcpConverted, err := ToMCPServers(mcp)
		if err == nil {
			model.MCP = mcpConverted
		}
	}

	// Workflow
	if workflow, ok := data["workflow"]; ok && workflow != nil {
		wf, err := ToWorkflow(workflow)
		if err == nil {
			model.Workflow = wf
		}
	}

	// Tools
	if tools, ok := data["tools"]; ok && tools != nil {
		raw, err := jsoniter.Marshal(tools)
		if err == nil {
			var tc ToolCalls
			if err := jsoniter.Unmarshal(raw, &tc); err == nil {
				model.Tools = &tc
			}
		}
	}

	// Placeholder
	if placeholder, ok := data["placeholder"]; ok && placeholder != nil {
		raw, err := jsoniter.Marshal(placeholder)
		if err == nil {
			var ph Placeholder
			if err := jsoniter.Unmarshal(raw, &ph); err == nil {
				model.Placeholder = &ph
			}
		}
	}

	// Locales
	if locales, ok := data["locales"]; ok && locales != nil {
		raw, err := jsoniter.Marshal(locales)
		if err == nil {
			var loc i18n.Map
			if err := jsoniter.Unmarshal(raw, &loc); err == nil {
				model.Locales = loc
			}
		}
	}

	// Permission fields
	if createdBy, ok := data["__yao_created_by"].(string); ok {
		model.YaoCreatedBy = createdBy
	}
	if updatedBy, ok := data["__yao_updated_by"].(string); ok {
		model.YaoUpdatedBy = updatedBy
	}
	if teamID, ok := data["__yao_team_id"].(string); ok {
		model.YaoTeamID = teamID
	}
	if tenantID, ok := data["__yao_tenant_id"].(string); ok {
		model.YaoTenantID = tenantID
	}

	return model, nil
}

// getBoolValue extracts a boolean value from a map, handling both bool and numeric types
func getBoolValue(data map[string]interface{}, key string) bool {
	if v, ok := data[key]; ok && v != nil {
		switch val := v.(type) {
		case bool:
			return val
		case int:
			return val != 0
		case int64:
			return val != 0
		case float64:
			return val != 0
		case string:
			return val == "true" || val == "1"
		}
	}
	return false
}

// ModelID generates an OpenAI-compatible model ID from assistant
// Format: [prefix-]assistantName-model-yao_assistantID
// prefix is optional, if provided, it will be prepended to the model ID
func (assistant AssistantModel) ModelID(prefix ...string) string {
	// Clean assistant name (remove spaces and special characters)
	assistantName := strings.ReplaceAll(assistant.Name, " ", "-")
	assistantName = strings.ToLower(assistantName)

	// Get connector name from assistant
	connectorName := assistant.Connector
	if connectorName == "" {
		log.Error("Assistant %s has no connector configured", assistant.ID)
		modelID := assistantName + "-unknown-yao_" + assistant.ID
		if len(prefix) > 0 && prefix[0] != "" {
			return prefix[0] + modelID
		}
		return modelID
	}

	// Get model name
	modelName := ""

	// First, try to get custom model from Options
	if assistant.Options != nil {
		if m, ok := assistant.Options["model"].(string); ok && m != "" {
			modelName = m
		}
	}

	// If no custom model in options, try to get from connector configuration
	if modelName == "" {
		conn, err := connector.Select(connectorName)
		if err != nil {
			log.Error("Failed to select connector %s for assistant %s: %v", connectorName, assistant.ID, err)
			modelID := assistantName + "-unknown-yao_" + assistant.ID
			if len(prefix) > 0 && prefix[0] != "" {
				return prefix[0] + modelID
			}
			return modelID
		}

		// Get model from connector settings
		settings := conn.Setting()
		if settings != nil {
			if m, ok := settings["model"].(string); ok && m != "" {
				modelName = m
			}
		}

		if modelName == "" {
			log.Error("Connector %s has no model configured for assistant %s", connectorName, assistant.ID)
			modelID := assistantName + "-unknown-yao_" + assistant.ID
			if len(prefix) > 0 && prefix[0] != "" {
				return prefix[0] + modelID
			}
			return modelID
		}
	}

	// Format: [prefix-]assistantName-model-yao_assistantID
	modelID := assistantName + "-" + modelName + "-yao_" + assistant.ID
	if len(prefix) > 0 && prefix[0] != "" {
		return prefix[0] + modelID
	}
	return modelID
}

// ParseModelID extracts assistant ID from model ID
// Expected format: [prefix-]assistantName-model-yao_assistantID
// The function handles optional prefixes (e.g., "yao-agents-")
func ParseModelID(modelID string) string {
	// Find the last occurrence of "yao_"
	parts := strings.Split(modelID, "-yao_")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}
