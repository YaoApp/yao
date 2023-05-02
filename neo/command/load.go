package command

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Commands the commands
var Commands = map[string]*Command{}

// Autopilots the autopilots
var Autopilots = []string{}

// Load load AIGC
func Load(cfg config.Config) error {
	exts := []string{"*.cmd.yml", "*.cmd.yaml"}
	messages := []string{}

	err := application.App.Walk("neo", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := share.ID(root, file)
		_, err := LoadFile(file, id)
		if err != nil {
			messages = append(messages, err.Error())
		}
		return nil
	}, exts...)

	if err != nil {
		return err
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}

	return nil

}

// LoadFile load AIGC by file
func LoadFile(file string, id string) (*Command, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}
	return LoadSource(data, file, id)
}

// LoadSource load AIGC
func LoadSource(data []byte, file, id string) (*Command, error) {

	cmd := Command{
		ID: id,
		Prepare: Prepare{
			Option: map[string]interface{}{},
		},
		Optional: Optional{
			Autopilot:   false,
			Confirm:     false,
			MaxAttempts: 10,
		},
	}

	err := application.Parse(file, data, &cmd)
	if err != nil {
		return nil, err
	}

	if cmd.Process == "" {
		return nil, fmt.Errorf("%s process is required", id)
	}

	if cmd.Prepare.Prompts == nil || len(cmd.Prepare.Prompts) == 0 {
		return nil, fmt.Errorf("%s prompts is required", id)
	}

	// create AI interface
	cmd.AI, err = cmd.newAI()
	if err != nil {
		return nil, err
	}

	// add to autopilots
	if cmd.Optional.Autopilot {
		Autopilots = append(Autopilots, id)
	}

	// save
	err = cmd.save()
	if err != nil {
		return nil, err
	}

	// add to AIGCs
	Commands[id] = &cmd
	return Commands[id], nil
}
