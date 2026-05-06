package assistant

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	gouOpenAI "github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/i18n"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/data"
	"gopkg.in/yaml.v3"
)

// systemAgents defines the system agents loaded from bindata
// These are internal agents used by the system (e.g., keyword extraction, querydsl generation)
// The directory name is without __yao. prefix, prefix is added during loading
// Format: directory name -> bindata path prefix
var systemAgents = []string{
	"keyword",
	"querydsl",
	"title",
	"prompt",
	"robot_prompt",
	"needsearch",
	"entity",
	"vision",
	"fetch",
	"loop_fallback",
}

// SystemConfig holds the system agents connector configuration
// This is set from agent.yml system block
type SystemConfig struct {
	// Role-level defaults (consumed by buildSystemRoles → SetDefaults)
	Default string // Default connector for the "default" role
	Light   string // Default connector for the "light" role
	Vision  string // Default connector for the "vision" role
	Audio   string // Default connector for the "audio" role
	Heavy   string // Default connector for the "heavy" role (complex reasoning)

	// Per-agent overrides (consumed by resolveSystemConnector → ast.Connector)
	Keyword      string // Connector for __yao.keyword agent
	QueryDSL     string // Connector for __yao.querydsl agent
	Title        string // Connector for __yao.title agent
	Prompt       string // Connector for __yao.prompt agent
	RobotPrompt  string // Connector for __yao.robot_prompt agent
	NeedSearch   string // Connector for __yao.needsearch agent
	Entity       string // Connector for __yao.entity agent
	LoopFallback string // Connector for __yao.loop_fallback agent
}

// systemConfig holds the system agents configuration (global variable like others in load.go)
var systemConfig *SystemConfig = nil

// SetSystemConfig sets the system agents configuration
func SetSystemConfig(config *SystemConfig) {
	systemConfig = config
}

// GetSystemConfig returns the system agents configuration
func GetSystemConfig() *SystemConfig {
	return systemConfig
}

// LoadSystemAgents loads the system agents from bindata
// These are internal agents like __yao.keyword and __yao.querydsl
// They are loaded before application assistants
// Behavior is same as LoadBuiltIn, just reads from bindata instead of filesystem
func LoadSystemAgents() error {

	// Get all existing system agents (for cleanup)
	deletedSystem := map[string]bool{}
	if storage != nil {
		// System agents have "system" tag
		tags := []string{"system"}
		builtIn := true
		res, err := storage.GetAssistants(store.AssistantFilter{
			Tags:    tags,
			BuiltIn: &builtIn,
			Select:  []string{"assistant_id", "id"},
		})
		if err != nil {
			log.Warn("Failed to get existing system agents: %v", err)
		} else {
			for _, assistant := range res.Data {
				deletedSystem[assistant.ID] = true
			}
		}
	}

	sort := 1
	for _, name := range systemAgents {
		// Build agent ID with __yao. prefix
		id := "__yao." + name
		pathPrefix := "yao/assistants/" + name

		assistant, err := loadSystemAgent(id, pathPrefix)
		if err != nil {
			log.Warn("Failed to load system agent %s: %v", id, err)
			continue
		}

		// Set sort order
		if assistant.Sort == 0 {
			assistant.Sort = sort
		}

		// Save to storage
		if err := assistant.Save(); err != nil {
			log.Warn("Failed to save system agent %s: %v", id, err)
			continue
		}

		// Initialize the assistant
		if err := assistant.initialize(); err != nil {
			log.Warn("Failed to initialize system agent %s: %v", id, err)
			continue
		}

		sort++
		loaded.Put(assistant)
		log.Trace("Loaded system agent: %s", id)

		// Remove from deleted list
		delete(deletedSystem, id)
	}

	// Remove deleted system agents
	if len(deletedSystem) > 0 {
		assistantIDs := []string{}
		for assistantID := range deletedSystem {
			assistantIDs = append(assistantIDs, assistantID)
		}
		if _, err := storage.DeleteAssistants(store.AssistantFilter{AssistantIDs: assistantIDs}); err != nil {
			log.Warn("Failed to delete obsolete system agents: %v", err)
		}
	}

	return nil
}

// loadSystemAgent loads a single system agent from bindata
// This follows the same pattern as LoadPath but reads from bindata
func loadSystemAgent(id, pathPrefix string) (*Assistant, error) {
	// Read package.yao from bindata
	pkgPath := pathPrefix + "/package.yao"
	pkgContent, err := data.Read(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", pkgPath, err)
	}

	// Parse package.yao
	var pkgData map[string]interface{}
	if err := application.Parse(pkgPath, pkgContent, &pkgData); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", pkgPath, err)
	}

	// Set assistant_id (no path - system agents are loaded from storage, not filesystem)
	pkgData["assistant_id"] = id

	// Set type if not specified
	if _, has := pkgData["type"]; !has {
		pkgData["type"] = "assistant"
	}

	// Override connector only if agent.yml has an explicit per-agent setting
	if override := resolveSystemConnector(id); override != "" {
		pkgData["connector"] = override
	}

	// Read prompts.yml from bindata (default prompts)
	promptsPath := pathPrefix + "/prompts.yml"
	promptsContent, err := data.Read(promptsPath)
	if err == nil {
		var prompts []store.Prompt
		if err := yaml.Unmarshal(promptsContent, &prompts); err == nil && len(prompts) > 0 {
			pkgData["prompts"] = prompts
		}
	}

	// Read prompt_presets from prompts directory
	presets := loadSystemPromptPresets(pathPrefix)
	if len(presets) > 0 {
		pkgData["prompt_presets"] = presets
	}

	// Load scripts from src directory (hook script source and other scripts sources)
	// These will be compiled by loadMap -> LoadScriptsFromData
	hookScriptSource, scriptsSource := loadSystemScripts(pathPrefix)
	if hookScriptSource != "" {
		pkgData["script"] = hookScriptSource
	}
	if len(scriptsSource) > 0 {
		pkgData["scripts"] = scriptsSource
	}

	// Read locales
	locales, err := loadSystemLocales(pathPrefix)
	if err == nil && len(locales) > 0 {
		pkgData["locales"] = locales
	}

	// Mark as system agent
	pkgData["readonly"] = true
	pkgData["built_in"] = true
	pkgData["tags"] = []string{"system"}

	// Load from map (same as LoadPath, includes initialize())
	return loadMap(pkgData)
}

// resolveSystemConnector returns an explicit per-agent connector override from agent.yml.
// Returns empty string if no override exists, so the connector declared in package.yao
// (e.g. "use::light") is preserved as-is.
func resolveSystemConnector(agentID string) string {
	if systemConfig == nil {
		return ""
	}
	switch agentID {
	case "__yao.keyword":
		return systemConfig.Keyword
	case "__yao.querydsl":
		return systemConfig.QueryDSL
	case "__yao.title":
		return systemConfig.Title
	case "__yao.prompt":
		return systemConfig.Prompt
	case "__yao.robot_prompt":
		return systemConfig.RobotPrompt
	case "__yao.needsearch":
		return systemConfig.NeedSearch
	case "__yao.entity":
		return systemConfig.Entity
	case "__yao.vision":
		return systemConfig.Vision
	case "__yao.audio":
		return systemConfig.Audio
	case "__yao.loop_fallback":
		return systemConfig.LoopFallback
	}
	return ""
}

// findCapableConnector finds the first connector that supports tool calling
func findCapableConnector() string {
	for id, conn := range connector.Connectors {
		if !conn.Is(connector.OPENAI) {
			continue
		}

		if connOpenAI, ok := conn.(*gouOpenAI.Connector); ok {
			if connOpenAI.Options.Capabilities != nil && connOpenAI.Options.Capabilities.ToolCalls {
				return id
			}
		}
	}

	return ""
}

// loadSystemPromptPresets loads prompt presets from bindata prompts directory
func loadSystemPromptPresets(pathPrefix string) map[string][]store.Prompt {
	presets := make(map[string][]store.Prompt)
	promptsDir := pathPrefix + "/prompts"

	// Try common preset files
	presetFiles := []string{"chat.yml", "task.yml", "code.yml", "analysis.yml"}
	for _, filename := range presetFiles {
		presetPath := promptsDir + "/" + filename
		content, err := data.Read(presetPath)
		if err != nil {
			continue
		}

		var prompts []store.Prompt
		if err := yaml.Unmarshal(content, &prompts); err == nil && len(prompts) > 0 {
			presetName := strings.TrimSuffix(filename, ".yml")
			presets[presetName] = prompts
		}
	}

	return presets
}

// loadSystemScripts loads scripts source from bindata src directory
// Returns hook script source and other scripts sources (as strings)
// These will be compiled by loadMap -> LoadScriptsFromData
func loadSystemScripts(pathPrefix string) (string, map[string]string) {
	srcDir := pathPrefix + "/src"

	// Try to load hook script (index.ts)
	var hookScriptSource string
	indexPath := srcDir + "/index.ts"
	indexContent, err := data.Read(indexPath)
	if err == nil && len(indexContent) > 0 {
		hookScriptSource = string(indexContent)
	}

	// Try to load other scripts
	scripts := make(map[string]string)
	scriptFiles := []string{"utils.ts", "helpers.ts", "tools.ts"}
	for _, filename := range scriptFiles {
		scriptPath := srcDir + "/" + filename
		content, err := data.Read(scriptPath)
		if err != nil {
			continue
		}

		scriptName := strings.TrimSuffix(filename, ".ts")
		scripts[scriptName] = string(content)
	}

	if len(scripts) == 0 {
		scripts = nil
	}

	return hookScriptSource, scripts
}

// loadSystemLocales loads locales from bindata
func loadSystemLocales(pathPrefix string) (i18n.Map, error) {
	locales := make(i18n.Map)

	// Try to load common locale files
	localeFiles := []string{"en-us.yml", "zh-cn.yml", "en.yml", "zh.yml"}
	localesDir := pathPrefix + "/locales"

	for _, filename := range localeFiles {
		localePath := filepath.Join(localesDir, filename)
		content, err := data.Read(localePath)
		if err != nil {
			continue
		}

		// Parse locale file
		locale := strings.TrimSuffix(filename, ".yml")
		var messages map[string]any
		if err := yaml.Unmarshal(content, &messages); err != nil {
			continue
		}

		locales[locale] = i18n.I18n{
			Locale:   locale,
			Messages: messages,
		}
	}

	return locales, nil
}
