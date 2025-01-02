package assistant

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/neo/store"
	"github.com/yaoapp/yao/share"
	"gopkg.in/yaml.v3"
)

// loaded the loaded assistant
var loaded = NewCache(200) // 200 is the default capacity
var storage store.Store = nil

// LoadBuiltIn load the built-in assistants
func LoadBuiltIn() error {
	root := `/assistants`
	app, err := fs.Get("app")
	if err != nil {
		return err
	}

	// Remove the built-in assistants
	if storage != nil {
		builtIn := true
		_, err := storage.DeleteAssistants(store.AssistantFilter{BuiltIn: &builtIn})
		if err != nil {
			return err
		}
	}

	// Check if the assistant is built-in
	if exists, _ := app.Exists(root); !exists {
		return nil
	}

	paths, err := app.ReadDir(root, true)
	if err != nil {
		return err
	}

	sort := 1
	for _, path := range paths {
		pkgfile := filepath.Join(path, "package.yao")
		if has, _ := app.Exists(pkgfile); !has {
			continue
		}

		assistant, err := LoadPath(path)
		if err != nil {
			return err
		}

		assistant.Readonly = true
		assistant.BuiltIn = true
		assistant.Sort = sort
		if assistant.Tags == nil {
			assistant.Tags = []string{"Built-in"}
		}

		sort++
		loaded.Put(assistant)

		// Save the assistant
		if storage != nil {
			_, err := storage.SaveAssistant(assistant.Map())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SetStorage set the storage
func SetStorage(s store.Store) {
	storage = s
}

// SetCache set the cache
func SetCache(capacity int) {
	ClearCache()
	loaded = NewCache(capacity)
}

// ClearCache clear the cache
func ClearCache() {
	if loaded != nil {
		loaded.Clear()
		loaded = nil
	}
}

// LoadStore create a new assistant from store
func LoadStore(id string) (*Assistant, error) {
	assistant, exists := loaded.Get(id)
	if exists {
		return assistant, nil
	}

	if storage == nil {
		return nil, fmt.Errorf("storage is not set")
	}

	data, err := storage.GetAssistant(id)
	if err != nil {
		return nil, err
	}

	// Load from path
	if data["path"] != nil {
		assistant, err = LoadPath(data["path"].(string))
		if err != nil {
			return nil, err
		}
		loaded.Put(assistant)
		return assistant, nil
	}

	// Load from store
	assistant, err = loadMap(data)
	if err != nil {
		return nil, err
	}

	loaded.Put(assistant)
	return assistant, nil
}

// LoadPath load assistant from path
func LoadPath(path string) (*Assistant, error) {
	app, err := fs.Get("app")
	if err != nil {
		return nil, err
	}

	pkgfile := filepath.Join(path, "package.yao")
	if has, _ := app.Exists(pkgfile); !has {
		return nil, fmt.Errorf("package.yao not found in %s", path)
	}

	pkg, err := app.ReadFile(pkgfile)
	if err != nil {
		return nil, err
	}

	id := strings.ReplaceAll(strings.TrimPrefix(path, "/assistants/"), "/", ".")
	var data map[string]interface{}
	err = jsoniter.Unmarshal(pkg, &data)
	if err != nil {
		return nil, err
	}

	// assistant_id
	data["assistant_id"] = id
	data["type"] = "assistant"
	data["path"] = path
	// prompts
	promptsfile := filepath.Join(path, "prompts.yml")
	if has, _ := app.Exists(promptsfile); has {
		prompts, err := loadPrompts(promptsfile, path)
		if err != nil {
			return nil, err
		}
		data["prompts"] = prompts
	}

	// load script
	scriptfile := filepath.Join(path, "src", "index.ts")
	if has, _ := app.Exists(scriptfile); has {
		script, err := loadScript(scriptfile, path)
		if err != nil {
			return nil, err
		}
		data["script"] = script
	}

	// load functions

	// load flow

	return loadMap(data)
}

func loadMap(data map[string]interface{}) (*Assistant, error) {

	assistant := &Assistant{}

	// assistant_id is required
	id, ok := data["assistant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("assistant_id is required")
	}
	assistant.ID = id

	// name is required
	name, ok := data["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}
	assistant.Name = name

	// avatar
	if avatar, ok := data["avatar"].(string); ok {
		assistant.Avatar = avatar
	}

	// Type
	if v, ok := data["type"].(string); ok {
		assistant.Type = v
	}

	// Mentionable
	if v, ok := data["mentionable"].(bool); ok {
		assistant.Mentionable = v
	}

	// Automated
	if v, ok := data["automated"].(bool); ok {
		assistant.Automated = v
	}

	// Readonly
	if v, ok := data["readonly"].(bool); ok {
		assistant.Readonly = v
	}

	// built_in
	if v, ok := data["built_in"].(bool); ok {
		assistant.BuiltIn = v
	}

	// sort
	if v, ok := data["sort"].(int); ok {
		assistant.Sort = v
	}

	// path
	if v, ok := data["path"].(string); ok {
		assistant.Path = v
	}

	// connector
	if connector, ok := data["connector"].(string); ok {
		assistant.Connector = connector
	}

	// tags
	if v, ok := data["tags"].([]string); ok {
		assistant.Tags = v
	}

	// options
	if v, ok := data["options"].(map[string]interface{}); ok {
		assistant.Options = v
	}

	// description
	if v, ok := data["description"].(string); ok {
		assistant.Description = v
	}

	// prompts
	if v, ok := data["prompts"].(string); ok {
		var prompts []Prompt
		err := yaml.Unmarshal([]byte(v), &prompts)
		if err != nil {
			return nil, err
		}
		assistant.Prompts = prompts
	}

	// script
	if data["script"] != nil {
		switch v := data["script"].(type) {
		case string:
			file := fmt.Sprintf("assistants/%s/src/index.ts", assistant.ID)
			script, err := loadScriptSource(v, file)
			if err != nil {
				return nil, err
			}
			assistant.Script = script
		case *v8.Script:
			assistant.Script = v
		}
	}

	return assistant, nil
}

func loadPrompts(file string, root string) (string, error) {

	app, err := fs.Get("app")
	if err != nil {
		return "", err
	}

	prompts, err := app.ReadFile(file)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`@assets/([^\s]+\.(md|yml|yaml|json|txt))`)
	prompts = re.ReplaceAllFunc(prompts, func(s []byte) []byte {
		asset := re.FindStringSubmatch(string(s))[1]
		assetFile := filepath.Join(root, "assets", asset)
		assetContent, err := app.ReadFile(assetFile)
		if err != nil {
			return []byte("")
		}
		// Add proper YAML formatting for content
		lines := strings.Split(string(assetContent), "\n")
		formattedContent := "|\n"
		for _, line := range lines {
			formattedContent += "    " + line + "\n"
		}
		return []byte(formattedContent)
	})

	return string(prompts), nil
}

func loadScript(file string, root string) (*v8.Script, error) {
	return v8.Load(file, share.ID(root, file))
}

func loadScriptSource(source string, file string) (*v8.Script, error) {
	script, err := v8.MakeScript([]byte(source), file, 5*time.Second, true)
	if err != nil {
		return nil, err
	}
	return script, nil
}

// Save save the assistant
func (ast *Assistant) Save() error {
	if storage == nil {
		return fmt.Errorf("storage is not set")
	}

	_, err := storage.SaveAssistant(ast.Map())
	return err
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
		"tags":         ast.Tags,
		"mentionable":  ast.Mentionable,
		"automated":    ast.Automated,
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
		API:         ast.API,
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
