package neo

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/attachment"
	"github.com/yaoapp/yao/neo/i18n"
	"github.com/yaoapp/yao/neo/store"
)

// Neo the neo AI assistant
var Neo *DSL

// Load load AIGC
func Load(cfg config.Config) error {

	setting := DSL{
		ID:     "neo",
		Allows: []string{},
		StoreSetting: store.Setting{
			Prefix:    "yao_neo_",
			Connector: "default",
		},
	}

	bytes, err := application.App.Read(filepath.Join("neo", "neo.yml"))
	if err != nil {
		return err
	}

	err = application.Parse("neo.yml", bytes, &setting)
	if err != nil {
		return err
	}

	if setting.StoreSetting.MaxSize == 0 {
		setting.StoreSetting.MaxSize = 20 // default is 20
	}

	// Default Assistant, Neo is the developer name, Mohe is the brand name of the assistant
	if setting.Use == nil {
		setting.Use = &Use{Default: "mohe"} // Neo is the developer name, Mohe is the brand name of the assistant
	}

	// Title Assistant
	if setting.Use.Title == "" {
		setting.Use.Title = setting.Use.Default
	}

	// Prompt Assistant
	if setting.Use.Prompt == "" {
		setting.Use.Prompt = setting.Use.Default
	}

	Neo = &setting

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
	if Neo.AuthSetting == nil {
		Neo.AuthSetting = &Auth{
			Models:        &AuthModels{User: "admin.user", Guest: "guest"},
			Fields:        &AuthFields{ID: "id", Roles: "roles", Permission: "permission"},
			SessionFields: &AuthSessionFields{ID: "user_id", Roles: "user_roles", Guest: "guest_id"},
		}
	}

	if Neo.AuthSetting.Models == nil {
		Neo.AuthSetting.Models = &AuthModels{User: "admin.user", Guest: "guest"}
	}

	if Neo.AuthSetting.Fields == nil {
		Neo.AuthSetting.Fields = &AuthFields{ID: "id", Roles: "roles", Permission: "permission"}
	}

	if Neo.AuthSetting.SessionFields == nil {
		Neo.AuthSetting.SessionFields = &AuthSessionFields{ID: "user_id", Roles: "user_roles", Guest: "guest_id"}
	}

	if Neo.AuthSetting.Models.User == "" {
		Neo.AuthSetting.Models.User = "admin.user"
	}

	if Neo.AuthSetting.Models.Guest == "" {
		Neo.AuthSetting.Models.Guest = "guest"
	}

	if Neo.AuthSetting.Fields.Roles == "" {
		Neo.AuthSetting.Fields.Roles = "roles"
	}

	if Neo.AuthSetting.Fields.Permission == "" {
		Neo.AuthSetting.Fields.Permission = "permission"
	}

	if Neo.AuthSetting.Fields.ID == "" {
		Neo.AuthSetting.Fields.ID = "id"
	}

	if Neo.AuthSetting.Fields.ID == "" {
		Neo.AuthSetting.Fields.ID = "id"
	}

	if Neo.AuthSetting.SessionFields.ID == "" {
		Neo.AuthSetting.SessionFields.ID = "user_id"
	}

	if Neo.AuthSetting.SessionFields.Roles == "" {
		Neo.AuthSetting.SessionFields.Roles = "user_roles"
	}

	if Neo.AuthSetting.SessionFields.Guest == "" {
		Neo.AuthSetting.SessionFields.Guest = "guest_id"
	}

	// Validate User Model and Fields
	if !model.Exists(Neo.AuthSetting.Models.User) {
		return fmt.Errorf("model %s not found", Neo.AuthSetting.Models.User)
	}
	user := model.Select(Neo.AuthSetting.Models.User)
	shouldHave := []string{Neo.AuthSetting.Fields.ID, Neo.AuthSetting.Fields.Roles, Neo.AuthSetting.Fields.Permission}
	for _, name := range shouldHave {
		if _, has := user.Columns[name]; !has {
			return fmt.Errorf("model %s should have column %s", Neo.AuthSetting.Models.User, name)
		}
	}

	return nil
}

// initUpload initialize the upload
func initUpload() error {

	if Neo.UploadSetting == nil {
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
	if Neo.UploadSetting.Chat == nil {
		_, err := attachment.RegisterDefault("chat")
		if err != nil {
			return err
		}
	}

	// Use the chat upload setting for knowledge upload, if the knowledge upload setting is not set.
	if Neo.UploadSetting.Knowledge == nil {
		if Neo.UploadSetting.Chat == nil {
			_, err := attachment.RegisterDefault("knowledge")
			if err != nil {
				return err
			}
		} else {
			_, err := attachment.Register("knowledge", Neo.UploadSetting.Chat.Driver, *Neo.UploadSetting.Chat)
			if err != nil {
				return err
			}
		}
	}

	// Use custom chat upload setting
	if Neo.UploadSetting.Chat != nil {
		Neo.UploadSetting.Chat.ReplaceEnv(config.Conf.DataRoot)
		_, err := attachment.Register("chat", Neo.UploadSetting.Chat.Driver, *Neo.UploadSetting.Chat) // Register the chat upload manager
		if err != nil {
			return err
		}
	}

	// Use custom knowledge upload setting
	if Neo.UploadSetting.Knowledge != nil {
		Neo.UploadSetting.Knowledge.ReplaceEnv(config.Conf.DataRoot)
		_, err := attachment.Register("knowledge", Neo.UploadSetting.Knowledge.Driver, *Neo.UploadSetting.Knowledge)
		if err != nil {
			return err
		}
	}

	// Use the chat upload setting for asset upload, if the asset upload setting is not set. (public assets)
	if Neo.UploadSetting.Assets == nil {
		_, err := attachment.RegisterDefault("assets")
		if err != nil {
			return err
		}
	}

	// Use custom asset upload setting
	if Neo.UploadSetting.Assets != nil {
		Neo.UploadSetting.Assets.ReplaceEnv(config.Conf.DataRoot)
		_, err := attachment.Register("assets", Neo.UploadSetting.Assets.Driver, *Neo.UploadSetting.Assets)
		if err != nil {
			return err
		}
	}
	return nil
}

// initGlobalI18n initialize the global i18n
func initGlobalI18n() error {
	locales, err := i18n.GetLocales("neo")
	if err != nil {
		return err
	}
	i18n.Locales["__global__"] = locales.Flatten()
	return nil
}

// initConnectors initialize the connectors
func initConnectors() error {
	path := filepath.Join("neo", "connectors.yml")
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

	Neo.Connectors = connectors
	return nil
}

// initStore initialize the store
func initStore() error {

	var err error
	if Neo.StoreSetting.Connector == "default" || Neo.StoreSetting.Connector == "" {
		Neo.Store, err = store.NewXun(Neo.StoreSetting)
		return err
	}

	// other connector
	conn, err := connector.Select(Neo.StoreSetting.Connector)
	if err != nil {
		return err
	}

	if conn.Is(connector.DATABASE) {
		Neo.Store, err = store.NewXun(Neo.StoreSetting)
		return err

	} else if conn.Is(connector.REDIS) {
		Neo.Store = store.NewRedis()
		return nil

	} else if conn.Is(connector.MONGO) {
		Neo.Store = store.NewMongo()
		return nil
	}

	return fmt.Errorf("%s store connector %s not support", Neo.ID, Neo.StoreSetting.Connector)
}

// initAssistant initialize the assistant
func initAssistant() error {

	// Set Storage
	assistant.SetStorage(Neo.Store)

	// Assistant Vision
	if Neo.Vision != nil {
		assistant.SetVision(Neo.Vision)
	}

	if Neo.Connectors != nil {
		assistant.SetConnectorSettings(Neo.Connectors)
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

	Neo.Assistant = defaultAssistant
	return nil
}

// defaultAssistant get the default assistant
func defaultAssistant() (*assistant.Assistant, error) {
	if Neo.Use == nil || Neo.Use.Default == "" {
		return nil, fmt.Errorf("default assistant not found")
	}
	return assistant.Get(Neo.Use.Default)
}
