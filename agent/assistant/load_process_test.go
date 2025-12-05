package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/testutils"
)

func TestLoadProcessIntegration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// After testutils.Prepare, all assistants should be loaded and scripts registered
	// Test calling mcpload assistant's tools.Hello function

	t.Run("CallHelloAfterLoad", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.Hello", map[string]interface{}{
			"name": "TestUser",
		})

		err := proc.Execute()
		assert.NoError(t, err)

		result := proc.Value()
		assert.NotNil(t, result)

		resultStr, ok := result.(string)
		assert.True(t, ok, "Result should be a string")
		assert.Contains(t, resultStr, "Hello, TestUser")
		assert.Contains(t, resultStr, "mcpload assistant")
	})

	t.Run("CallPingAfterLoad", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.Ping", map[string]interface{}{
			"message": "integration test",
		})

		err := proc.Execute()
		assert.NoError(t, err)

		result := proc.Value()
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok, "Result should be a map")
		assert.Equal(t, "integration test", resultMap["message"])
		assert.Contains(t, resultMap["echo"], "Pong")
		assert.NotEmpty(t, resultMap["timestamp"])
	})

	t.Run("CallCalculateAfterLoad", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.Calculate", map[string]interface{}{
			"operation": "add",
			"a":         float64(100),
			"b":         float64(50),
		})

		err := proc.Execute()
		assert.NoError(t, err)

		result := proc.Value()
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok, "Result should be a map")
		assert.Equal(t, float64(150), resultMap["result"])
		assert.Equal(t, "add", resultMap["operation"])
		assert.Equal(t, float64(100), resultMap["a"])
		assert.Equal(t, float64(50), resultMap["b"])
	})

	t.Run("CallNonExistentScript", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.nonexistent.Method")

		err := proc.Execute()
		assert.NotNil(t, err, "Should return error for non-existent script")
		assert.Contains(t, err.Error(), "Exception|404")
	})

	t.Run("CallNonExistentMethod", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.NonExistentMethod")

		err := proc.Execute()
		assert.NotNil(t, err, "Should return error for non-existent method")
		assert.Contains(t, err.Error(), "Exception|500")
	})
}

func TestLoadProcessMultipleAssistants(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Test that multiple assistants can have their scripts registered
	// and process calls work correctly for different assistants

	t.Run("MCPLoadAssistant", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.Hello", map[string]interface{}{
			"name": "User1",
		})

		err := proc.Execute()
		assert.NoError(t, err)

		result := proc.Value()
		resultStr, ok := result.(string)
		assert.True(t, ok)
		assert.Contains(t, resultStr, "mcpload assistant")
	})

	// If there are other test assistants with scripts, they can be tested here
	// For now, we verify that the handler is properly isolated per assistant
	t.Run("VerifyIsolation", func(t *testing.T) {
		// Verify that the mcpload handler is correctly registered
		handler, exists := process.Handlers["agents.tests.mcpload.tools"]
		assert.True(t, exists, "Handler should be registered")
		assert.NotNil(t, handler)
	})
}
