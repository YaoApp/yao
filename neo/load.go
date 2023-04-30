package neo

import (
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/aigc"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/conversation"
)

// Neo the neo AI assistant
var Neo *DSL

// Load load AIGC
func Load(cfg config.Config) error {

	setting := DSL{
		ID:      "neo",
		Prompts: []aigc.Prompt{},
		Option:  map[string]interface{}{},
		Allows:  []string{},
		ConversationSetting: conversation.Setting{
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

	if setting.ConversationSetting.MaxSize == 0 {
		setting.ConversationSetting.MaxSize = 100
	}

	Neo = &setting
	err = Neo.newAI()
	if err != nil {
		return err
	}

	err = Neo.newConversation()
	if err != nil {
		return err
	}

	return nil
}

// LoadCommands load the commands
func (neo *DSL) LoadCommands() {}
