package store

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load store
func Load(cfg config.Config) error {

	// Ignore if the stores directory does not exist
	exists, err := application.App.Exists("stores")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	messages := []string{}
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	err = application.App.Walk("stores", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := store.Load(file, share.ID(root, file))
		if err != nil {
			messages = append(messages, err.Error())
		}
		return err
	}, exts...)

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";\n"))
	}
	return err
}
