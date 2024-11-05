package command

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/aigc"
	"github.com/yaoapp/yao/neo/command/driver"
	"github.com/yaoapp/yao/neo/command/query"
	"github.com/yaoapp/yao/openai"
)

// DefaultStore the default store driver
var DefaultStore Store
var recmd, _ = regexp.Compile(`^\/([a-zA-Z]+) +`)
var reCmdOnly, _ = regexp.Compile(`^\/([a-zA-Z]+)$`)

// SetStore the driver interface
func SetStore(store Store) {
	DefaultStore = store
}

// Match the command from the content
func Match(sid string, query query.Param, input string) (string, error) {

	if DefaultStore == nil {
		return "", fmt.Errorf("command store is not set")
	}

	// Check the command from the store
	if id, cid, has := DefaultStore.GetRequest(sid); has {
		fmt.Println("Match Requst:", id)
		return cid, nil
	}

	// Match the command use the command ID
	match := reCmdOnly.FindSubmatch([]byte(strings.TrimSpace(input)))
	if match == nil {
		match = recmd.FindSubmatch([]byte(strings.TrimSpace(input)))
	}
	if match != nil {
		key := fmt.Sprintf("[Index]%s", match[1])
		fmt.Println("Match Index:", key)

		if cmd, ok := DefaultStore.Get(key); ok {
			fmt.Println("Match Command:", cmd.ID)
			return cmd.ID, nil
		}
	}

	return DefaultStore.Match(query, input)
}

// Exit the command
func Exit(sid string) error {
	if DefaultStore == nil {
		return fmt.Errorf("command store is not set")
	}
	DefaultStore.DelRequest(sid)
	return nil
}

// GetCommands get all commands
func GetCommands() ([]driver.Command, error) {
	if DefaultStore == nil {
		return nil, fmt.Errorf("command store is not set")
	}
	return DefaultStore.GetCommands()
}

// save the command to the store
func (cmd *Command) save() error {
	if DefaultStore == nil {
		return nil
	}

	args := []map[string]interface{}{}
	for _, arg := range cmd.Args {
		args = append(args, map[string]interface{}{
			"name":        arg.Name,
			"description": arg.Description,
			"type":        arg.Type,
			"required":    arg.Required,
		})
	}

	data := driver.Command{
		ID:          cmd.ID,
		Name:        cmd.Name,
		Use:         cmd.Use,
		Description: cmd.Description,
		Args:        args,
		Stack:       cmd.Stack,
		Path:        cmd.Path,
	}

	if cmd.Use != "" {
		key := fmt.Sprintf("[Index]%s", cmd.Use)
		err := DefaultStore.Set(key, data)
		if err != nil {
			return err
		}
	}

	return DefaultStore.Set(cmd.ID, data)
}

// NewAI create a new AI
func (cmd *Command) newAI() (aigc.AI, error) {

	if cmd.Connector == "" || strings.HasPrefix(cmd.Connector, "moapi") {
		model := "gpt-3.5-turbo"
		if strings.HasPrefix(cmd.Connector, "moapi:") {
			model = strings.TrimPrefix(cmd.Connector, "moapi:")
		}

		ai, err := openai.NewMoapi(model)
		if err != nil {
			return nil, err
		}

		cmd.AI = ai
		return cmd.AI, nil
	}

	conn, err := connector.Select(cmd.Connector)
	if err != nil {
		return nil, err
	}

	if conn.Is(connector.OPENAI) {
		return openai.New(cmd.Connector)
	}

	return nil, fmt.Errorf("%s connector %s not support, should be a openai", cmd.ID, cmd.Connector)
}
