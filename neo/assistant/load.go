package assistant

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/rag/driver"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/neo/store"
	neovision "github.com/yaoapp/yao/neo/vision"
	"github.com/yaoapp/yao/openai"
	"github.com/yaoapp/yao/share"
	"gopkg.in/yaml.v3"
)

// loaded the loaded assistant
var loaded = NewCache(200) // 200 is the default capacity
var storage store.Store = nil
var rag *RAG = nil
var vision *neovision.Vision = nil
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
		if assistant.Sort == 0 {
			assistant.Sort = sort
		}
		if assistant.Tags == nil {
			assistant.Tags = []string{"Built-in"}
		}

		// Check if the assistant has Built-in tag
		hasBuiltIn := false
		for _, tag := range assistant.Tags {
			if tag == "Built-in" {
				hasBuiltIn = true
				break
			}
		}

		// add Built-in tag if not exists
		if !hasBuiltIn {
			assistant.Tags = append(assistant.Tags, "Built-in")
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

	}

	return nil
}

// SetStorage set the storage
func SetStorage(s store.Store) {
	storage = s
}

// SetVision set the vision
func SetVision(v *neovision.Vision) {
	vision = v
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

	// load functions
	functionsfile := filepath.Join(path, "functions.json")
	if has, _ := app.Exists(functionsfile); has {
		functions, ts, err := loadFunctions(functionsfile)
		if err != nil {
			return nil, err
		}
		data["functions"] = functions
		updatedAt = max(updatedAt, ts)
		data["updated_at"] = updatedAt
	}

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

	// functions
	if funcs, has := data["functions"]; has {
		switch vv := funcs.(type) {
		case []Function:
			assistant.Functions = vv
		default:
			raw, err := jsoniter.Marshal(vv)
			if err != nil {
				return nil, err
			}
			var functions []Function
			err = jsoniter.Unmarshal(raw, &functions)
			if err != nil {
				return nil, err
			}
			assistant.Functions = functions
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

func loadFunctions(file string) ([]Function, int64, error) {

	app, err := fs.Get("app")
	if err != nil {
		return nil, 0, err
	}

	ts, err := app.ModTime(file)
	if err != nil {
		return nil, 0, err
	}

	raw, err := app.ReadFile(file)
	if err != nil {
		return nil, 0, err
	}

	var functions []Function
	err = jsoniter.Unmarshal(raw, &functions)
	if err != nil {
		return nil, 0, err
	}

	return functions, ts.UnixNano(), nil
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
