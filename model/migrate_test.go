package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestBatchMigrate(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	t.Run("LoadSystemModels", func(t *testing.T) {
		models, err := loadSystemModels()
		assert.NoError(t, err, "Should load system models without error")
		assert.NotEmpty(t, models, "Should have loaded system models")

		// Check that all models have table names
		for id, mod := range models {
			assert.NotEmpty(t, mod.MetaData.Table.Name, "Model %s should have table name", id)
		}

		t.Logf("Loaded %d system models", len(models))
	})

	t.Run("LoadAssistantModels", func(t *testing.T) {
		models, errs := loadAssistantModels()
		assert.Empty(t, errs, "Should load assistant models without critical errors")

		t.Logf("Loaded %d assistant models", len(models))
	})

	t.Run("BatchMigrateAllModels", func(t *testing.T) {
		// Load all models
		systemModels, err := loadSystemModels()
		assert.NoError(t, err)

		assistantModels, _ := loadAssistantModels()

		// Combine all models
		allModels := make(map[string]*model.Model)
		for id, mod := range systemModels {
			allModels[id] = mod
		}
		for id, mod := range assistantModels {
			allModels[id] = mod
		}

		// Run batch migrate
		err = BatchMigrate(allModels)
		assert.NoError(t, err, "Batch migrate should succeed")

		t.Logf("Batch migrated %d models", len(allModels))
	})

	t.Run("BatchMigrateIdempotent", func(t *testing.T) {
		// Load models
		systemModels, err := loadSystemModels()
		assert.NoError(t, err)

		// Run batch migrate twice - should be idempotent
		err = BatchMigrate(systemModels)
		assert.NoError(t, err, "First batch migrate should succeed")

		err = BatchMigrate(systemModels)
		assert.NoError(t, err, "Second batch migrate should also succeed (idempotent)")

		t.Logf("Batch migrate is idempotent")
	})
}
