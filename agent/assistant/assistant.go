package assistant

import (
	"fmt"
	"path"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/agent/i18n"
	store "github.com/yaoapp/yao/agent/store/types"
	sui "github.com/yaoapp/yao/sui/core"
)

// Get get the assistant by id
func Get(id string) (*Assistant, error) {
	return LoadStore(id)
}

// GetPlaceholder returns the placeholder of the assistant
func (ast *Assistant) GetPlaceholder(locale string) *store.Placeholder {

	prompts := []string{}
	if ast.Placeholder.Prompts != nil {
		prompts = i18n.Translate(ast.ID, locale, ast.Placeholder.Prompts).([]string)
	}
	title := i18n.Translate(ast.ID, locale, ast.Placeholder.Title).(string)
	description := i18n.Translate(ast.ID, locale, ast.Placeholder.Description).(string)
	return &store.Placeholder{
		Title:       title,
		Description: description,
		Prompts:     prompts,
	}
}

// GetName returns the name of the assistant
func (ast *Assistant) GetName(locale string) string {
	return i18n.Translate(ast.ID, locale, ast.Name).(string)
}

// GetDescription returns the description of the assistant
func (ast *Assistant) GetDescription(locale string) string {
	return i18n.Translate(ast.ID, locale, ast.Description).(string)
}

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
		"assistant_id":           ast.ID,
		"type":                   ast.Type,
		"name":                   ast.Name,
		"readonly":               ast.Readonly,
		"public":                 ast.Public,
		"share":                  ast.Share,
		"avatar":                 ast.Avatar,
		"connector":              ast.Connector,
		"connector_options":      ast.ConnectorOptions,
		"path":                   ast.Path,
		"built_in":               ast.BuiltIn,
		"sort":                   ast.Sort,
		"description":            ast.Description,
		"options":                ast.Options,
		"prompts":                ast.Prompts,
		"prompt_presets":         ast.PromptPresets,
		"disable_global_prompts": ast.DisableGlobalPrompts,
		"source":                 ast.Source,
		"kb":                     ast.KB,
		"mcp":                    ast.MCP,
		"workflow":               ast.Workflow,
		"tags":                   ast.Tags,
		"mentionable":            ast.Mentionable,
		"automated":              ast.Automated,
		"placeholder":            ast.Placeholder,
		"locales":                ast.Locales,
		"created_at":             store.ToMySQLTime(ast.CreatedAt),
		"updated_at":             store.ToMySQLTime(ast.UpdatedAt),
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
			ID:                   ast.ID,
			Type:                 ast.Type,
			Name:                 ast.Name,
			Avatar:               ast.Avatar,
			Connector:            ast.Connector,
			Path:                 ast.Path,
			BuiltIn:              ast.BuiltIn,
			Sort:                 ast.Sort,
			Description:          ast.Description,
			Readonly:             ast.Readonly,
			Public:               ast.Public,
			Share:                ast.Share,
			Mentionable:          ast.Mentionable,
			Automated:            ast.Automated,
			DisableGlobalPrompts: ast.DisableGlobalPrompts,
			Source:               ast.Source,
			CreatedAt:            ast.CreatedAt,
			UpdatedAt:            ast.UpdatedAt,
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
			clone.MCP.Servers = make([]store.MCPServerConfig, len(ast.MCP.Servers))
			for i, server := range ast.MCP.Servers {
				clone.MCP.Servers[i] = store.MCPServerConfig{
					ServerID: server.ServerID,
				}
				// Deep copy Resources slice
				if server.Resources != nil {
					clone.MCP.Servers[i].Resources = make([]string, len(server.Resources))
					copy(clone.MCP.Servers[i].Resources, server.Resources)
				}
				// Deep copy Tools slice
				if server.Tools != nil {
					clone.MCP.Servers[i].Tools = make([]string, len(server.Tools))
					copy(clone.MCP.Servers[i].Tools, server.Tools)
				}
			}
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

	// Deep copy prompt presets
	if ast.PromptPresets != nil {
		clone.PromptPresets = make(map[string][]store.Prompt)
		for k, v := range ast.PromptPresets {
			prompts := make([]store.Prompt, len(v))
			copy(prompts, v)
			clone.PromptPresets[k] = prompts
		}
	}

	// Deep copy connector options
	if ast.ConnectorOptions != nil {
		clone.ConnectorOptions = &store.ConnectorOptions{
			Optional: ast.ConnectorOptions.Optional,
		}
		if ast.ConnectorOptions.Connectors != nil {
			clone.ConnectorOptions.Connectors = make([]string, len(ast.ConnectorOptions.Connectors))
			copy(clone.ConnectorOptions.Connectors, ast.ConnectorOptions.Connectors)
		}
		if ast.ConnectorOptions.Filters != nil {
			clone.ConnectorOptions.Filters = make([]store.ModelCapability, len(ast.ConnectorOptions.Filters))
			copy(clone.ConnectorOptions.Filters, ast.ConnectorOptions.Filters)
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

	// Note: tools field is deprecated, now handled by MCP

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
	if v, ok := data["disable_global_prompts"].(bool); ok {
		ast.DisableGlobalPrompts = v
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
	if v, ok := data["source"].(string); ok {
		ast.Source = v
	}

	// ConnectorOptions
	if v, has := data["connector_options"]; has {
		connOpts, err := store.ToConnectorOptions(v)
		if err != nil {
			return err
		}
		ast.ConnectorOptions = connOpts
	}

	// PromptPresets
	if v, has := data["prompt_presets"]; has {
		presets, err := store.ToPromptPresets(v)
		if err != nil {
			return err
		}
		ast.PromptPresets = presets
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
