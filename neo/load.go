package neo

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/rag"
	"github.com/yaoapp/yao/neo/store"
	"github.com/yaoapp/yao/neo/vision"
	"github.com/yaoapp/yao/neo/vision/driver"
)

// Neo the neo AI assistant
var Neo *DSL

// Load load AIGC
func Load(cfg config.Config) error {

	setting := DSL{
		ID:      "neo",
		Prompts: []assistant.Prompt{},
		Option:  map[string]interface{}{},
		Allows:  []string{},
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
		setting.StoreSetting.MaxSize = 100
	}

	// Default Assistant
	if setting.Use == nil {
		setting.Use = &Use{Default: "neo"}
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

	// Initialize RAG
	initRAG()

	// Initialize Vision
	initVision()

	// Initialize Assistant
	err = initAssistant()
	if err != nil {
		return err
	}

	return nil
}

// initRAG initialize the RAG instance
func initRAG() {
	if Neo.RAGSetting.Engine.Driver == "" {
		return
	}
	instance, err := rag.New(Neo.RAGSetting)
	if err != nil {
		color.Red("[Neo] Failed to initialize RAG: %v", err)
		log.Error("[Neo] Failed to initialize RAG: %v", err)
		return
	}

	Neo.RAG = instance
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

// initVision initialize the Vision instance
func initVision() {
	if Neo.VisionSetting.Storage.Driver == "" {
		return
	}

	cfg := &driver.Config{
		Storage: Neo.VisionSetting.Storage,
		Model:   Neo.VisionSetting.Model,
	}

	instance, err := vision.New(cfg)
	if err != nil {
		color.Red("[Neo] Failed to initialize Vision: %v", err)
		log.Error("[Neo] Failed to initialize Vision: %v", err)
		return
	}

	Neo.Vision = instance
}

// initAssistant initialize the assistant
func initAssistant() error {

	// Set Storage
	assistant.SetStorage(Neo.Store)

	// Assistant RAG
	if Neo.RAG != nil {
		assistant.SetRAG(
			Neo.RAG.Engine(),
			Neo.RAG.FileUpload(),
			Neo.RAG.Vectorizer(),
			assistant.RAGSetting{
				IndexPrefix: Neo.RAGSetting.IndexPrefix,
			},
		)
	}

	// Assistant Vision
	if Neo.Vision != nil {
		assistant.SetVision(Neo.Vision)
	}

	if Neo.Connectors != nil {
		assistant.SetConnectorSettings(Neo.Connectors)
	}

	// Default Connector
	assistant.SetConnector(Neo.Connector)

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
	if Neo.Use != nil && Neo.Use.Default != "" {
		return assistant.Get(Neo.Use.Default)
	}

	name := Neo.Name
	if name == "" {
		name = "Neo"
	}

	return assistant.GetByConnector(Neo.Connector, name)
}
