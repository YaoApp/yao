package assistant

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/rag/driver"
	"github.com/yaoapp/kun/log"
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

	// Update Index in background
	go func() {
		err := ast.UpdateIndex()
		if err != nil {
			log.Error("failed to update index for assistant %s: %s", ast.ID, err)
			color.Red("failed to update index for assistant %s: %s", ast.ID, err)
		}
	}()

	return nil
}

// UpdateIndex update the index for RAG
func (ast *Assistant) UpdateIndex() error {

	// RAG is not enabled
	if rag == nil {
		return nil
	}

	if rag.Engine == nil {
		return fmt.Errorf("engine is not set")
	}

	// Update Index
	index := fmt.Sprintf("%sassistants", rag.Setting.IndexPrefix)
	id := fmt.Sprintf("assistant_%s", ast.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check if the index exists
	exists, err := rag.Engine.HasIndex(ctx, index)
	if err != nil {
		return err
	}

	// Create the index if it does not exist
	if !exists {
		ctxCreate, cancelCreate := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelCreate()
		err = rag.Engine.CreateIndex(ctxCreate, driver.IndexConfig{Name: index})
		if err != nil {
			return err
		}
	}

	// Check if the document exists
	exists, err = rag.Engine.HasDocument(ctx, index, id)
	if err != nil {
		return err
	}

	// Check if the document is updated
	if exists {
		metadata, err := rag.Engine.GetMetadata(ctx, index, id)
		if err != nil {
			return err
		}

		if v, ok := metadata["updated_at"].(string); ok {
			updatedAt, err := stringToTimestamp(v)
			if err != nil {
				return err
			}
			if updatedAt >= ast.UpdatedAt {
				return nil
			}
		}
	}

	// Update the index
	content, err := jsoniter.MarshalToString(ast.Map())
	if err != nil {
		return err
	}

	metadata := map[string]interface{}{
		"assistant_id": ast.ID,
		"type":         ast.Type,
		"name":         ast.Name,
		"updated_at":   fmt.Sprintf("%d", ast.UpdatedAt),
	}

	return rag.Engine.IndexDoc(ctx, index, &driver.Document{
		DocID:    id,
		Content:  content,
		Metadata: metadata,
	})
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
		"functions":    ast.Functions,
		"tags":         ast.Tags,
		"mentionable":  ast.Mentionable,
		"automated":    ast.Automated,
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

	// Deep copy flows
	if ast.Flows != nil {
		clone.Flows = make([]map[string]interface{}, len(ast.Flows))
		for i, flow := range ast.Flows {
			cloneFlow := make(map[string]interface{})
			for k, v := range flow {
				cloneFlow[k] = v
			}
			clone.Flows[i] = cloneFlow
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
