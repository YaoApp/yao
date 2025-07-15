package model

import (
	"fmt"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/share"
)

// SystemModels system models
var systemModels = map[string]string{
	"__yao.assistant":  "yao/models/assistant.mod.yao",
	"__yao.attachment": "yao/models/attachment.mod.yao",
	"__yao.audit":      "yao/models/audit.mod.yao",
	"__yao.chat":       "yao/models/chat.mod.yao",
	"__yao.config":     "yao/models/config.mod.yao",
	"__yao.dsl":        "yao/models/dsl.mod.yao",
	"__yao.history":    "yao/models/history.mod.yao",
	"__yao.kb":         "yao/models/kb.mod.yao",
}

// Load load models
func Load(cfg config.Config) error {

	messages := []string{}

	model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, cfg.DB.AESKey)), "AES")
	model.WithCrypt([]byte(`{}`), "PASSWORD")

	// Load system models
	err := loadSystemModels()
	if err != nil {
		return err
	}

	// Load filesystem models
	exts := []string{"*.mod.yao", "*.mod.json", "*.mod.jsonc"}
	err = application.App.Walk("models", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := model.Load(file, share.ID(root, file))
		if err != nil {
			messages = append(messages, err.Error())
		}
		return err
	}, exts...)

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";\n"))
	}

	// Load database models ( ignore error)
	err = loadDatabaseModels()
	if err != nil {
		log.Error("load database models error: %s", err.Error())
	}

	return err
}

func loadSystemModels() error {
	for id, path := range systemModels {
		content, err := data.Read(path)
		if err != nil {
			return err
		}

		// Parse model
		var data map[string]interface{}
		err = application.Parse(path, content, &data)
		if err != nil {
			return err
		}

		// Set prefix
		if table, ok := data["table"].(map[string]interface{}); ok {
			if name, ok := table["name"].(string); ok {
				table["name"] = share.App.Prefix + name
				content, err = jsoniter.Marshal(data)
				if err != nil {
					log.Error("failed to marshal model data: %v", err)
					return fmt.Errorf("failed to marshal model data: %v", err)
				}
			}
		}

		// Load Model
		mod, err := model.LoadSource(content, id, filepath.Join("__system", path))
		if err != nil {
			log.Error("load system model %s error: %s", id, err.Error())
			return err
		}

		// Auto migrate
		err = mod.Migrate(true, model.WithDonotInsertValues(true))
		if err != nil {
			log.Error("migrate system model %s error: %s", id, err.Error())
			return err
		}
	}

	return nil
}

func loadDatabaseModels() error {
	return nil
}
