package assistant

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/rag/driver"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/store"
	agentvision "github.com/yaoapp/yao/agent/vision"
	"github.com/yaoapp/yao/openai"
	"github.com/yaoapp/yao/share"
	"gopkg.in/yaml.v3"
)

// loaded the loaded assistant
var loaded = NewCache(200) // 200 is the default capacity
var storage store.Store = nil
var rag *RAG = nil
var search interface{} = nil
var connectorSettings map[string]ConnectorSetting = map[string]ConnectorSetting{}
var vision *agentvision.Vision = nil
var defaultConnector string = "" // default connector

// LoadBuiltIn load the built-in assistants
func LoadBuiltIn() error {

	// Clear the cache
	loaded.Clear()

	root := `/assistants`
	app, err := fs.Get("app")
	if err != nil {
		return err
	}

	// Get all existing built-in assistants
	deletedBuiltIn := map[string]bool{}

	// Remove the built-in assistants
	if storage != nil {

		builtIn := true
		res, err := storage.GetAssistants(store.AssistantFilter{BuiltIn: &builtIn, Select: []string{"assistant_id", "id"}})
		if err != nil {
			return err
		}

		// Get all existing built-in assistants
		for _, assistant := range res.Data {
			assistantID := assistant["assistant_id"].(string)
			deletedBuiltIn[assistantID] = true
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
		if assistant.Sort == 0 {
			assistant.Sort = sort
		}
		if assistant.Tags == nil {
			assistant.Tags = []string{}
		}

		// Save the assistant
		err = assistant.Save()
		if err != nil {
			return err
		}

		// Initialize the assistant
		err = assistant.initialize()
		if err != nil {
			return err
		}

		sort++
		loaded.Put(assistant)

		// Remove the built-in assistant from the store
		if _, ok := deletedBuiltIn[assistant.ID]; ok {
			delete(deletedBuiltIn, assistant.ID)
		}
	}

	// Remove deleted built-in assistants
	if len(deletedBuiltIn) > 0 {
		assistantIDs := []string{}
		for assistantID := range deletedBuiltIn {
			assistantIDs = append(assistantIDs, assistantID)
		}
		_, err := storage.DeleteAssistants(store.AssistantFilter{AssistantIDs: assistantIDs})
		if err != nil {
			return err
		}
	}

	return nil
}

// SetStorage set the storage
func SetStorage(s store.Store) {
	storage = s
}

// SetVision set the vision
func SetVision(v *agentvision.Vision) {
	vision = v
}

// SetConnectorSettings set the connector settings
func SetConnectorSettings(settings map[string]ConnectorSetting) {
	connectorSettings = settings
}

// SetConnector set the connector
func SetConnector(c string) {
	defaultConnector = c
}

// SetRAG set the RAG engine
// e: the RAG engine
// u: the RAG file uploader
// v: the RAG vectorizer
func SetRAG(e driver.Engine, u driver.FileUpload, v driver.Vectorizer, setting RAGSetting) {
	rag = &RAG{
		Engine:     e,
		Uploader:   u,
		Vectorizer: v,
		Setting:    setting,
	}
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

// GetCache returns the loaded cache
func GetCache() *Cache {
	return loaded
}

// LoadStore create a new assistant from store
func LoadStore(id string) (*Assistant, error) {

	if id == "" {
		return nil, fmt.Errorf("assistant_id is required")
	}

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

// loadPackage loads and parses the package.yao file
func loadPackage(path string) (map[string]interface{}, error) {
	app, err := fs.Get("app")
	if err != nil {
		return nil, err
	}

	pkgfile := filepath.Join(path, "package.yao")
	if has, _ := app.Exists(pkgfile); !has {
		return nil, fmt.Errorf("package.yao not found in %s", path)
	}

	pkgraw, err := app.ReadFile(pkgfile)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = application.Parse(pkgfile, pkgraw, &data)
	if err != nil {
		return nil, err
	}

	// Process connector environment variable
	if connector, ok := data["connector"].(string); ok {
		if strings.HasPrefix(connector, "$ENV.") {
			envKey := strings.TrimPrefix(connector, "$ENV.")
			if envValue := os.Getenv(envKey); envValue != "" {
				data["connector"] = envValue
			}
		}
	}

	return data, nil
}

// LoadPath load assistant from path
func LoadPath(path string) (*Assistant, error) {
	app, err := fs.Get("app")
	if err != nil {
		return nil, err
	}

	data, err := loadPackage(path)
	if err != nil {
		return nil, err
	}

	// assistant_id
	id := strings.ReplaceAll(strings.TrimPrefix(path, "/assistants/"), "/", ".")
	data["assistant_id"] = id
	data["path"] = path
	if _, has := data["type"]; !has {
		data["type"] = "assistant"
	}

	updatedAt := int64(0)

	// prompts
	promptsfile := filepath.Join(path, "prompts.yml")
	if has, _ := app.Exists(promptsfile); has {
		prompts, ts, err := loadPrompts(promptsfile, path)
		if err != nil {
			return nil, err
		}
		data["prompts"] = prompts
		data["updated_at"] = ts
		updatedAt = ts
	}

	// load script
	scriptfile := filepath.Join(path, "src", "index.ts")
	if has, _ := app.Exists(scriptfile); has {
		script, ts, err := loadScript(scriptfile, path)
		if err != nil {
			return nil, err
		}
		data["script"] = script
		data["updated_at"] = max(updatedAt, ts)
	}

	// load tools
	toolsfile := filepath.Join(path, "tools.yao")
	if has, _ := app.Exists(toolsfile); has {
		tools, ts, err := loadTools(toolsfile)
		if err != nil {
			return nil, err
		}
		data["tools"] = tools
		updatedAt = max(updatedAt, ts)
	}

	// i18ns
	locales, err := i18n.GetLocales(path)
	if err != nil {
		return nil, err
	}
	data["locales"] = locales
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

	// Placeholder
	if v, ok := data["placeholder"]; ok {

		switch vv := v.(type) {
		case string:
			placeholder, err := jsoniter.Marshal(vv)
			if err != nil {
				return nil, err
			}
			assistant.Placeholder = &Placeholder{}
			err = jsoniter.Unmarshal(placeholder, assistant.Placeholder)
			if err != nil {
				return nil, err
			}

		case map[string]interface{}:
			raw, err := jsoniter.Marshal(vv)
			if err != nil {
				return nil, err
			}

			assistant.Placeholder = &Placeholder{}
			err = jsoniter.Unmarshal(raw, assistant.Placeholder)
			if err != nil {
				return nil, err
			}

		case *Placeholder:
			assistant.Placeholder = vv

		case nil:
			assistant.Placeholder = nil
		}
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
	if v, has := data["sort"]; has {
		assistant.Sort = cast.ToInt(v)
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
	if v, has := data["tags"]; has {
		switch vv := v.(type) {
		case []string:
			assistant.Tags = vv
		case []interface{}:
			var tags []string
			for _, tag := range vv {
				tags = append(tags, cast.ToString(tag))
			}
			assistant.Tags = tags

		case interface{}:
			raw, err := jsoniter.Marshal(vv)
			if err != nil {
				return nil, err
			}
			var tags []string
			err = jsoniter.Unmarshal(raw, &tags)
			if err != nil {
				return nil, err
			}
			assistant.Tags = tags

		case string:
			assistant.Tags = []string{vv}
		}
	}

	// options
	if v, ok := data["options"].(map[string]interface{}); ok {
		assistant.Options = v
	}

	// description
	if v, ok := data["description"].(string); ok {
		assistant.Description = v
	}

	// locales
	if locales, ok := data["locales"].(i18n.Map); ok {
		assistant.Locales = locales
		i18n.Locales[id] = locales.FlattenWithGlobal()
	}

	// Search options
	if v, ok := data["search"].(map[string]interface{}); ok {
		assistant.Search = &SearchOption{}
		raw, err := jsoniter.Marshal(v)
		if err != nil {
			return nil, err
		}

		// Unmarshal the raw data
		err = jsoniter.Unmarshal(raw, assistant.Search)
		if err != nil {
			return nil, err
		}
	}

	// Knowledge options
	if v, ok := data["knowledge"].(map[string]interface{}); ok {
		assistant.Knowledge = &KnowledgeOption{}
		raw, err := jsoniter.Marshal(v)
		if err != nil {
			return nil, err
		}
		// Unmarshal the raw data
		err = jsoniter.Unmarshal(raw, assistant.Knowledge)
		if err != nil {
			return nil, err
		}
	}

	// prompts
	if prompts, has := data["prompts"]; has {

		switch v := prompts.(type) {
		case []Prompt:
			assistant.Prompts = v

		case string:
			var prompts []Prompt
			err := yaml.Unmarshal([]byte(v), &prompts)
			if err != nil {
				return nil, err
			}
			assistant.Prompts = prompts

		default:
			raw, err := jsoniter.Marshal(v)
			if err != nil {
				return nil, err
			}

			var prompts []Prompt
			err = jsoniter.Unmarshal(raw, &prompts)
			if err != nil {
				return nil, err
			}
			assistant.Prompts = prompts
		}
	}

	// tools
	if tools, has := data["tools"]; has {
		switch vv := tools.(type) {
		case []Tool:
			assistant.Tools = &ToolCalls{
				Tools:   vv,
				Prompts: assistant.Prompts,
			}

		case ToolCalls:
			assistant.Tools = &vv

		default:
			raw, err := jsoniter.Marshal(tools)
			if err != nil {
				return nil, fmt.Errorf("tools format error %s", err.Error())
			}

			var tools ToolCalls
			err = jsoniter.Unmarshal(raw, &tools)
			if err != nil {
				return nil, fmt.Errorf("tools format error %s", err.Error())
			}
			assistant.Tools = &tools
		}
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

	// created_at
	if v, has := data["created_at"]; has {
		ts, err := getTimestamp(v)
		if err != nil {
			return nil, err
		}
		assistant.CreatedAt = ts
	}

	// updated_at
	if v, has := data["updated_at"]; has {
		ts, err := getTimestamp(v)
		if err != nil {
			return nil, err
		}
		assistant.UpdatedAt = ts
	}

	// Initialize the assistant
	err := assistant.initialize()
	if err != nil {
		return nil, err
	}

	return assistant, nil
}

func loadPrompts(file string, root string) (string, int64, error) {

	app, err := fs.Get("app")
	if err != nil {
		return "", 0, err
	}

	ts, err := app.ModTime(file)
	if err != nil {
		return "", 0, err
	}

	prompts, err := app.ReadFile(file)
	if err != nil {
		return "", 0, err
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

	return string(prompts), ts.UnixNano(), nil
}

func loadScript(file string, root string) (*v8.Script, int64, error) {

	app, err := fs.Get("app")
	if err != nil {
		return nil, 0, err
	}

	ts, err := app.ModTime(file)
	if err != nil {
		return nil, 0, err
	}

	script, err := v8.Load(file, share.ID(root, file))
	if err != nil {
		return nil, 0, err
	}

	return script, ts.UnixNano(), nil
}

func loadScriptSource(source string, file string) (*v8.Script, error) {
	script, err := v8.MakeScript([]byte(source), file, 5*time.Second, true)
	if err != nil {
		return nil, err
	}
	return script, nil
}

// Init init the assistant
// Choose the connector and initialize the assistant
func (ast *Assistant) initialize() error {

	conn := defaultConnector
	if ast.Connector != "" {
		conn = ast.Connector
	}
	ast.Connector = conn

	api, err := openai.New(conn)
	if err != nil {
		return err
	}
	ast.openai = api

	// Check if the assistant supports vision
	model := api.Model()
	if v, ok := ast.Options["model"].(string); ok {
		model = strings.TrimLeft(v, "moapi:")
	}
	if _, ok := VisionCapableModels[model]; ok {
		ast.vision = true
	}

	// Check if the assistant has an init hook
	if ast.Script != nil {
		scriptCtx, err := ast.Script.NewContext("", nil)
		if err != nil {
			return err
		}
		defer scriptCtx.Close()
		ast.initHook = scriptCtx.Global().Has("init")
	}

	return nil
}

func loadTools(file string) (*ToolCalls, int64, error) {

	app, err := fs.Get("app")
	if err != nil {
		return nil, 0, err
	}

	content, err := app.ReadFile(file)
	if err != nil {
		return nil, 0, err
	}

	ts, err := app.ModTime(file)
	if err != nil {
		return nil, 0, err
	}

	if len(content) == 0 {
		return &ToolCalls{Tools: []Tool{}, Prompts: []Prompt{}}, ts.UnixNano(), nil
	}

	var tools ToolCalls
	err = application.Parse(file, content, &tools)
	if err != nil {
		return nil, 0, err
	}

	return &tools, ts.UnixNano(), nil
}
