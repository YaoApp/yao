//go:build unit

package event_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	event "github.com/yaoapp/yao/openapi/event"
)

func TestStripInternalFields_NilPayload(t *testing.T) {
	result := event.ExportStripInternalFields(nil)
	assert.Nil(t, result)
}

func TestStripInternalFields_EmptyMap(t *testing.T) {
	result := event.ExportStripInternalFields(map[string]any{})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestStripInternalFields_NoInternalFields(t *testing.T) {
	payload := map[string]any{
		"chat_id":    "c1",
		"title":      "hello",
		"run_status": "running",
	}
	result := event.ExportStripInternalFields(payload)
	assert.Equal(t, payload, result)
}

func TestStripInternalFields_RemovesYaoPrefixedFields(t *testing.T) {
	payload := map[string]any{
		"chat_id":          "c1",
		"__yao_team_id":    "t1",
		"__yao_created_by": "u1",
		"run_status":       "done",
	}
	result := event.ExportStripInternalFields(payload)
	assert.Equal(t, map[string]any{"chat_id": "c1", "run_status": "done"}, result)
}

func TestStripInternalFields_AllInternalFields(t *testing.T) {
	payload := map[string]any{
		"__yao_team_id":    "t1",
		"__yao_created_by": "u1",
		"__yao_other":      "x",
	}
	result := event.ExportStripInternalFields(payload)
	assert.Empty(t, result)
}

func TestStripInternalFields_PreservesNonYaoUnderscore(t *testing.T) {
	payload := map[string]any{
		"__other_field": "keep",
		"__yao_secret":  "remove",
		"normal":        "keep",
	}
	result := event.ExportStripInternalFields(payload)
	assert.Equal(t, map[string]any{"__other_field": "keep", "normal": "keep"}, result)
}

func TestStripInternalFields_DoesNotMutateOriginal(t *testing.T) {
	payload := map[string]any{
		"chat_id":       "c1",
		"__yao_team_id": "t1",
	}
	_ = event.ExportStripInternalFields(payload)
	assert.Contains(t, payload, "__yao_team_id", "original should not be mutated")
}
