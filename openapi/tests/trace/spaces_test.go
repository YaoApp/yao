package trace_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/trace/types"
)

func TestGetSpaces(t *testing.T) {
	// Prepare test trace with spaces
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Create additional spaces for this test
	space1, err := data.Manager.CreateSpace(types.TraceSpaceOption{
		Label:       "Memory Space",
		Type:        "memory",
		Icon:        "memory",
		Description: "Test memory space",
	})
	assert.NoError(t, err)

	space2, err := data.Manager.CreateSpace(types.TraceSpaceOption{
		Label:       "Cache Space",
		Type:        "cache",
		Icon:        "cache",
		Description: "Test cache space",
	})
	assert.NoError(t, err)

	// Add some data to spaces
	err = data.Manager.SetSpaceValue(space1.ID, "key1", "value1")
	assert.NoError(t, err)
	err = data.Manager.SetSpaceValue(space2.ID, "key2", "value2")
	assert.NoError(t, err)

	// Make API request
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/spaces", data.ServerURL, data.BaseURL, data.TraceID)
	req, _ := http.NewRequest("GET", requestURL, nil)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Verify response
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Verify structure
	assert.Equal(t, data.TraceID, result["trace_id"])
	assert.NotNil(t, result["spaces"])
	assert.NotNil(t, result["count"])

	spaces := result["spaces"].([]any)
	assert.GreaterOrEqual(t, len(spaces), 2) // At least the 2 spaces we created, plus the one from prepareTestTrace
	assert.Equal(t, float64(len(spaces)), result["count"])

	// Verify space metadata (should not include data field)
	spaceLabels := make(map[string]bool)
	typeFound := 0
	for _, s := range spaces {
		space := s.(map[string]any)
		assert.NotNil(t, space["id"])
		assert.NotNil(t, space["label"])
		assert.NotNil(t, space["created_at"])
		assert.NotNil(t, space["updated_at"])
		assert.Nil(t, space["data"]) // Should NOT include key-value data

		// Verify type field is present
		if space["type"] != nil {
			spaceType, ok := space["type"].(string)
			assert.True(t, ok, "Type should be a string")
			assert.NotEmpty(t, spaceType, "Type should not be empty")
			typeFound++
		}

		spaceLabels[space["label"].(string)] = true
	}

	assert.GreaterOrEqual(t, typeFound, 2, "At least 2 spaces should have type field")

	assert.True(t, spaceLabels["Memory Space"])
	assert.True(t, spaceLabels["Cache Space"])

	t.Logf("Retrieved %d spaces for trace %s", len(spaces), data.TraceID)
}

func TestGetSpaceByID(t *testing.T) {
	// Prepare test trace
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Create a space with specific data
	space, err := data.Manager.CreateSpace(types.TraceSpaceOption{
		Label:       "Detailed Space",
		Type:        "detailed",
		Icon:        "memory",
		Description: "Space with detailed data",
		Metadata:    map[string]any{"cache_enabled": true},
	})
	assert.NoError(t, err)

	// Add key-value data
	err = data.Manager.SetSpaceValue(space.ID, "key1", "value1")
	assert.NoError(t, err)
	err = data.Manager.SetSpaceValue(space.ID, "key2", 123)
	assert.NoError(t, err)
	err = data.Manager.SetSpaceValue(space.ID, "key3", map[string]any{"nested": "data"})
	assert.NoError(t, err)

	// Make API request
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/spaces/%s", data.ServerURL, data.BaseURL, data.TraceID, space.ID)
	req, _ := http.NewRequest("GET", requestURL, nil)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Verify response
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Verify space metadata
	assert.Equal(t, space.ID, result["id"])
	assert.Equal(t, "Detailed Space", result["label"])
	assert.Equal(t, "detailed", result["type"])
	assert.Equal(t, "memory", result["icon"])
	assert.Equal(t, "Space with detailed data", result["description"])
	assert.NotNil(t, result["created_at"])
	assert.NotNil(t, result["updated_at"])

	// Verify metadata
	metadata := result["metadata"].(map[string]any)
	assert.Equal(t, true, metadata["cache_enabled"])

	// Verify key-value data
	spaceData := result["data"].(map[string]any)
	assert.Len(t, spaceData, 3)
	assert.Equal(t, "value1", spaceData["key1"])
	assert.Equal(t, float64(123), spaceData["key2"]) // JSON numbers are float64
	nestedData := spaceData["key3"].(map[string]any)
	assert.Equal(t, "data", nestedData["nested"])

	t.Logf("Retrieved space %s with %d key-value pairs from trace %s", space.ID, len(spaceData), data.TraceID)
}

func TestGetSpaceByIDNotFound(t *testing.T) {
	// Prepare test trace
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Make API request with non-existent space ID
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/spaces/non_existent_space", data.ServerURL, data.BaseURL, data.TraceID)
	req, _ := http.NewRequest("GET", requestURL, nil)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Verify 404 response
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.NotNil(t, result["error"])
}
