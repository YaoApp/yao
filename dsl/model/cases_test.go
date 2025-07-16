package model

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/dsl/types"
	"github.com/yaoapp/yao/test"
)

// systemModels system models
var systemModels = map[string]string{
	"__yao.dsl": "yao/models/dsl.mod.yao",
}

func TestMain(m *testing.M) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Load system models
	model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, config.Conf.DB.AESKey)), "AES")
	model.WithCrypt([]byte(`{}`), "PASSWORD")
	err := loadSystemModels()
	if err != nil {
		log.Error("Load system models error: %s", err.Error())
		os.Exit(1)
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// loadSystemModels load system models
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
				table["name"] = "__yao_" + name
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

		// Drop table first
		err = mod.DropTable()
		if err != nil {
			log.Error("drop table error: %s", err.Error())
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

// cleanTestData cleans test data from database
func cleanTestData() error {
	m := model.Select("__yao.dsl")
	err := m.DropTable()
	if err != nil {
		return err
	}
	err = m.Migrate(false, model.WithDonotInsertValues(true))
	if err != nil {
		return err
	}
	return nil
}

// getTestID generates a unique test ID
func getTestID() string {
	return fmt.Sprintf("test_%d", time.Now().UnixNano())
}

// TestCase defines a single test case
type TestCase struct {
	ID            string
	Source        string
	UpdatedSource string
	Tags          []string
	Label         string
	Description   string
}

// NewTestCase creates a new test case
func NewTestCase() *TestCase {
	id := getTestID()
	return &TestCase{
		ID: id,
		Source: fmt.Sprintf(`{
			"name": "%s",
			"table": { "name": "%s", "comment": "Test User" },
			"columns": [
				{ "name": "id", "type": "ID" },
				{ "name": "name", "type": "string", "length": 80, "comment": "User Name", "index": true },
				{ "name": "status", "type": "enum", "option": ["active", "disabled"], "default": "active", "comment": "Status", "index": true }
			],
			"tags": ["test_%s"],
			"label": "Test Label",
			"description": "Test Description",
			"option": { "timestamps": true, "soft_deletes": true }
		}`, id, id, id),
		UpdatedSource: fmt.Sprintf(`{
			"name": "%s",
			"table": { "name": "%s", "comment": "Updated Test User" },
			"columns": [
				{ "name": "id", "type": "ID" },
				{ "name": "name", "type": "string", "length": 80, "comment": "User Name", "index": true },
				{ "name": "status", "type": "enum", "option": ["active", "disabled", "pending"], "default": "active", "comment": "Status", "index": true }
			],
			"tags": ["test_%s", "updated"],
			"label": "Updated Label",
			"description": "Updated Description",
			"option": { "timestamps": true, "soft_deletes": true }
		}`, id, id, id),
		Tags:        []string{fmt.Sprintf("test_%s", id)},
		Label:       "Test Label",
		Description: "Test Description",
	}
}

// CreateOptions returns creation options
func (tc *TestCase) CreateOptions() *types.CreateOptions {
	return &types.CreateOptions{
		ID:     tc.ID,
		Source: tc.Source,
	}
}

// UpdateOptions returns update options
func (tc *TestCase) UpdateOptions() *types.UpdateOptions {
	return &types.UpdateOptions{
		ID:     tc.ID,
		Source: tc.UpdatedSource,
	}
}

// UpdateInfoOptions returns update info options
func (tc *TestCase) UpdateInfoOptions() *types.UpdateOptions {
	return &types.UpdateOptions{
		ID: tc.ID,
		Info: &types.Info{
			Label:       "Updated via Info",
			Tags:        []string{"tag1", "info"},
			Description: "Updated via info field",
		},
	}
}

// ListOptions returns list options
func (tc *TestCase) ListOptions(withSource bool) *types.ListOptions {
	return &types.ListOptions{
		Source: withSource,
		Tags:   tc.Tags,
	}
}

// AssertInfo verifies if the information is correct
func (tc *TestCase) AssertInfo(info *types.Info) bool {
	if info == nil {
		return false
	}
	return info.ID == tc.ID &&
		info.Type == types.TypeModel &&
		info.Label == tc.Label &&
		len(info.Tags) == len(tc.Tags) &&
		info.Description == tc.Description &&
		!info.Readonly &&
		!info.Builtin &&
		!info.Mtime.IsZero() &&
		!info.Ctime.IsZero()
}

// AssertUpdatedInfo verifies if the updated information is correct
func (tc *TestCase) AssertUpdatedInfo(info *types.Info) bool {
	if info == nil {
		return false
	}
	return info.ID == tc.ID &&
		info.Type == types.TypeModel &&
		info.Label == "Updated Label" &&
		len(info.Tags) == 2 &&
		info.Description == "Updated Description" &&
		!info.Readonly &&
		!info.Builtin &&
		!info.Mtime.IsZero() &&
		!info.Ctime.IsZero()
}

// AssertUpdatedInfoViaInfo verifies if the information updated via Info is correct
func (tc *TestCase) AssertUpdatedInfoViaInfo(info *types.Info) bool {
	if info == nil {
		return false
	}
	return info.ID == tc.ID &&
		info.Type == types.TypeModel &&
		info.Label == "Updated via Info" &&
		len(info.Tags) == 2 &&
		info.Description == "Updated via info field" &&
		!info.Readonly &&
		!info.Builtin &&
		!info.Mtime.IsZero() &&
		!info.Ctime.IsZero()
}
