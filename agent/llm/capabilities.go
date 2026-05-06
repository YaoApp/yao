package llm

import (
	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
)

// GetCapabilities get the capabilities of a connector by connector ID
// Reads capabilities from connector's Setting()["capabilities"], with fallback to defaults.
func GetCapabilities(connectorID string) *goullm.Capabilities {
	if connectorID == "" {
		return getDefaultCapabilities()
	}

	conn, err := connector.Select(connectorID)
	if err != nil {
		return getDefaultCapabilities()
	}

	return GetCapabilitiesFromConn(conn)
}

// GetCapabilitiesFromConn get the capabilities from a connector instance.
// Prefers LLMConnector.GetCapabilities() when available, falls back to Setting() parsing.
func GetCapabilitiesFromConn(conn connector.Connector) *goullm.Capabilities {
	if conn == nil {
		return getDefaultCapabilities()
	}

	// Prefer typed LLMConnector interface
	if lc, ok := conn.(goullm.LLMConnector); ok {
		if caps := lc.GetCapabilities(); caps != nil {
			return caps
		}
	}

	// Fallback to Setting() parsing for non-LLMConnector or nil capabilities
	settings := conn.Setting()
	if settings != nil {
		if caps, ok := settings["capabilities"]; ok {
			if capabilities, ok := caps.(*goullm.Capabilities); ok {
				return capabilities
			}
			if capabilities, ok := caps.(goullm.Capabilities); ok {
				return &capabilities
			}
			if capsMap, ok := caps.(map[string]interface{}); ok {
				return capabilitiesFromMap(capsMap)
			}
		}
	}

	return getDefaultCapabilities()
}

// capabilitiesFromMap converts a JSON-deserialized map into goullm.Capabilities.
func capabilitiesFromMap(m map[string]interface{}) *goullm.Capabilities {
	caps := getDefaultCapabilities()
	if v, ok := m["streaming"].(bool); ok {
		caps.Streaming = v
	}
	if v, ok := m["tool_calls"].(bool); ok {
		caps.ToolCalls = v
	}
	if v, ok := m["vision"]; ok {
		caps.Vision = v
	}
	if v, ok := m["audio"].(bool); ok {
		caps.Audio = v
	}
	if v, ok := m["stt"].(bool); ok {
		caps.STT = v
	}
	if v, ok := m["reasoning"].(bool); ok {
		caps.Reasoning = v
	}
	if v, ok := m["json"].(bool); ok {
		caps.JSON = v
	}
	if v, ok := m["multimodal"].(bool); ok {
		caps.Multimodal = v
	}
	if v, ok := m["temperature_adjustable"].(bool); ok {
		caps.TemperatureAdjustable = v
	}
	if v, ok := m["embedding"].(bool); ok {
		caps.Embedding = v
	}
	if v, ok := m["image_generation"].(bool); ok {
		caps.ImageGeneration = v
	}
	return caps
}

// getDefaultCapabilities returns minimal default capabilities
func getDefaultCapabilities() *goullm.Capabilities {
	return &goullm.Capabilities{
		Vision:                false,
		ToolCalls:             false,
		Audio:                 false,
		Reasoning:             false,
		Streaming:             false,
		JSON:                  false,
		Multimodal:            false,
		TemperatureAdjustable: true,
	}
}

// GetCapabilitiesMap get capabilities as map[string]interface{} for API responses
func GetCapabilitiesMap(connectorID string) map[string]interface{} {
	caps := GetCapabilities(connectorID)
	if caps == nil {
		return nil
	}

	return ToMap(caps)
}

// ToMap converts Capabilities to map[string]interface{}.
// Delegates to the canonical Capabilities.ToMap() method in gou/llm.
func ToMap(caps *goullm.Capabilities) map[string]interface{} {
	return caps.ToMap()
}
