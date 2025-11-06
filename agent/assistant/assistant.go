package assistant

import (
	"fmt"
	"path"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	sui "github.com/yaoapp/yao/sui/core"
)

// Save save the assistant
func (ast *Assistant) Save() error {
	if storage == nil {
		return fmt.Errorf("storage is not set")
	}

	_, err := storage.SaveAssistant(ast.Map())
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
		"avatar":       ast.Avatar,
		"connector":    ast.Connector,
		"path":         ast.Path,
		"built_in":     ast.BuiltIn,
		"sort":         ast.Sort,
		"description":  ast.Description,
		"options":      ast.Options,
		"prompts":      ast.Prompts,
		"tools":        ast.Tools,
		"tags":         ast.Tags,
		"mentionable":  ast.Mentionable,
		"automated":    ast.Automated,
		"placeholder":  ast.Placeholder,
		"locales":      ast.Locales,
		"created_at":   timeToMySQLFormat(ast.CreatedAt),
		"updated_at":   timeToMySQLFormat(ast.UpdatedAt),
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
		Mentionable: ast.Mentionable,
		Automated:   ast.Automated,
		Script:      ast.Script,
		openai:      ast.openai,
	}

	// Deep copy tags
	if ast.Tags != nil {
		clone.Tags = make([]string, len(ast.Tags))
		copy(clone.Tags, ast.Tags)
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
		clone.Prompts = make([]Prompt, len(ast.Prompts))
		copy(clone.Prompts, ast.Prompts)
	}

	// Deep copy tools
	if ast.Tools != nil {
		clone.Tools = &ToolCalls{}
		if ast.Tools.Tools != nil {
			clone.Tools.Tools = make([]Tool, len(ast.Tools.Tools))
			copy(clone.Tools.Tools, ast.Tools.Tools)
		}

		if ast.Tools.Prompts != nil {
			clone.Tools.Prompts = make([]Prompt, len(ast.Tools.Prompts))
			copy(clone.Tools.Prompts, ast.Tools.Prompts)
		}
	}

	// Deep copy workflow
	if ast.Workflow != nil {
		clone.Workflow = make(map[string]interface{})
		for k, v := range ast.Workflow {
			clone.Workflow[k] = v
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
		case []Tool:
			ast.Tools = &ToolCalls{
				Tools:   tools,
				Prompts: ast.Prompts,
			}

		case *ToolCalls:
			ast.Tools = tools

		default:
			raw, err := jsoniter.Marshal(tools)
			if err != nil {
				return err
			}
			ast.Tools = &ToolCalls{}
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
	if v, ok := data["tags"].([]string); ok {
		ast.Tags = v
	}
	if v, ok := data["options"].(map[string]interface{}); ok {
		ast.Options = v
	}

	return ast.Validate()
}
