package model

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range model.Models {
		ids[id] = true
	}

	// Standard models
	assert.True(t, ids["user"])
	assert.True(t, ids["category"])
	assert.True(t, ids["tag"])
	assert.True(t, ids["pet"])
	assert.True(t, ids["pet.tag"])
	assert.True(t, ids["user.pet"])
	assert.True(t, ids["tests.user"])

	// Agent models
	assert.True(t, ids["agents.tests.mcpload.test_record"], "Agent model agents.tests.mcpload.test_record should be loaded")
	assert.True(t, ids["agents.tests.mcpload.nested.item"], "Agent nested model agents.tests.mcpload.nested.item should be loaded")

	// Verify table names have correct prefix
	if testRecordModel, exists := model.Models["agents.tests.mcpload.test_record"]; exists {
		assert.Equal(t, "agents_tests_mcpload_test_records", testRecordModel.MetaData.Table.Name, "Table name should have agents_tests_mcpload_ prefix")
		t.Logf("✓ Agent model table name: %s", testRecordModel.MetaData.Table.Name)
	}

	if nestedItemModel, exists := model.Models["agents.tests.mcpload.nested.item"]; exists {
		assert.Equal(t, "agents_tests_mcpload_items", nestedItemModel.MetaData.Table.Name, "Nested model table name should have agents_tests_mcpload_ prefix")
		t.Logf("✓ Nested agent model table name: %s", nestedItemModel.MetaData.Table.Name)
	}

	// Log all agent models found
	agentModels := []string{}
	for id := range model.Models {
		if len(id) >= 7 && id[:7] == "agents." {
			agentModels = append(agentModels, id)
		}
	}
	if len(agentModels) > 0 {
		t.Logf("✓ Found %d agent model(s):", len(agentModels))
		for _, id := range agentModels {
			t.Logf("  - %s", id)
		}
	}
}
