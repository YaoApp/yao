package attachment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := Load(config.Conf)
	assert.NoError(t, err)
	check(t)
}

func check(t *testing.T) {
	// Check that managers are loaded
	assert.NotEmpty(t, Managers, "Managers should not be empty after loading")

	// Check system uploader
	_, exists := Managers["__yao.attachment"]
	assert.True(t, exists, "System uploader __yao.attachment should be loaded")

	// Check test app uploaders (must exist)
	// These are the uploaders in yao-dev-app/uploaders/
	_, hasData := Managers["data"]
	_, hasTest := Managers["test"]

	// Both test uploaders should be loaded
	assert.True(t, hasData, "Test uploader 'data' should be loaded from data.local.yao")
	assert.True(t, hasTest, "Test uploader 'test' should be loaded from test.s3.yao")

	// Log all loaded managers for debugging
	t.Logf("Loaded managers: %v", getManagerNames())
}

// getManagerNames returns a slice of manager names for testing
func getManagerNames() []string {
	names := make([]string, 0, len(Managers))
	for name := range Managers {
		names = append(names, name)
	}
	return names
}
