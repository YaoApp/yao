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
	"__yao.agent.assistant":    "yao/models/agent/assistant.mod.yao",
	"__yao.agent.chat":         "yao/models/agent/chat.mod.yao",
	"__yao.agent.execution":    "yao/models/agent/execution.mod.yao",
	"__yao.agent.message":      "yao/models/agent/message.mod.yao",
	"__yao.agent.resume":       "yao/models/agent/resume.mod.yao",
	"__yao.agent.search":       "yao/models/agent/search.mod.yao",
	"__yao.attachment":         "yao/models/attachment.mod.yao",
	"__yao.audit":              "yao/models/audit.mod.yao",
	"__yao.config":             "yao/models/config.mod.yao",
	"__yao.dsl":                "yao/models/dsl.mod.yao",
	"__yao.invitation":         "yao/models/invitation.mod.yao",
	"__yao.job.category":       "yao/models/job/category.mod.yao",
	"__yao.job":                "yao/models/job/job.mod.yao",
	"__yao.job.execution":      "yao/models/job/execution.mod.yao",
	"__yao.job.log":            "yao/models/job/log.mod.yao",
	"__yao.kb.collection":      "yao/models/kb/collection.mod.yao",
	"__yao.kb.document":        "yao/models/kb/document.mod.yao",
	"__yao.team":               "yao/models/team.mod.yao",
	"__yao.member":             "yao/models/member.mod.yao",
	"__yao.user":               "yao/models/user.mod.yao",
	"__yao.role":               "yao/models/role.mod.yao",
	"__yao.user.type":          "yao/models/user/type.mod.yao",
	"__yao.user.oauth_account": "yao/models/user/oauth_account.mod.yao",
}

// Load load models
func Load(cfg config.Config) error {

	messages := []string{}

	model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, cfg.DB.AESKey)), "AES")
	model.WithCrypt([]byte(`{}`), "PASSWORD")

	// Load system models (without migrate)
	systemModels, err := loadSystemModels()
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
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}

	// Load models from assistants (without migrate)
	assistantModels, errsAssistants := loadAssistantModels()
	if len(errsAssistants) > 0 {
		for _, err := range errsAssistants {
			log.Error("Load assistant models error: %s", err.Error())
		}
	}

	// Batch migrate all system and assistant models
	allModels := make(map[string]*model.Model)
	for id, mod := range systemModels {
		allModels[id] = mod
	}
	for id, mod := range assistantModels {
		allModels[id] = mod
	}

	err = BatchMigrate(allModels)
	if err != nil {
		return err
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

// LoadSystemModels load system models (without migration)
func loadSystemModels() (map[string]*model.Model, error) {
	models := make(map[string]*model.Model)

	for id, path := range systemModels {
		content, err := data.Read(path)
		if err != nil {
			return nil, err
		}

		// Parse model
		var data map[string]interface{}
		err = application.Parse(path, content, &data)
		if err != nil {
			return nil, err
		}

		// Set prefix
		if table, ok := data["table"].(map[string]interface{}); ok {
			if name, ok := table["name"].(string); ok {
				table["name"] = share.App.Prefix + name
				content, err = jsoniter.Marshal(data)
				if err != nil {
					log.Error("failed to marshal model data: %v", err)
					return nil, fmt.Errorf("failed to marshal model data: %v", err)
				}
			}
		}

		// Load Model (just parse, no migration)
		mod, err := model.LoadSource(content, id, filepath.Join("__system", path))
		if err != nil {
			log.Error("load system model %s error: %s", id, err.Error())
			return nil, err
		}

		models[id] = mod
	}

	return models, nil
}

// loadAssistantModels load models from assistants directory (without migration)
func loadAssistantModels() (map[string]*model.Model, []error) {
	models := make(map[string]*model.Model)
	var errs []error = []error{}

	// Check if assistants directory exists
	exists, err := application.App.Exists("assistants")
	if err != nil || !exists {
		log.Trace("Assistants directory not found or not accessible")
		return models, errs
	}

	log.Trace("Loading models from assistants directory...")

	// Track processed assistants to avoid duplicates
	processedAssistants := make(map[string]bool)

	// Walk through assistants directory to find all valid assistants with models
	err = application.App.Walk("assistants", func(root, file string, isdir bool) error {
		if !isdir {
			return nil
		}

		// Check if this is a valid assistant directory (has package.yao)
		pkgFile := filepath.Join(root, file, "package.yao")
		pkgExists, _ := application.App.Exists(pkgFile)
		if !pkgExists {
			return nil
		}

		// Extract assistant ID from path
		assistantID := strings.TrimPrefix(file, "/")
		assistantID = strings.ReplaceAll(assistantID, "/", ".")

		// Skip if already processed
		if processedAssistants[assistantID] {
			return nil
		}
		processedAssistants[assistantID] = true

		log.Trace("Found assistant: %s", assistantID)

		// Check if the assistant has a models directory
		modelsDir := filepath.Join(root, file, "models")
		modelsDirExists, _ := application.App.Exists(modelsDir)
		if !modelsDirExists {
			log.Trace("Assistant %s has no models directory", assistantID)
			return nil
		}

		log.Trace("Loading models from assistant %s", assistantID)

		// Load models from the assistant's models directory
		exts := []string{"*.mod.yao", "*.mod.json", "*.mod.jsonc"}
		err := application.App.Walk(modelsDir, func(modelRoot, modelFile string, modelIsDir bool) error {
			if modelIsDir {
				return nil
			}

			// Generate model ID with agents.<assistantID>./ prefix
			// Support nested paths: "models/foo/bar.mod.yao" -> "foo.bar"
			relPath := strings.TrimPrefix(modelFile, modelsDir+"/")
			relPath = strings.TrimPrefix(relPath, "/")
			relPath = strings.TrimSuffix(relPath, ".mod.yao")
			relPath = strings.TrimSuffix(relPath, ".mod.json")
			relPath = strings.TrimSuffix(relPath, ".mod.jsonc")
			modelName := strings.ReplaceAll(relPath, "/", ".")
			modelID := fmt.Sprintf("agents.%s.%s", assistantID, modelName)

			log.Trace("Loading model %s from file %s", modelID, modelFile)

			// Read and modify model to add table prefix
			content, err := application.App.Read(modelFile)
			if err != nil {
				log.Error("Failed to read model file %s: %s", modelFile, err.Error())
				errs = append(errs, fmt.Errorf("failed to read model %s: %w", modelID, err))
				return nil
			}

			// Parse model
			var modelData map[string]interface{}
			err = application.Parse(modelFile, content, &modelData)
			if err != nil {
				log.Error("Failed to parse model %s: %s", modelID, err.Error())
				errs = append(errs, fmt.Errorf("failed to parse model %s: %w", modelID, err))
				return nil
			}

			// Set table name prefix: agents_<assistantID>_
			// Convert dots to underscores: tests.mcpload -> agents_tests_mcpload_
			if table, ok := modelData["table"].(map[string]interface{}); ok {
				if tableName, ok := table["name"].(string); ok {
					// Generate prefix from assistant ID
					prefix := "agents_" + strings.ReplaceAll(assistantID, ".", "_") + "_"

					// Remove any existing prefix if present
					tableName = strings.TrimPrefix(tableName, "agents_mcpload_")
					tableName = strings.TrimPrefix(tableName, prefix)

					table["name"] = prefix + tableName
					content, err = jsoniter.Marshal(modelData)
					if err != nil {
						log.Error("Failed to marshal model data for %s: %v", modelID, err)
						errs = append(errs, fmt.Errorf("failed to marshal model %s: %w", modelID, err))
						return nil
					}
				}
			}

			// Load model with modified content (just parse, no migration)
			mod, err := model.LoadSource(content, modelID, modelFile)
			if err != nil {
				log.Error("Failed to load model %s from assistant %s: %s", modelID, assistantID, err.Error())
				errs = append(errs, fmt.Errorf("failed to load model %s: %w", modelID, err))
				return nil // Continue loading other models
			}

			models[modelID] = mod
			log.Trace("Loaded model: %s", modelID)
			return nil
		}, exts...)

		if err != nil {
			errs = append(errs, fmt.Errorf("failed to walk models in assistant %s: %w", assistantID, err))
		}

		return nil
	}, "")

	if err != nil {
		errs = append(errs, fmt.Errorf("failed to walk assistants directory: %w", err))
	}

	return models, errs
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
