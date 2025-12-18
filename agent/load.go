package agent

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	gouOpenAI "github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	searchDefaults "github.com/yaoapp/yao/agent/search/defaults"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	storeMongo "github.com/yaoapp/yao/agent/store/mongo"
	storeRedis "github.com/yaoapp/yao/agent/store/redis"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/agent/types"
	"github.com/yaoapp/yao/config"
)

var agentDSL *types.DSL

// Load load AIGC
func Load(cfg config.Config) error {

	setting := types.DSL{
		Cache: "__yao.agent.cache", // default is "__yao.agent.cache"
		StoreSetting: store.Setting{
			MaxSize: 20,
			TTL:     90 * 24 * 60 * 60, // 90 days in seconds
		},
	}

	bytes, err := application.App.Read(filepath.Join("agent", "agent.yml"))
	if err != nil {
		return err
	}

	err = application.Parse("agent.yml", bytes, &setting)
	if err != nil {
		return err
	}

	if setting.StoreSetting.MaxSize == 0 {
		setting.StoreSetting.MaxSize = 20 // default is 20
	}

	// Default Assistant, Agent is the developer name, Mohe is the brand name of the assistant
	if setting.Uses == nil {
		setting.Uses = &types.Uses{Default: "mohe"} // Agent is the developer name, Mohe is the brand name of the assistant
	}

	// Title Assistant
	if setting.Uses.Title == "" {
		setting.Uses.Title = setting.Uses.Default
	}

	// Prompt Assistant
	if setting.Uses.Prompt == "" {
		setting.Uses.Prompt = setting.Uses.Default
	}

	agentDSL = &setting

	// Store Setting
	err = initStore()
	if err != nil {
		return err
	}

	// Initialize model capabilities
	err = initModelCapabilities()
	if err != nil {
		return err
	}

	// Initialize Global I18n
	err = initGlobalI18n()
	if err != nil {
		return err
	}

	// Initialize Global Prompts
	err = initGlobalPrompts()
	if err != nil {
		return err
	}

	// Initialize KB Configuration
	err = initKBConfig()
	if err != nil {
		return err
	}

	// Initialize Search Configuration
	err = initSearchConfig()
	if err != nil {
		return err
	}

	// Initialize Assistant
	err = initAssistant()
	if err != nil {
		return err
	}

	return nil
}

// GetAgent returns the Agent settings
func GetAgent() *types.DSL {
	return agentDSL
}

// initGlobalI18n initialize the global i18n
func initGlobalI18n() error {
	locales, err := i18n.GetLocales("agent")
	if err != nil {
		return err
	}
	i18n.Locales["__global__"] = locales.Flatten()
	return nil
}

// initGlobalPrompts initialize the global prompts from agent/prompts.yml
func initGlobalPrompts() error {
	prompts, _, err := store.LoadGlobalPrompts()
	if err != nil {
		return err
	}
	agentDSL.GlobalPrompts = prompts
	return nil
}

// GetGlobalPrompts returns the global prompts
// ctx: context variables for parsing $CTX.* variables
func GetGlobalPrompts(ctx map[string]string) []store.Prompt {
	if agentDSL == nil || len(agentDSL.GlobalPrompts) == 0 {
		return nil
	}
	return store.Prompts(agentDSL.GlobalPrompts).Parse(ctx)
}

// initModelCapabilities initialize the model capabilities configuration
func initModelCapabilities() error {
	path := filepath.Join("agent", "models.yml")
	if exists, _ := application.App.Exists(path); !exists {
		return nil
	}

	// Read the model capabilities configuration
	bytes, err := application.App.Read(path)
	if err != nil {
		return err
	}

	var models map[string]gouOpenAI.Capabilities = map[string]gouOpenAI.Capabilities{}
	err = application.Parse("models.yml", bytes, &models)
	if err != nil {
		return err
	}

	agentDSL.Models = models
	return nil
}

// initStore initialize the store
func initStore() error {

	var err error
	if agentDSL.StoreSetting.Connector == "default" || agentDSL.StoreSetting.Connector == "" {
		agentDSL.Store, err = xun.NewXun(agentDSL.StoreSetting)
		return err
	}

	// other connector
	conn, err := connector.Select(agentDSL.StoreSetting.Connector)
	if err != nil {
		return fmt.Errorf("load connectors error: %s", err.Error())
	}

	if conn.Is(connector.DATABASE) {
		agentDSL.Store, err = xun.NewXun(agentDSL.StoreSetting)
		return err

	} else if conn.Is(connector.REDIS) {
		agentDSL.Store = storeRedis.NewRedis()
		return nil

	} else if conn.Is(connector.MONGO) {
		agentDSL.Store = storeMongo.NewMongo()
		return nil
	}

	return fmt.Errorf("Agent store connector %s not support", agentDSL.StoreSetting.Connector)
}

// initAssistant initialize the assistant
func initAssistant() error {

	// Set Storage
	assistant.SetStorage(agentDSL.Store)

	// Set Store Setting (MaxSize, TTL, etc.)
	assistant.SetStoreSetting(&agentDSL.StoreSetting)

	// Set global Uses configuration
	if agentDSL.Uses != nil {
		globalUses := &context.Uses{
			Vision:   agentDSL.Uses.Vision,
			Audio:    agentDSL.Uses.Audio,
			Search:   agentDSL.Uses.Search,
			Fetch:    agentDSL.Uses.Fetch,
			Web:      agentDSL.Uses.Web,
			Keyword:  agentDSL.Uses.Keyword,
			QueryDSL: agentDSL.Uses.QueryDSL,
			Rerank:   agentDSL.Uses.Rerank,
		}
		assistant.SetGlobalUses(globalUses)
	}

	// Set global prompts
	if len(agentDSL.GlobalPrompts) > 0 {
		assistant.SetGlobalPrompts(agentDSL.GlobalPrompts)
	}

	if agentDSL.Models != nil {
		assistant.SetModelCapabilities(agentDSL.Models)
	}

	if agentDSL.KB != nil {
		assistant.SetGlobalKBSetting(agentDSL.KB)
	}

	if agentDSL.Search != nil {
		assistant.SetGlobalSearchConfig(agentDSL.Search)
	}

	// Set system agents configuration
	if agentDSL.System != nil {
		assistant.SetSystemConfig(&assistant.SystemConfig{
			Default:    agentDSL.System.Default,
			Keyword:    agentDSL.System.Keyword,
			QueryDSL:   agentDSL.System.QueryDSL,
			Title:      agentDSL.System.Title,
			Prompt:     agentDSL.System.Prompt,
			NeedSearch: agentDSL.System.NeedSearch,
			Entity:     agentDSL.System.Entity,
		})
	}

	// Load System Agents (from bindata: __yao.keyword, __yao.querydsl, etc.)
	if err := assistant.LoadSystemAgents(); err != nil {
		return err
	}

	// Load Built-in Assistants (from application /assistants directory)
	err := assistant.LoadBuiltIn()
	if err != nil {
		return err
	}

	// Default Assistant
	defaultAssistant, err := defaultAssistant()
	if err != nil {
		return err
	}

	agentDSL.Assistant = defaultAssistant
	return nil
}

// initKBConfig initialize the knowledge base configuration from agent/kb.yml
func initKBConfig() error {
	path := filepath.Join("agent", "kb.yml")
	if exists, _ := application.App.Exists(path); !exists {
		return nil // KB config is optional
	}

	// Read the KB configuration
	bytes, err := application.App.Read(path)
	if err != nil {
		return err
	}

	var kbSetting store.KBSetting
	err = application.Parse("kb.yml", bytes, &kbSetting)
	if err != nil {
		return err
	}

	agentDSL.KB = &kbSetting
	return nil
}

// initSearchConfig initialize the search configuration from agent/search.yml
func initSearchConfig() error {
	// Start with system defaults
	agentDSL.Search = searchDefaults.SystemDefaults

	path := filepath.Join("agent", "search.yml")
	if exists, _ := application.App.Exists(path); !exists {
		return nil // Search config is optional, use defaults
	}

	// Read the search configuration
	bytes, err := application.App.Read(path)
	if err != nil {
		return err
	}

	var searchConfig searchTypes.Config
	err = application.Parse("search.yml", bytes, &searchConfig)
	if err != nil {
		return err
	}

	// Merge with defaults
	agentDSL.Search = mergeSearchConfig(searchDefaults.SystemDefaults, &searchConfig)
	return nil
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
			if override.Web.Provider != "" {
				result.Web.Provider = override.Web.Provider
			}
			if override.Web.APIKeyEnv != "" {
				result.Web.APIKeyEnv = override.Web.APIKeyEnv
			}
			if override.Web.MaxResults > 0 {
				result.Web.MaxResults = override.Web.MaxResults
			}
		}
	}

	// Merge KB config
	if override.KB != nil {
		if result.KB == nil {
			result.KB = override.KB
		} else {
			if len(override.KB.Collections) > 0 {
				result.KB.Collections = override.KB.Collections
			}
			if override.KB.Threshold > 0 {
				result.KB.Threshold = override.KB.Threshold
			}
			if override.KB.Graph {
				result.KB.Graph = override.KB.Graph
			}
		}
	}

	// Merge DB config
	if override.DB != nil {
		if result.DB == nil {
			result.DB = override.DB
		} else {
			if len(override.DB.Models) > 0 {
				result.DB.Models = override.DB.Models
			}
			if override.DB.MaxResults > 0 {
				result.DB.MaxResults = override.DB.MaxResults
			}
		}
	}

	// Merge Keyword config
	if override.Keyword != nil {
		if result.Keyword == nil {
			result.Keyword = override.Keyword
		} else {
			if override.Keyword.MaxKeywords > 0 {
				result.Keyword.MaxKeywords = override.Keyword.MaxKeywords
			}
			if override.Keyword.Language != "" {
				result.Keyword.Language = override.Keyword.Language
			}
		}
	}

	// Merge QueryDSL config
	if override.QueryDSL != nil {
		result.QueryDSL = override.QueryDSL
	}

	// Merge Rerank config
	if override.Rerank != nil {
		if result.Rerank == nil {
			result.Rerank = override.Rerank
		} else {
			if override.Rerank.TopN > 0 {
				result.Rerank.TopN = override.Rerank.TopN
			}
		}
	}

	// Merge Citation config
	if override.Citation != nil {
		if result.Citation == nil {
			result.Citation = override.Citation
		} else {
			if override.Citation.Format != "" {
				result.Citation.Format = override.Citation.Format
			}
			// AutoInjectPrompt is a bool, need to check if explicitly set
			result.Citation.AutoInjectPrompt = override.Citation.AutoInjectPrompt
			if override.Citation.CustomPrompt != "" {
				result.Citation.CustomPrompt = override.Citation.CustomPrompt
			}
		}
	}

	// Merge Weights config
	if override.Weights != nil {
		if result.Weights == nil {
			result.Weights = override.Weights
		} else {
			if override.Weights.User > 0 {
				result.Weights.User = override.Weights.User
			}
			if override.Weights.Hook > 0 {
				result.Weights.Hook = override.Weights.Hook
			}
			if override.Weights.Auto > 0 {
				result.Weights.Auto = override.Weights.Auto
			}
		}
	}

	// Merge Options config
	if override.Options != nil {
		if result.Options == nil {
			result.Options = override.Options
		} else {
			if override.Options.SkipThreshold > 0 {
				result.Options.SkipThreshold = override.Options.SkipThreshold
			}
		}
	}

	return &result
}

// GetSearchConfig returns the global search configuration
func GetSearchConfig() *searchTypes.Config {
	if agentDSL == nil {
		return searchDefaults.SystemDefaults
	}
	return agentDSL.Search
}

// defaultAssistant get the default assistant
func defaultAssistant() (*assistant.Assistant, error) {
	if agentDSL.Uses == nil || agentDSL.Uses.Default == "" {
		return nil, fmt.Errorf("default assistant not found")
	}
	return assistant.Get(agentDSL.Uses.Default)
}
