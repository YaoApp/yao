package model

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/dsl"
	"github.com/yaoapp/yao/dsl/types"
	"github.com/yaoapp/yao/share"
)

// SystemModels system models
var systemModels = map[string]string{
	"__yao.assistant":          "yao/models/assistant.mod.yao",
	"__yao.attachment":         "yao/models/attachment.mod.yao",
	"__yao.audit":              "yao/models/audit.mod.yao",
	"__yao.chat":               "yao/models/chat.mod.yao",
	"__yao.config":             "yao/models/config.mod.yao",
	"__yao.dsl":                "yao/models/dsl.mod.yao",
	"__yao.history":            "yao/models/history.mod.yao",
	"__yao.kb.collection":      "yao/models/kb/collection.mod.yao",
	"__yao.kb.document":        "yao/models/kb/document.mod.yao",
	"__yao.user":               "yao/models/user.mod.yao",
	"__yao.user_role":          "yao/models/user_role.mod.yao",
	"__yao.user_type":          "yao/models/user_type.mod.yao",
	"__yao.user_oauth_account": "yao/models/user_oauth_account.mod.yao",
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
		for _, message := range messages {
			log.Error("Load filesystem models error: %s", message)
		}
		return fmt.Errorf(strings.Join(messages, ";\n"))
	}

	// Load database models ( ignore error)
	errs := loadDatabaseModels()
	if len(errs) > 0 {
		for _, err := range errs {
			log.Error("Load database models error: %s", err.Error())
		}
	}
	return err
}

// LoadSystemModels load system models
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
		err = mod.Migrate(false, model.WithDonotInsertValues(true))
		if err != nil {
			log.Error("migrate system model %s error: %s", id, err.Error())
			return err
		}
	}

	return nil
}

// LoadDatabaseModels load database models
func loadDatabaseModels() []error {

	var errs []error = []error{}
	manager, err := dsl.New(types.TypeModel)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	models, err := manager.List(ctx, &types.ListOptions{Store: types.StoreTypeDB, Source: true})
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	// Load models
	for _, info := range models {
		_, err := model.LoadSource([]byte(info.Source), info.ID, info.Path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
