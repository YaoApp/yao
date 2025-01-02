package neo

import (
	"path/filepath"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/rag"
	"github.com/yaoapp/yao/neo/store"
)

// Neo the neo AI assistant
var Neo *DSL

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

// Load load AIGC
func Load(cfg config.Config) error {

	setting := DSL{
		ID:      "neo",
		Prompts: []assistant.Prompt{},
		Option:  map[string]interface{}{},
		Allows:  []string{},
		StoreSetting: store.Setting{
			Table:     "yao_neo_conversation",
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
	err = Neo.createStore()
	if err != nil {
		return err
	}

	// Initialize RAG
	Neo.initRAG()

	// Load Built-in Assistants
	assistant.SetStorage(Neo.Store)
	err = assistant.LoadBuiltIn()
	if err != nil {
		return err
	}

	defaultAssistant, err := Neo.defaultAssistant()
	if err != nil {
		return err
	}

	Neo.Assistant = defaultAssistant.API
	return nil
}
