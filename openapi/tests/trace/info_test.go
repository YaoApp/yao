package trace_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGetInfo tests the trace info API endpoint
func TestGetInfo(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Test GET /traces/:traceID/info
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/info", data.ServerURL, data.BaseURL, data.TraceID)

	req, err := http.NewRequest("GET", requestURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

	// Parse response
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	assert.NoError(t, err)

	// Verify response structure
	assert.Equal(t, data.TraceID, responseData["id"], "Trace ID should match")
	assert.Equal(t, "local", responseData["driver"], "Driver should be local")
	assert.NotNil(t, responseData["status"], "Should have status field")
	assert.NotNil(t, responseData["created_at"], "Should have created_at field")
	assert.NotNil(t, responseData["updated_at"], "Should have updated_at field")

	// Verify metadata
	metadata, ok := responseData["metadata"].(map[string]interface{})
	assert.True(t, ok, "Should have metadata")
	assert.Equal(t, "api_test", metadata["test_type"], "Metadata should match")
	assert.Equal(t, "common_trace_data", metadata["test_name"], "Metadata should match")

	// Verify user info
	assert.Equal(t, data.TokenInfo.UserID, responseData["created_by"], "Created by should match")

	t.Logf("Retrieved trace info for %s", data.TraceID)
}

// TestGetInfoNotFound tests getting info for non-existent trace
func TestGetInfoNotFound(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Try to get info for non-existent trace
	requestURL := fmt.Sprintf("%s%s/trace/traces/nonexistent/info", data.ServerURL, data.BaseURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Expected status code 404 for non-existent trace")
}
