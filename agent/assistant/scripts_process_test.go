package assistant_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/testutils"
)

func TestScriptsProcessFlow(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Get the mcpload assistant
	assistantID := "tests.mcpload"
	ast, err := assistant.Get(assistantID)
	assert.NoError(t, err)
	assert.NotNil(t, ast, "Assistant should be loaded")

	// Check that scripts were loaded
	assert.NotNil(t, ast.Scripts)
	assert.Greater(t, len(ast.Scripts), 0, "Should have loaded at least one script")

	// Verify tools.ts was loaded
	toolsScript, hasTools := ast.Scripts["tools"]
	assert.True(t, hasTools, "Should have loaded tools script")
	assert.NotNil(t, toolsScript)

	// Register scripts as process handlers
	err = ast.RegisterScripts()
	assert.NoError(t, err)

	// Test 1: Call Hello function
	t.Run("CallHelloFunction", func(t *testing.T) {
		handlerID := "agents.tests.mcpload.tools"
		handler, exists := process.Handlers[handlerID]
		assert.True(t, exists, "Handler should be registered")

		p := &process.Process{
			ID:      handlerID + ".Hello",
			Method:  "Hello",
			Args:    []interface{}{map[string]interface{}{"name": "Yao"}},
			Context: context.Background(),
		}

		result := handler(p)
		assert.NotNil(t, result)

		resultStr, ok := result.(string)
		assert.True(t, ok, "Result should be a string")
		assert.Contains(t, resultStr, "Hello, Yao")
	})

	// Test 2: Call Ping function
	t.Run("CallPingFunction", func(t *testing.T) {
		handlerID := "agents.tests.mcpload.tools"
		handler, exists := process.Handlers[handlerID]
		assert.True(t, exists, "Handler should be registered")

		p := &process.Process{
			ID:      handlerID + ".Ping",
			Method:  "Ping",
			Args:    []interface{}{map[string]interface{}{"message": "test"}},
			Context: context.Background(),
		}

		result := handler(p)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok, "Result should be a map")
		assert.Equal(t, "test", resultMap["message"])
		assert.Contains(t, resultMap["echo"], "Pong")
	})

	// Test 3: Call Calculate function
	t.Run("CallCalculateFunction", func(t *testing.T) {
		handlerID := "agents.tests.mcpload.tools"
		handler, exists := process.Handlers[handlerID]
		assert.True(t, exists, "Handler should be registered")

		p := &process.Process{
			ID:     handlerID + ".Calculate",
			Method: "Calculate",
			Args: []interface{}{map[string]interface{}{
				"operation": "add",
				"a":         float64(10),
				"b":         float64(5),
			}},
			Context: context.Background(),
		}

		result := handler(p)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok, "Result should be a map")
		assert.Equal(t, float64(15), resultMap["result"])
	})

	// Test 4: Unregister scripts
	t.Run("UnregisterScripts", func(t *testing.T) {
		err := ast.UnregisterScripts()
		assert.NoError(t, err)

		// Verify handlers are removed
		handlerID := "agents.tests.mcpload.tools"
		_, exists := process.Handlers[handlerID]
		assert.False(t, exists, "Handler should be unregistered")
	})
}

func TestScriptsProcessUsing(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Get the mcpload assistant
	assistantID := "tests.mcpload"
	ast, err := assistant.Get(assistantID)
	assert.NoError(t, err)
	assert.NotNil(t, ast)

	// Register scripts
	err = ast.RegisterScripts()
	assert.NoError(t, err)
	defer ast.UnregisterScripts()

	// Test 1: Call Hello using process.New().Execute()
	t.Run("ProcessHello", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.Hello", map[string]interface{}{
			"name": "Yao",
		})

		err := proc.Execute()
		assert.NoError(t, err)

		result := proc.Value()
		assert.NotNil(t, result)

		resultStr, ok := result.(string)
		assert.True(t, ok, "Result should be a string")
		assert.Contains(t, resultStr, "Hello, Yao")
	})

	// Test 2: Call Ping using process.New().Execute()
	t.Run("ProcessPing", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.Ping", map[string]interface{}{
			"message": "test message",
		})

		err := proc.Execute()
		assert.NoError(t, err)

		result := proc.Value()
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok, "Result should be a map")
		assert.Equal(t, "test message", resultMap["message"])
		assert.Contains(t, resultMap["echo"], "Pong")
	})

	// Test 3: Call Calculate using process.New().Execute()
	t.Run("ProcessCalculate", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.Calculate", map[string]interface{}{
			"operation": "multiply",
			"a":         float64(6),
			"b":         float64(7),
		})

		err := proc.Execute()
		assert.NoError(t, err)

		result := proc.Value()
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok, "Result should be a map")
		assert.Equal(t, float64(42), resultMap["result"])
		assert.Equal(t, "multiply", resultMap["operation"])
	})
}

func TestScriptsProcessError(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Get the mcpload assistant
	assistantID := "tests.mcpload"
	ast, err := assistant.Get(assistantID)
	assert.NoError(t, err)
	assert.NotNil(t, ast)

	// Register scripts
	err = ast.RegisterScripts()
	assert.NoError(t, err)
	defer ast.UnregisterScripts()

	// Test calling non-existent method
	t.Run("CallNonExistentMethod", func(t *testing.T) {
		proc := process.New("agents.tests.mcpload.tools.NonExistent")

		err := proc.Execute()
		assert.NotNil(t, err, "Should return error when calling non-existent method")
		assert.Contains(t, err.Error(), "Exception|500")
	})
}
