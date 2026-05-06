package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/llmprovider"
)

func TestResolveConnectorForResponse_ExplicitID(t *testing.T) {
	resolved, raw := resolveConnectorForResponse("some-connector-id", nil)
	assert.Equal(t, "some-connector-id", resolved)
	assert.Empty(t, raw, "non use:: prefix should not set raw value")
}

func TestResolveConnectorForResponse_ExplicitIDWithModel(t *testing.T) {
	resolved, raw := resolveConnectorForResponse("t123.openai:gpt-4o", nil)
	assert.Equal(t, "t123.openai:gpt-4o", resolved)
	assert.Empty(t, raw)
}

func TestResolveConnectorForResponse_EmptyString(t *testing.T) {
	resolved, raw := resolveConnectorForResponse("", nil)
	assert.Empty(t, resolved)
	assert.Empty(t, raw)
}

func TestResolveConnectorForResponse_NilGlobal(t *testing.T) {
	orig := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = orig }()

	resolved, raw := resolveConnectorForResponse("use::default", nil)
	assert.Equal(t, "use::default", resolved, "should return original when Global is nil")
	assert.Equal(t, "use::default", raw)
}

func TestResolveConnectorForResponse_UseUnresolvable(t *testing.T) {
	orig := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = orig }()

	resolved, raw := resolveConnectorForResponse("use::light", nil)
	assert.Equal(t, "use::light", resolved, "unresolvable role returns original")
	assert.Equal(t, "use::light", raw)
}

func TestResolveConnectorForResponse_UseEmptyRole(t *testing.T) {
	orig := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = orig }()

	resolved, raw := resolveConnectorForResponse("use::", nil)
	assert.Equal(t, "use::", resolved, "empty role with nil Global returns original")
	assert.Equal(t, "use::", raw)
}

func TestApplyConnectorResolve_NoConnector(t *testing.T) {
	result := map[string]interface{}{"name": "test"}
	applyConnectorResolve(result, nil)
	assert.Nil(t, result["connector_raw"], "should not add connector_raw when no connector")
}

func TestApplyConnectorResolve_ExplicitConnector(t *testing.T) {
	result := map[string]interface{}{"connector": "t123.openai:gpt-4o"}
	applyConnectorResolve(result, nil)
	assert.Equal(t, "t123.openai:gpt-4o", result["connector"])
	assert.Nil(t, result["connector_raw"], "should not add connector_raw for explicit IDs")
}

func TestApplyConnectorResolve_UsePrefix(t *testing.T) {
	orig := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = orig }()

	result := map[string]interface{}{"connector": "use::default"}
	applyConnectorResolve(result, nil)
	assert.Equal(t, "use::default", result["connector"], "unresolvable returns original")
	assert.Equal(t, "use::default", result["connector_raw"])
}
