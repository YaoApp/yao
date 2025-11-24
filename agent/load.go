package agent

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/agent/api"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	mongoStore "github.com/yaoapp/yao/agent/store/mongo"
	redisStore "github.com/yaoapp/yao/agent/store/redis"
	store "github.com/yaoapp/yao/agent/store/types"
	xunStore "github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/agent/types"
	"github.com/yaoapp/yao/config"
)

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

	// Initialize Agent API
	api.Agent = &api.API{DSL: &setting}

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

	// Initialize Assistant
	err = initAssistant()
	if err != nil {
		return err
	}

	return nil
}

// GetAgent returns the Agent instance
func GetAgent() *api.API {
	if api.Agent == nil {
		exception.New("Agent is not initialized", 500).Throw()
	}
	return api.Agent
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

	var models map[string]assistant.ModelCapabilities = map[string]assistant.ModelCapabilities{}
	err = application.Parse("models.yml", bytes, &models)
	if err != nil {
		return err
	}

	api.Agent.DSL.Models = models
	return nil
}

// initStore initialize the store
func initStore() error {

	var err error
	if api.Agent.DSL.StoreSetting.Connector == "default" || api.Agent.DSL.StoreSetting.Connector == "" {
		api.Agent.DSL.Store, err = xunStore.NewXun(api.Agent.DSL.StoreSetting)
		return err
	}

	// other connector
	conn, err := connector.Select(api.Agent.DSL.StoreSetting.Connector)
	if err != nil {
		return fmt.Errorf("load connectors error: %s", err.Error())
	}

	if conn.Is(connector.DATABASE) {
		api.Agent.DSL.Store, err = xunStore.NewXun(api.Agent.DSL.StoreSetting)
		return err

	} else if conn.Is(connector.REDIS) {
		api.Agent.DSL.Store = redisStore.NewRedis()
		return nil

	} else if conn.Is(connector.MONGO) {
		api.Agent.DSL.Store = mongoStore.NewMongo()
		return nil
	}

	return fmt.Errorf("Agent store connector %s not support", api.Agent.DSL.StoreSetting.Connector)
}

// initAssistant initialize the assistant
func initAssistant() error {

	// Set Storage
	assistant.SetStorage(api.Agent.DSL.Store)

	// Assistant Vision
	if api.Agent.DSL.Vision != nil {
		assistant.SetVision(api.Agent.DSL.Vision)
	}

	// Set global Uses configuration
	if api.Agent.DSL.Uses != nil {
		globalUses := &context.Uses{
			Vision: api.Agent.DSL.Uses.Vision,
			Audio:  api.Agent.DSL.Uses.Audio,
			Search: api.Agent.DSL.Uses.Search,
			Fetch:  api.Agent.DSL.Uses.Fetch,
		}
		assistant.SetGlobalUses(globalUses)
	}

	if api.Agent.DSL.Models != nil {
		assistant.SetModelCapabilities(api.Agent.DSL.Models)
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

	api.Agent.DSL.Assistant = defaultAssistant
	return nil
}

// defaultAssistant get the default assistant
func defaultAssistant() (*assistant.Assistant, error) {
	if api.Agent.DSL.Uses == nil || api.Agent.DSL.Uses.Default == "" {
		return nil, fmt.Errorf("default assistant not found")
	}
	return assistant.Get(api.Agent.DSL.Uses.Default)
}
