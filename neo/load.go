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

	Neo = &setting

	// Store Setting
	err = Neo.initStore()
	if err != nil {
		return err
	}

	// Initialize RAG
	Neo.initRAG()

	// Initialize Vision
	Neo.initVision()

	// Initialize Assistant
	err = Neo.initAssistant()
	if err != nil {
		return err
	}

	return nil
}

// initRAG initialize the RAG instance
func (neo *DSL) initRAG() {
	if neo.RAGSetting.Engine.Driver == "" {
		return
	}
	instance, err := rag.New(neo.RAGSetting)
	if err != nil {
		color.Red("[Neo] Failed to initialize RAG: %v", err)
		log.Error("[Neo] Failed to initialize RAG: %v", err)
		return
	}

	neo.RAG = instance
}

// initStore initialize the store
func (neo *DSL) initStore() error {

	var err error
	if neo.StoreSetting.Connector == "default" || neo.StoreSetting.Connector == "" {
		neo.Store, err = store.NewXun(neo.StoreSetting)
		return err
	}

	// other connector
	conn, err := connector.Select(neo.StoreSetting.Connector)
	if err != nil {
		return err
	}

	if conn.Is(connector.DATABASE) {
		neo.Store, err = store.NewXun(neo.StoreSetting)
		return err

	} else if conn.Is(connector.REDIS) {
		neo.Store = store.NewRedis()
		return nil

	} else if conn.Is(connector.MONGO) {
		neo.Store = store.NewMongo()
		return nil
	}

	return fmt.Errorf("%s store connector %s not support", neo.ID, neo.StoreSetting.Connector)
}

// initVision initialize the Vision instance
func (neo *DSL) initVision() {
	if neo.VisionSetting.Storage.Driver == "" {
		return
	}

	cfg := &driver.Config{
		Storage: neo.VisionSetting.Storage,
		Model:   neo.VisionSetting.Model,
	}

	instance, err := vision.New(cfg)
	if err != nil {
		color.Red("[Neo] Failed to initialize Vision: %v", err)
		log.Error("[Neo] Failed to initialize Vision: %v", err)
		return
	}

	neo.Vision = instance
}

// initAssistant initialize the assistant
func (neo *DSL) initAssistant() error {

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

	// Default Connector
	assistant.SetConnector(Neo.Connector)

	// Load Built-in Assistants
	err := assistant.LoadBuiltIn()
	if err != nil {
		return err
	}

	// Default Assistant
	defaultAssistant, err := Neo.defaultAssistant()
	if err != nil {
		return err
	}

	Neo.Assistant = defaultAssistant
	return nil
}

// defaultAssistant get the default assistant
func (neo *DSL) defaultAssistant() (*assistant.Assistant, error) {
	if neo.Use != "" {
		return assistant.Get(neo.Use)
	}

	name := neo.Name
	if name == "" {
		name = "Neo"
	}

	return assistant.GetByConnector(neo.Connector, name)
}
