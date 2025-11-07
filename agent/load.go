package agent

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/i18n"
	mongoStore "github.com/yaoapp/yao/agent/store/mongo"
	redisStore "github.com/yaoapp/yao/agent/store/redis"
	store "github.com/yaoapp/yao/agent/store/types"
	xunStore "github.com/yaoapp/yao/agent/store/xun"
	"github.com/yaoapp/yao/config"
)

// Agent the agent AI assistant
var Agent *DSL

// Load load AIGC
func Load(cfg config.Config) error {

	setting := DSL{
		ID: "agent",
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
	if setting.Use == nil {
		setting.Use = &Use{Default: "mohe"} // Agent is the developer name, Mohe is the brand name of the assistant
	}

	// Title Assistant
	if setting.Use.Title == "" {
		setting.Use.Title = setting.Use.Default
	}

	// Prompt Assistant
	if setting.Use.Prompt == "" {
		setting.Use.Prompt = setting.Use.Default
	}

	Agent = &setting

	// Store Setting
	err = initStore()
	if err != nil {
		return err
	}

	// Initialize Connector settings
	err = initConnectorSettings()
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

// initGlobalI18n initialize the global i18n
func initGlobalI18n() error {
	locales, err := i18n.GetLocales("agent")
	if err != nil {
		return err
	}
	i18n.Locales["__global__"] = locales.Flatten()
	return nil
}

// initConnectors initialize the connectors
func initConnectorSettings() error {
	path := filepath.Join("agent", "connectors.yml")
	if exists, _ := application.App.Exists(path); !exists {
		return nil
	}

	// Open the connectors
	bytes, err := application.App.Read(path)
	if err != nil {
		return err
	}

	var connectors map[string]assistant.ConnectorSetting = map[string]assistant.ConnectorSetting{}
	err = application.Parse("connectors.yml", bytes, &connectors)
	if err != nil {
		return err
	}

	Agent.Connectors = connectors
	return nil
}

// initStore initialize the store
func initStore() error {

	var err error
	if Agent.StoreSetting.Connector == "default" || Agent.StoreSetting.Connector == "" {
		Agent.Store, err = xunStore.NewXun(Agent.StoreSetting)
		return err
	}

	// other connector
	conn, err := connector.Select(Agent.StoreSetting.Connector)
	if err != nil {
		return fmt.Errorf("load connectors error: %s", err.Error())
	}

	if conn.Is(connector.DATABASE) {
		Agent.Store, err = xunStore.NewXun(Agent.StoreSetting)
		return err

	} else if conn.Is(connector.REDIS) {
		Agent.Store = redisStore.NewRedis()
		return nil

	} else if conn.Is(connector.MONGO) {
		Agent.Store = mongoStore.NewMongo()
		return nil
	}

	return fmt.Errorf("%s store connector %s not support", Agent.ID, Agent.StoreSetting.Connector)
}

// initAssistant initialize the assistant
func initAssistant() error {

	// Set Storage
	assistant.SetStorage(Agent.Store)

	// Assistant Vision
	if Agent.Vision != nil {
		assistant.SetVision(Agent.Vision)
	}

	if Agent.Connectors != nil {
		assistant.SetConnectorSettings(Agent.Connectors)
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

	Agent.Assistant = defaultAssistant
	return nil
}

// defaultAssistant get the default assistant
func defaultAssistant() (*assistant.Assistant, error) {
	if Agent.Use == nil || Agent.Use.Default == "" {
		return nil, fmt.Errorf("default assistant not found")
	}
	return assistant.Get(Agent.Use.Default)
}
