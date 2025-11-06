package agent

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/store"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/config"
)

// Agent the agent AI assistant
var Agent *DSL

// Load load AIGC
func Load(cfg config.Config) error {

	setting := DSL{
		ID:     "agent",
		Allows: []string{},
		StoreSetting: store.Setting{
			Prefix:    "yao_agent_",
			Connector: "default",
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

	// Initialize Connectors
	err = initConnectors()
	if err != nil {
		return err
	}

	// Initialize Global I18n
	err = initGlobalI18n()
	if err != nil {
		return err
	}

	// Initialize Auth
	err = initAuth()
	if err != nil {
		return err
	}

	// Initialize Upload
	err = initUpload()
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

// initAuth initialize the auth
func initAuth() error {
	if Agent.AuthSetting == nil {
		Agent.AuthSetting = &Auth{
			Models:        &AuthModels{User: "admin.user", Guest: "guest"},
			Fields:        &AuthFields{ID: "id", Roles: "roles", Permission: "permission"},
			SessionFields: &AuthSessionFields{ID: "user_id", Roles: "user_roles", Guest: "guest_id"},
		}
	}

	if Agent.AuthSetting.Models == nil {
		Agent.AuthSetting.Models = &AuthModels{User: "admin.user", Guest: "guest"}
	}

	if Agent.AuthSetting.Fields == nil {
		Agent.AuthSetting.Fields = &AuthFields{ID: "id", Roles: "roles", Permission: "permission"}
	}

	if Agent.AuthSetting.SessionFields == nil {
		Agent.AuthSetting.SessionFields = &AuthSessionFields{ID: "user_id", Roles: "user_roles", Guest: "guest_id"}
	}

	if Agent.AuthSetting.Models.User == "" {
		Agent.AuthSetting.Models.User = "admin.user"
	}

	if Agent.AuthSetting.Models.Guest == "" {
		Agent.AuthSetting.Models.Guest = "guest"
	}

	if Agent.AuthSetting.Fields.Roles == "" {
		Agent.AuthSetting.Fields.Roles = "roles"
	}

	if Agent.AuthSetting.Fields.Permission == "" {
		Agent.AuthSetting.Fields.Permission = "permission"
	}

	if Agent.AuthSetting.Fields.ID == "" {
		Agent.AuthSetting.Fields.ID = "id"
	}

	if Agent.AuthSetting.Fields.ID == "" {
		Agent.AuthSetting.Fields.ID = "id"
	}

	if Agent.AuthSetting.SessionFields.ID == "" {
		Agent.AuthSetting.SessionFields.ID = "user_id"
	}

	if Agent.AuthSetting.SessionFields.Roles == "" {
		Agent.AuthSetting.SessionFields.Roles = "user_roles"
	}

	if Agent.AuthSetting.SessionFields.Guest == "" {
		Agent.AuthSetting.SessionFields.Guest = "guest_id"
	}

	// Validate User Model and Fields
	if !model.Exists(Agent.AuthSetting.Models.User) {
		return fmt.Errorf("model %s not found", Agent.AuthSetting.Models.User)
	}
	user := model.Select(Agent.AuthSetting.Models.User)
	shouldHave := []string{Agent.AuthSetting.Fields.ID, Agent.AuthSetting.Fields.Roles, Agent.AuthSetting.Fields.Permission}
	for _, name := range shouldHave {
		if _, has := user.Columns[name]; !has {
			return fmt.Errorf("model %s should have column %s", Agent.AuthSetting.Models.User, name)
		}
	}

	return nil
}

// initUpload initialize the upload
func initUpload() error {

	if Agent.UploadSetting == nil {
		_, err := attachment.RegisterDefault("chat")
		if err != nil {
			return err
		}
		_, err = attachment.RegisterDefault("knowledge")
		if err != nil {
			return err
		}
		return nil
	}

	// If the chat upload setting is not set, use the default chat upload setting.
	if Agent.UploadSetting.Chat == nil {
		_, err := attachment.RegisterDefault("chat")
		if err != nil {
			return err
		}
	}

	// Use the chat upload setting for knowledge upload, if the knowledge upload setting is not set.
	if Agent.UploadSetting.Knowledge == nil {
		if Agent.UploadSetting.Chat == nil {
			_, err := attachment.RegisterDefault("knowledge")
			if err != nil {
				return err
			}
		} else {
			_, err := attachment.Register("knowledge", Agent.UploadSetting.Chat.Driver, *Agent.UploadSetting.Chat)
			if err != nil {
				return err
			}
		}
	}

	// Use custom chat upload setting
	if Agent.UploadSetting.Chat != nil {
		Agent.UploadSetting.Chat.ReplaceEnv(config.Conf.DataRoot)
		_, err := attachment.Register("chat", Agent.UploadSetting.Chat.Driver, *Agent.UploadSetting.Chat) // Register the chat upload manager
		if err != nil {
			return err
		}
	}

	// Use custom knowledge upload setting
	if Agent.UploadSetting.Knowledge != nil {
		Agent.UploadSetting.Knowledge.ReplaceEnv(config.Conf.DataRoot)
		_, err := attachment.Register("knowledge", Agent.UploadSetting.Knowledge.Driver, *Agent.UploadSetting.Knowledge)
		if err != nil {
			return err
		}
	}

	// Use the chat upload setting for asset upload, if the asset upload setting is not set. (public assets)
	if Agent.UploadSetting.Assets == nil {
		_, err := attachment.RegisterDefault("assets")
		if err != nil {
			return err
		}
	}

	// Use custom asset upload setting
	if Agent.UploadSetting.Assets != nil {
		Agent.UploadSetting.Assets.ReplaceEnv(config.Conf.DataRoot)
		_, err := attachment.Register("assets", Agent.UploadSetting.Assets.Driver, *Agent.UploadSetting.Assets)
		if err != nil {
			return err
		}
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
func initConnectors() error {
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
		Agent.Store, err = store.NewXun(Agent.StoreSetting)
		return err
	}

	// other connector
	conn, err := connector.Select(Agent.StoreSetting.Connector)
	if err != nil {
		return err
	}

	if conn.Is(connector.DATABASE) {
		Agent.Store, err = store.NewXun(Agent.StoreSetting)
		return err

	} else if conn.Is(connector.REDIS) {
		Agent.Store = store.NewRedis()
		return nil

	} else if conn.Is(connector.MONGO) {
		Agent.Store = store.NewMongo()
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
