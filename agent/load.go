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

	// Set global Uses configuration
	if agentDSL.Uses != nil {
		globalUses := &context.Uses{
			Vision: agentDSL.Uses.Vision,
			Audio:  agentDSL.Uses.Audio,
			Search: agentDSL.Uses.Search,
			Fetch:  agentDSL.Uses.Fetch,
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

	// Load Built-in Assistants
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

// defaultAssistant get the default assistant
func defaultAssistant() (*assistant.Assistant, error) {
	if agentDSL.Uses == nil || agentDSL.Uses.Default == "" {
		return nil, fmt.Errorf("default assistant not found")
	}
	return assistant.Get(agentDSL.Uses.Default)
}
