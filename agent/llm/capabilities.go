package llm

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
)

// GetCapabilities get the capabilities of a connector by connector ID
// This is a unified function to get connector capabilities with proper priority:
// 1. User-defined model capabilities from agent/models.yml (passed via modelCapabilities map)
// 2. Connector's Setting()["capabilities"] (default capabilities from connector)
// 3. Fallback to minimal default capabilities
//
// Usage in Agent with user-defined models:
//
//	capabilities := llm.GetCapabilities(connectorID, modelCapabilities)
//
// Usage in API (without user-defined models):
//
//	capabilities := llm.GetCapabilities(connectorID, nil)
func GetCapabilities(connectorID string, modelCapabilities map[string]openai.Capabilities) *openai.Capabilities {
	if connectorID == "" {
		return getDefaultCapabilities()
	}

	// Priority 1: Check user-defined model capabilities from agent/models.yml
	if modelCapabilities != nil {
		if modelCaps, exists := modelCapabilities[connectorID]; exists {
			return &modelCaps
		}
	}

	// Priority 2: Get connector and extract capabilities from Setting()
	conn, err := connector.Select(connectorID)
	if err != nil {
		// If connector not found, return default
		return getDefaultCapabilities()
	}

	return GetCapabilitiesFromConn(conn, modelCapabilities)
}

// GetCapabilitiesFromConn get the capabilities from a connector instance
// This is useful when you already have the connector object
func GetCapabilitiesFromConn(conn connector.Connector, modelCapabilities map[string]openai.Capabilities) *openai.Capabilities {
	if conn == nil {
		return getDefaultCapabilities()
	}

	connectorID := conn.ID()

	// Priority 1: Check user-defined model capabilities from agent/models.yml
	if modelCapabilities != nil {
		if modelCaps, exists := modelCapabilities[connectorID]; exists {
			return &modelCaps
		}
	}

	// Priority 2: Get capabilities from connector's Setting() method
	settings := conn.Setting()
	if settings != nil {
		if caps, ok := settings["capabilities"]; ok {
			// Try to convert to *openai.Capabilities
			if capabilities, ok := caps.(*openai.Capabilities); ok {
				return capabilities
			}
			// Try to convert to openai.Capabilities (value type)
			if capabilities, ok := caps.(openai.Capabilities); ok {
				return &capabilities
			}
		}
	}

	// Priority 3: Fallback to minimal default capabilities
	return getDefaultCapabilities()
}

// getDefaultCapabilities returns minimal default capabilities
// This should rarely be used as modern connectors provide capabilities via Setting()
func getDefaultCapabilities() *openai.Capabilities {
	return &openai.Capabilities{
		Vision:                false,
		ToolCalls:             false,
		Audio:                 false,
		Reasoning:             false,
		Streaming:             false,
		JSON:                  false,
		Multimodal:            false,
		TemperatureAdjustable: true, // Default to true for non-reasoning models
	}
}

// GetCapabilitiesMap get capabilities as map[string]interface{} for API responses
// This is useful for OpenAPI responses that need JSON-serializable format
func GetCapabilitiesMap(connectorID string, modelCapabilities map[string]openai.Capabilities) map[string]interface{} {
	caps := GetCapabilities(connectorID, modelCapabilities)
	if caps == nil {
		return nil
	}

	return ToMap(caps)
}

// ToMap converts openai.Capabilities to map[string]interface{}
// This is useful for JSON serialization in API responses
func ToMap(caps *openai.Capabilities) map[string]interface{} {
	if caps == nil {
		return nil
	}

	result := make(map[string]interface{})

	// Handle Vision field specially as it can be bool or string
	if caps.Vision != nil {
		result["vision"] = caps.Vision
	}

	result["audio"] = caps.Audio
	result["tool_calls"] = caps.ToolCalls
	result["reasoning"] = caps.Reasoning
	result["streaming"] = caps.Streaming
	result["json"] = caps.JSON
	result["multimodal"] = caps.Multimodal
	result["temperature_adjustable"] = caps.TemperatureAdjustable

	return result
}
