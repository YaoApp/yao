package aigc

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load AIGC
func Load(cfg config.Config) error {

	// Ignore if the aigcs directory does not exist
	exists, err := application.App.Exists("aigcs")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	exts := []string{"*.ai.yml", "*.ai.yaml"}
	messages := []string{}
	err = application.App.Walk("aigcs", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := aigcID(root, file)
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

// aigcID parses AIGC ID from file path
// Special handling for .ai.yml and .ai.yaml extensions
// e.g., "aigcs/translate.ai.yml" -> "translate"
func aigcID(root, file string) string {
	id := share.ID(root, file)
	// Remove "_ai" suffix caused by .ai.yml/.ai.yaml extension
	// share.ID treats .yml/.yaml as single extension, so "translate.ai.yml" becomes "translate_ai"
	id = strings.TrimSuffix(id, "_ai")
	return id
}

// LoadFile load AIGC by file
func LoadFile(file string, id string) (*DSL, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}
	return LoadSource(data, file, id)
}

// LoadSource load AIGC
func LoadSource(data []byte, file, id string) (*DSL, error) {

	dsl := DSL{
		ID: id,
		Optional: Optional{
			Autopilot: false,
			JSON:      false,
		},
	}

	err := application.Parse(file, data, &dsl)
	if err != nil {
		return nil, err
	}

	if dsl.Prompts == nil || len(dsl.Prompts) == 0 {
		return nil, fmt.Errorf("%s prompts is required", id)
	}

	// create AI interface
	dsl.AI, err = dsl.newAI()
	if err != nil {
		return nil, err
	}

	// add to autopilots
	if dsl.Optional.Autopilot {
		Autopilots = append(Autopilots, id)
	}

	// add to AIGCs
	AIGCs[id] = &dsl
	return AIGCs[id], nil
}
