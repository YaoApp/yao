package assistant

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"github.com/yaoapp/gou/application"
	gouOpenAI "github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openai"
	"gopkg.in/yaml.v3"
)

// loaded the loaded assistant
var loaded = NewCache(200) // 200 is the default capacity
var storage store.Store = nil
var storeSetting *store.Setting = nil // store setting from agent.yml
var modelCapabilities map[string]gouOpenAI.Capabilities = map[string]gouOpenAI.Capabilities{}
var defaultConnector string = ""                 // default connector
var globalUses *context.Uses = nil               // global uses configuration from agent.yml
var globalPrompts []store.Prompt = nil           // global prompts from agent/prompts.yml
var globalKBSetting *store.KBSetting = nil       // global KB setting from agent/kb.yml
var globalSearchConfig *searchTypes.Config = nil // global search config from agent/search.yml

// LoadBuiltIn load the built-in assistants
func LoadBuiltIn() error {

	// Clear non-system agents from cache (preserve system agents loaded by LoadSystemAgents)
	loaded.ClearExcept(func(id string) bool {
		return strings.HasPrefix(id, "__yao.") // Keep system agents
	})

	root := `/assistants`
	app, err := fs.Get("app")
	if err != nil {
		return err
	}

	// Get all existing built-in assistants
	deletedBuiltIn := map[string]bool{}

	// Remove the built-in assistants (exclude system agents with __yao. prefix)
	if storage != nil {

		builtIn := true
		res, err := storage.GetAssistants(store.AssistantFilter{BuiltIn: &builtIn, Select: []string{"assistant_id", "id"}})
		if err != nil {
			return err
		}

		// Get all existing built-in assistants (exclude system agents)
		for _, assistant := range res.Data {
			// Skip system agents (they are managed by LoadSystemAgents)
			if strings.HasPrefix(assistant.ID, "__yao.") {
				continue
			}
			deletedBuiltIn[assistant.ID] = true
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
		delete(deletedBuiltIn, assistant.ID)
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

// GetStorage returns the storage (for testing purposes)
func GetStorage() store.Store {
	return storage
}

// SetModelCapabilities set the model capabilities configuration
func SetModelCapabilities(capabilities map[string]gouOpenAI.Capabilities) {
	modelCapabilities = capabilities
}

// SetConnector set the connector
func SetConnector(c string) {
	defaultConnector = c
}

// SetGlobalUses set the global uses configuration
func SetGlobalUses(uses *context.Uses) {
	globalUses = uses
}

// SetGlobalPrompts set the global prompts from agent/prompts.yml
func SetGlobalPrompts(prompts []store.Prompt) {
	globalPrompts = prompts
}

// SetStoreSetting set the store setting from agent.yml
func SetStoreSetting(setting *store.Setting) {
	storeSetting = setting
}

// GetStoreSetting returns the store setting
func GetStoreSetting() *store.Setting {
	return storeSetting
}

// GetGlobalPrompts returns the global prompts with variables parsed
// ctx: context variables for parsing $CTX.* variables
func GetGlobalPrompts(ctx map[string]string) []store.Prompt {
	if len(globalPrompts) == 0 {
		return nil
	}
	return store.Prompts(globalPrompts).Parse(ctx)
}

// SetGlobalKBSetting set the global KB setting from agent/kb.yml
func SetGlobalKBSetting(kbSetting *store.KBSetting) {
	globalKBSetting = kbSetting
}

// GetGlobalKBSetting returns the global KB setting
func GetGlobalKBSetting() *store.KBSetting {
	return globalKBSetting
}

// SetGlobalSearchConfig set the global search config from agent/search.yml
func SetGlobalSearchConfig(config *searchTypes.Config) {
	globalSearchConfig = config
}

// GetGlobalSearchConfig returns the global search config
func GetGlobalSearchConfig() *searchTypes.Config {
	return globalSearchConfig
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

	// Request all fields when loading assistant from store
	storeModel, err := storage.GetAssistant(id, store.AssistantFullFields)
	if err != nil {
		return nil, err
	}

	// Load from path
	if storeModel.Path != "" {
		assistant, err = LoadPath(storeModel.Path)
		if err != nil {
			return nil, err
		}
		loaded.Put(assistant)
		return assistant, nil
	}

	// Create assistant from store model
	assistant = &Assistant{AssistantModel: *storeModel}

	// Load script from source field if present
	if assistant.Source != "" {
		script, err := loadSource(assistant.Source, assistant.ID)
		if err != nil {
			return nil, err
		}
		assistant.HookScript = script
	}

	// Initialize the assistant
	err = assistant.initialize()
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

	// prompts (default prompts from prompts.yml)
	promptsfile := filepath.Join(path, "prompts.yml")
	if has, _ := app.Exists(promptsfile); has {
		prompts, ts, err := store.LoadPrompts(promptsfile, path)
		if err != nil {
			return nil, err
		}
		data["prompts"] = prompts
		data["updated_at"] = ts
		updatedAt = ts
	}

	// prompt_presets (from prompts directory, key is filename without extension)
	promptsDir := filepath.Join(path, "prompts")
	if has, _ := app.Exists(promptsDir); has {
		presets, ts, err := store.LoadPromptPresets(promptsDir, path)
		if err != nil {
			return nil, err
		}
		if len(presets) > 0 {
			data["prompt_presets"] = presets
			updatedAt = max(updatedAt, ts)
		}
	}

	// load scripts (hook script and other scripts) from src directory
	srcDir := filepath.Join(path, "src")
	if has, _ := app.Exists(srcDir); has {
		hookScript, scripts, err := LoadScripts(srcDir)
		if err != nil {
			return nil, err
		}

		// Set hook script and update timestamp
		if hookScript != nil {
			data["script"] = hookScript
			// Get timestamp from index.ts if exists
			scriptfile := filepath.Join(srcDir, "index.ts")
			if ts, err := app.ModTime(scriptfile); err == nil {
				data["updated_at"] = max(updatedAt, ts.UnixNano())
			}
		}

		// Set other scripts
		if len(scripts) > 0 {
			data["scripts"] = scripts
		}
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
			assistant.Placeholder = &store.Placeholder{}
			err = jsoniter.Unmarshal(placeholder, assistant.Placeholder)
			if err != nil {
				return nil, err
			}

		case map[string]interface{}:
			raw, err := jsoniter.Marshal(vv)
			if err != nil {
				return nil, err
			}

			assistant.Placeholder = &store.Placeholder{}
			err = jsoniter.Unmarshal(raw, assistant.Placeholder)
			if err != nil {
				return nil, err
			}

		case *store.Placeholder:
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

	// modes
	if v, has := data["modes"]; has {
		modes, err := store.ToModes(v)
		if err != nil {
			return nil, err
		}
		assistant.Modes = modes
	}

	// default_mode
	if v, ok := data["default_mode"].(string); ok {
		assistant.DefaultMode = v
	}

	// DisableGlobalPrompts
	if v, ok := data["disable_global_prompts"].(bool); ok {
		assistant.DisableGlobalPrompts = v
	}

	// Readonly
	if v, ok := data["readonly"].(bool); ok {
		assistant.Readonly = v
	}

	// Public
	if v, ok := data["public"].(bool); ok {
		assistant.Public = v
	}

	// Share
	if v, ok := data["share"].(string); ok {
		assistant.Share = v
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

	// connector_options
	if connOpts, has := data["connector_options"]; has {
		opts, err := store.ToConnectorOptions(connOpts)
		if err != nil {
			return nil, err
		}
		assistant.ConnectorOptions = opts
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

		case string:
			assistant.Tags = []string{vv}

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
		flattened := locales.FlattenWithGlobal()

		// Auto-inject assistant name and description into all locales
		// so that {{name}} and {{description}} templates can be resolved
		for locale, i18nObj := range flattened {
			if i18nObj.Messages == nil {
				i18nObj.Messages = make(map[string]any)
			}
			// Add name and description if not already present
			if _, exists := i18nObj.Messages["name"]; !exists && assistant.Name != "" {
				i18nObj.Messages["name"] = assistant.Name
			}
			if _, exists := i18nObj.Messages["description"]; !exists && assistant.Description != "" {
				i18nObj.Messages["description"] = assistant.Description
			}
			flattened[locale] = i18nObj
		}

		i18n.Locales[id] = flattened
	} else {
		// No locales defined, create default with name and description for all common locales
		if assistant.Name != "" || assistant.Description != "" {
			defaultLocales := make(map[string]i18n.I18n)
			// Create entries for all common locales so {{name}} can be resolved
			commonLocales := []string{"en", "en-us", "zh", "zh-cn", "zh-tw"}
			for _, locale := range commonLocales {
				defaultLocales[locale] = i18n.I18n{
					Locale: locale,
					Messages: map[string]any{
						"name":        assistant.Name,
						"description": assistant.Description,
					},
				}
			}
			i18n.Locales[id] = defaultLocales
		}
	}

	// Search configuration (from package.yao search block)
	// This contains search options like web.max_results, kb.threshold, citation.format, etc.
	// Merge hierarchy: global config < assistant config
	switch v := data["search"].(type) {

	case *searchTypes.Config:
		assistant.Search = v

	case searchTypes.Config:
		assistant.Search = &v

	case map[string]interface{}:
		var assistantSearch searchTypes.Config
		raw, err := jsoniter.Marshal(v)
		if err != nil {
			return nil, err
		}
		err = jsoniter.Unmarshal(raw, &assistantSearch)
		if err != nil {
			return nil, err
		}
		// Merge with global search config
		assistant.Search = mergeSearchConfig(globalSearchConfig, &assistantSearch)

	default:
		assistant.Search = globalSearchConfig
	}

	// prompts
	if prompts, has := data["prompts"]; has {

		switch v := prompts.(type) {
		case []store.Prompt:
			assistant.Prompts = v

		case string:
			var prompts []store.Prompt
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

			var prompts []store.Prompt
			err = jsoniter.Unmarshal(raw, &prompts)
			if err != nil {
				return nil, err
			}
			assistant.Prompts = prompts
		}
	}

	// prompt_presets
	if presets, has := data["prompt_presets"]; has {
		promptPresets, err := store.ToPromptPresets(presets)
		if err != nil {
			return nil, err
		}
		assistant.PromptPresets = promptPresets
	}

	// source (hook script code) - store the source code
	if source, ok := data["source"].(string); ok {
		assistant.Source = source
	}

	// kb
	if kb, has := data["kb"]; has {
		knowledgeBase, err := store.ToKnowledgeBase(kb)
		if err != nil {
			return nil, err
		}
		assistant.KB = knowledgeBase
	}

	// db
	if db, has := data["db"]; has {
		database, err := store.ToDatabase(db)
		if err != nil {
			return nil, err
		}
		assistant.DB = database
	}

	// mcp
	if mcp, has := data["mcp"]; has {
		mcpServers, err := store.ToMCPServers(mcp)
		if err != nil {
			return nil, err
		}
		assistant.MCP = mcpServers
	}

	// workflow
	if workflow, has := data["workflow"]; has {
		wf, err := store.ToWorkflow(workflow)
		if err != nil {
			return nil, err
		}
		assistant.Workflow = wf
	}

	// uses (wrapper configurations for vision, audio, etc.)
	// Merge hierarchy: global uses < assistant uses
	if uses, has := data["uses"]; has {
		var assistantUses *context.Uses
		switch v := uses.(type) {
		case *context.Uses:
			assistantUses = v
		case context.Uses:
			assistantUses = &v
		default:
			raw, err := jsoniter.Marshal(v)
			if err != nil {
				return nil, err
			}
			var usesConfig context.Uses
			err = jsoniter.Unmarshal(raw, &usesConfig)
			if err != nil {
				return nil, err
			}
			assistantUses = &usesConfig
		}
		// Merge with global uses
		assistant.Uses = mergeUses(globalUses, assistantUses)
	} else if globalUses != nil {
		// No assistant-specific uses, use global
		assistant.Uses = globalUses
	}

	// Load scripts (hook script and other scripts)
	hookScript, scripts, scriptErr := LoadScriptsFromData(data, assistant.ID)
	if scriptErr != nil {
		return nil, scriptErr
	}
	assistant.HookScript = hookScript
	assistant.Scripts = scripts

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

	// Register scripts as process handlers
	if len(ast.Scripts) > 0 {
		if err := ast.RegisterScripts(); err != nil {
			return fmt.Errorf("failed to register scripts: %w", err)
		}
	}

	return nil
}

// mergeUses merges two Uses configs (base < override)
func mergeUses(base, override *context.Uses) *context.Uses {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := *base // Copy base

	// Override with non-empty values
	if override.Vision != "" {
		result.Vision = override.Vision
	}
	if override.Audio != "" {
		result.Audio = override.Audio
	}
	if override.Search != "" {
		result.Search = override.Search
	}
	if override.Fetch != "" {
		result.Fetch = override.Fetch
	}
	if override.Web != "" {
		result.Web = override.Web
	}
	if override.Keyword != "" {
		result.Keyword = override.Keyword
	}
	if override.QueryDSL != "" {
		result.QueryDSL = override.QueryDSL
	}
	if override.Rerank != "" {
		result.Rerank = override.Rerank
	}

	return &result
}

// mergeSearchConfig merges two search configs (base < override)
func mergeSearchConfig(base, override *searchTypes.Config) *searchTypes.Config {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := *base // Copy base

	// Merge Web config
	if override.Web != nil {
		if result.Web == nil {
			result.Web = override.Web
		} else {
			merged := *result.Web
			if override.Web.Provider != "" {
				merged.Provider = override.Web.Provider
			}
			if override.Web.APIKeyEnv != "" {
				merged.APIKeyEnv = override.Web.APIKeyEnv
			}
			if override.Web.MaxResults > 0 {
				merged.MaxResults = override.Web.MaxResults
			}
			result.Web = &merged
		}
	}

	// Merge KB config
	if override.KB != nil {
		if result.KB == nil {
			result.KB = override.KB
		} else {
			merged := *result.KB
			if len(override.KB.Collections) > 0 {
				merged.Collections = override.KB.Collections
			}
			if override.KB.Threshold > 0 {
				merged.Threshold = override.KB.Threshold
			}
			if override.KB.Graph {
				merged.Graph = override.KB.Graph
			}
			result.KB = &merged
		}
	}

	// Merge DB config
	if override.DB != nil {
		if result.DB == nil {
			result.DB = override.DB
		} else {
			merged := *result.DB
			if len(override.DB.Models) > 0 {
				merged.Models = override.DB.Models
			}
			if override.DB.MaxResults > 0 {
				merged.MaxResults = override.DB.MaxResults
			}
			result.DB = &merged
		}
	}

	// Merge Keyword config
	if override.Keyword != nil {
		if result.Keyword == nil {
			result.Keyword = override.Keyword
		} else {
			merged := *result.Keyword
			if override.Keyword.MaxKeywords > 0 {
				merged.MaxKeywords = override.Keyword.MaxKeywords
			}
			if override.Keyword.Language != "" {
				merged.Language = override.Keyword.Language
			}
			result.Keyword = &merged
		}
	}

	// Merge QueryDSL config
	if override.QueryDSL != nil {
		if result.QueryDSL == nil {
			result.QueryDSL = override.QueryDSL
		} else {
			merged := *result.QueryDSL
			if override.QueryDSL.Strict {
				merged.Strict = override.QueryDSL.Strict
			}
			result.QueryDSL = &merged
		}
	}

	// Merge Rerank config
	if override.Rerank != nil {
		if result.Rerank == nil {
			result.Rerank = override.Rerank
		} else {
			merged := *result.Rerank
			if override.Rerank.TopN > 0 {
				merged.TopN = override.Rerank.TopN
			}
			result.Rerank = &merged
		}
	}

	// Merge Citation config
	if override.Citation != nil {
		if result.Citation == nil {
			result.Citation = override.Citation
		} else {
			merged := *result.Citation
			if override.Citation.Format != "" {
				merged.Format = override.Citation.Format
			}
			// AutoInjectPrompt is a bool, so we check if it's explicitly set
			// by checking if the whole Citation block was provided
			merged.AutoInjectPrompt = override.Citation.AutoInjectPrompt
			if override.Citation.CustomPrompt != "" {
				merged.CustomPrompt = override.Citation.CustomPrompt
			}
			result.Citation = &merged
		}
	}

	// Merge Weights config
	if override.Weights != nil {
		if result.Weights == nil {
			result.Weights = override.Weights
		} else {
			merged := *result.Weights
			if override.Weights.User > 0 {
				merged.User = override.Weights.User
			}
			if override.Weights.Hook > 0 {
				merged.Hook = override.Weights.Hook
			}
			if override.Weights.Auto > 0 {
				merged.Auto = override.Weights.Auto
			}
			result.Weights = &merged
		}
	}

	// Merge Options config
	if override.Options != nil {
		if result.Options == nil {
			result.Options = override.Options
		} else {
			merged := *result.Options
			if override.Options.SkipThreshold > 0 {
				merged.SkipThreshold = override.Options.SkipThreshold
			}
			result.Options = &merged
		}
	}

	return &result
}
