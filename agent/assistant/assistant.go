package assistant

import (
	"fmt"
	"path"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/agent/i18n"
	store "github.com/yaoapp/yao/agent/store/types"
	sui "github.com/yaoapp/yao/sui/core"
)

// Save save the assistant
func (ast *Assistant) Save() error {
	if storage == nil {
		return fmt.Errorf("storage is not set")
	}

	_, err := storage.SaveAssistant(&ast.AssistantModel)
	if err != nil {
		return err
	}

	return nil
}

// Map convert the assistant to a map
func (ast *Assistant) Map() map[string]interface{} {

	if ast == nil {
		return nil
	}

	return map[string]interface{}{
		"assistant_id": ast.ID,
		"type":         ast.Type,
		"name":         ast.Name,
		"readonly":     ast.Readonly,
		"public":       ast.Public,
		"share":        ast.Share,
		"avatar":       ast.Avatar,
		"connector":    ast.Connector,
		"path":         ast.Path,
		"built_in":     ast.BuiltIn,
		"sort":         ast.Sort,
		"description":  ast.Description,
		"options":      ast.Options,
		"prompts":      ast.Prompts,
		"kb":           ast.KB,
		"mcp":          ast.MCP,
		"tools":        ast.Tools,
		"workflow":     ast.Workflow,
		"tags":         ast.Tags,
		"mentionable":  ast.Mentionable,
		"automated":    ast.Automated,
		"placeholder":  ast.Placeholder,
		"locales":      ast.Locales,
		"created_at":   store.ToMySQLTime(ast.CreatedAt),
		"updated_at":   store.ToMySQLTime(ast.UpdatedAt),
	}
}

// Validate validates the assistant configuration
func (ast *Assistant) Validate() error {
	if ast.ID == "" {
		return fmt.Errorf("assistant_id is required")
	}
	if ast.Name == "" {
		return fmt.Errorf("name is required")
	}
	if ast.Connector == "" {
		return fmt.Errorf("connector is required")
	}
	return nil
}

// Assets get the assets content
func (ast *Assistant) Assets(name string, data sui.Data) (string, error) {

	app, err := fs.Get("app")
	if err != nil {
		return "", err
	}

	root := path.Join(ast.Path, "assets", name)
	raw, err := app.ReadFile(root)
	if err != nil {
		return "", err
	}

	if data != nil {
		content, _ := data.Replace(string(raw))
		return content, nil
	}

	return string(raw), nil
}

// Clone creates a deep copy of the assistant
func (ast *Assistant) Clone() *Assistant {
	if ast == nil {
		return nil
	}

	clone := &Assistant{
		AssistantModel: store.AssistantModel{
			ID:          ast.ID,
			Type:        ast.Type,
			Name:        ast.Name,
			Avatar:      ast.Avatar,
			Connector:   ast.Connector,
			Path:        ast.Path,
			BuiltIn:     ast.BuiltIn,
			Sort:        ast.Sort,
			Description: ast.Description,
			Readonly:    ast.Readonly,
			Public:      ast.Public,
			Share:       ast.Share,
			Mentionable: ast.Mentionable,
			Automated:   ast.Automated,
			CreatedAt:   ast.CreatedAt,
			UpdatedAt:   ast.UpdatedAt,
		},
		Search: ast.Search,
		Script: ast.Script,
		openai: ast.openai,
	}

	// Deep copy tags
	if ast.Tags != nil {
		clone.Tags = make([]string, len(ast.Tags))
		copy(clone.Tags, ast.Tags)
	}

	// Deep copy KB
	if ast.KB != nil {
		clone.KB = &store.KnowledgeBase{}
		if ast.KB.Collections != nil {
			clone.KB.Collections = make([]string, len(ast.KB.Collections))
			copy(clone.KB.Collections, ast.KB.Collections)
		}
		if ast.KB.Options != nil {
			clone.KB.Options = make(map[string]interface{})
			for k, v := range ast.KB.Options {
				clone.KB.Options[k] = v
			}
		}
	}

	// Deep copy MCP
	if ast.MCP != nil {
		clone.MCP = &store.MCPServers{}
		if ast.MCP.Servers != nil {
			clone.MCP.Servers = make([]string, len(ast.MCP.Servers))
			copy(clone.MCP.Servers, ast.MCP.Servers)
		}
		if ast.MCP.Options != nil {
			clone.MCP.Options = make(map[string]interface{})
			for k, v := range ast.MCP.Options {
				clone.MCP.Options[k] = v
			}
		}
	}

	// Deep copy options
	if ast.Options != nil {
		clone.Options = make(map[string]interface{})
		for k, v := range ast.Options {
			clone.Options[k] = v
		}
	}

	// Deep copy prompts
	if ast.Prompts != nil {
		clone.Prompts = make([]store.Prompt, len(ast.Prompts))
		copy(clone.Prompts, ast.Prompts)
	}

	// Deep copy tools
	if ast.Tools != nil {
		clone.Tools = &store.ToolCalls{}
		if ast.Tools.Tools != nil {
			clone.Tools.Tools = make([]store.Tool, len(ast.Tools.Tools))
			copy(clone.Tools.Tools, ast.Tools.Tools)
		}

		if ast.Tools.Prompts != nil {
			clone.Tools.Prompts = make([]store.Prompt, len(ast.Tools.Prompts))
			copy(clone.Tools.Prompts, ast.Tools.Prompts)
		}
	}

	// Deep copy workflow
	if ast.Workflow != nil {
		clone.Workflow = &store.Workflow{}
		if ast.Workflow.Workflows != nil {
			clone.Workflow.Workflows = make([]string, len(ast.Workflow.Workflows))
			copy(clone.Workflow.Workflows, ast.Workflow.Workflows)
		}
		if ast.Workflow.Options != nil {
			clone.Workflow.Options = make(map[string]interface{})
			for k, v := range ast.Workflow.Options {
				clone.Workflow.Options[k] = v
			}
		}
	}

	// Deep copy placeholder
	if ast.Placeholder != nil {
		clone.Placeholder = &store.Placeholder{
			Title:       ast.Placeholder.Title,
			Description: ast.Placeholder.Description,
		}
		if ast.Placeholder.Prompts != nil {
			clone.Placeholder.Prompts = make([]string, len(ast.Placeholder.Prompts))
			copy(clone.Placeholder.Prompts, ast.Placeholder.Prompts)
		}
	}

	// Deep copy locales
	if ast.Locales != nil {
		clone.Locales = make(i18n.Map)
		for k, v := range ast.Locales {
			// Deep copy messages
			messages := make(map[string]any)
			if v.Messages != nil {
				for mk, mv := range v.Messages {
					messages[mk] = mv
				}
			}
			clone.Locales[k] = i18n.I18n{
				Locale:   v.Locale,
				Messages: messages,
			}
		}
	}

	return clone
}

// Update updates the assistant properties
func (ast *Assistant) Update(data map[string]interface{}) error {
	if ast == nil {
		return fmt.Errorf("assistant is nil")
	}

	if v, ok := data["name"].(string); ok {
		ast.Name = v
	}
	if v, ok := data["avatar"].(string); ok {
		ast.Avatar = v
	}
	if v, ok := data["description"].(string); ok {
		ast.Description = v
	}
	if v, ok := data["connector"].(string); ok {
		ast.Connector = v
	}

	if v, has := data["tools"]; has {
		switch tools := v.(type) {
		case []store.Tool:
			ast.Tools = &store.ToolCalls{
				Tools:   tools,
				Prompts: ast.Prompts,
			}

		case *store.ToolCalls:
			ast.Tools = tools

		default:
			raw, err := jsoniter.Marshal(tools)
			if err != nil {
				return err
			}
			ast.Tools = &store.ToolCalls{}
			err = jsoniter.Unmarshal(raw, &ast.Tools)
			if err != nil {
				return err
			}
		}
	}

	if v, ok := data["type"].(string); ok {
		ast.Type = v
	}
	if v, ok := data["sort"].(int); ok {
		ast.Sort = v
	}
	if v, ok := data["mentionable"].(bool); ok {
		ast.Mentionable = v
	}
	if v, ok := data["automated"].(bool); ok {
		ast.Automated = v
	}
	if v, ok := data["readonly"].(bool); ok {
		ast.Readonly = v
	}
	if v, ok := data["public"].(bool); ok {
		ast.Public = v
	}
	if v, ok := data["share"].(string); ok {
		ast.Share = v
	}
	if v, ok := data["tags"].([]string); ok {
		ast.Tags = v
	}
	if v, ok := data["options"].(map[string]interface{}); ok {
		ast.Options = v
	}

	// KB
	if v, has := data["kb"]; has {
		kb, err := store.ToKnowledgeBase(v)
		if err != nil {
			return err
		}
		ast.KB = kb
	}

	// MCP
	if v, has := data["mcp"]; has {
		mcp, err := store.ToMCPServers(v)
		if err != nil {
			return err
		}
		ast.MCP = mcp
	}

	// Workflow
	if v, has := data["workflow"]; has {
		workflow, err := store.ToWorkflow(v)
		if err != nil {
			return err
		}
		ast.Workflow = workflow
	}

	return ast.Validate()
}
