package neo

import (
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/store"
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
