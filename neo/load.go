package neo

import (
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/aigc"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/command"
	"github.com/yaoapp/yao/neo/command/driver"
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
		Command: Command{Parser: ""},
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

	// AI Setting
	err = Neo.newAI()
	if err != nil {
		return err
	}

	// Conversation Setting
	err = Neo.newConversation()
	if err != nil {
		return err
	}

	// Command Setting
	parser := setting.Command.Parser
	if parser == "" || parser == "default" {
		parser = setting.Connector
	}
	store, err := driver.NewMemory(setting.Command.Parser, nil)
	if err != nil {
		return err
	}
	command.SetStore(store)

	// Load the commands
	err = command.Load(cfg)
	if err != nil {
		log.Error("Command Load Error: %s", err.Error())
	}

	return nil
}
