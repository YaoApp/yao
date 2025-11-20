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

// TestGetLogs tests the get all logs API endpoint
func TestGetLogs(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Test GET /traces/:traceID/logs
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/logs", data.ServerURL, data.BaseURL, data.TraceID)

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
	assert.Equal(t, data.TraceID, responseData["trace_id"], "Trace ID should match")
	assert.NotNil(t, responseData["logs"], "Should have logs field")
	assert.NotNil(t, responseData["count"], "Should have count field")

	logs, ok := responseData["logs"].([]interface{})
	assert.True(t, ok, "Logs should be an array")
	assert.NotEmpty(t, logs, "Logs array should not be empty")

	count := int(responseData["count"].(float64))
	assert.GreaterOrEqual(t, count, 6, "Should have at least 6 log entries (6 node logs)")

	// Verify log structure and collect log levels
	logLevels := make(map[string]int)
	for _, l := range logs {
		log, ok := l.(map[string]interface{})
		assert.True(t, ok, "Each log should be an object")
		assert.NotNil(t, log["timestamp"], "Log should have timestamp")
		assert.NotEmpty(t, log["level"], "Log should have level")
		assert.NotEmpty(t, log["message"], "Log should have message")

		level := log["level"].(string)
		logLevels[level]++
	}

	assert.Greater(t, logLevels["info"], 0, "Should have info logs")
	assert.Greater(t, logLevels["debug"], 0, "Should have debug logs")
	assert.Greater(t, logLevels["warn"], 0, "Should have warn logs")
	assert.Greater(t, logLevels["error"], 0, "Should have error logs")

	t.Logf("Retrieved %d logs for trace %s (info: %d, debug: %d, warn: %d, error: %d)",
		count, data.TraceID, logLevels["info"], logLevels["debug"], logLevels["warn"], logLevels["error"])
}

// TestGetLogsByNode tests the get logs by node ID API endpoint
func TestGetLogsByNode(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Test GET /traces/:traceID/logs/:nodeID with Node1
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/logs/%s", data.ServerURL, data.BaseURL, data.TraceID, data.Node1ID)

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
	assert.Equal(t, data.TraceID, responseData["trace_id"], "Trace ID should match")
	assert.Equal(t, data.Node1ID, responseData["node_id"], "Node ID should match")
	assert.NotNil(t, responseData["logs"], "Should have logs field")

	logs, ok := responseData["logs"].([]interface{})
	assert.True(t, ok, "Logs should be an array")
	assert.NotEmpty(t, logs, "Logs array should not be empty")

	// Verify all logs belong to the specific node
	for _, l := range logs {
		log, ok := l.(map[string]interface{})
		assert.True(t, ok, "Each log should be an object")
		assert.Equal(t, data.Node1ID, log["node_id"], "All logs should belong to the specified node")
		assert.NotEmpty(t, log["message"], "Log should have message")
	}

	// Should have at least 2 logs for Node1 (info + debug)
	count := int(responseData["count"].(float64))
	assert.GreaterOrEqual(t, count, 2, "Node1 should have at least 2 log entries")

	t.Logf("Retrieved %d logs for node %s in trace %s", count, data.Node1ID, data.TraceID)
}

// TestGetLogsByNodeNotFound tests getting logs for non-existent node
func TestGetLogsByNodeNotFound(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Try to get logs for non-existent node
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/logs/nonexistent", data.ServerURL, data.BaseURL, data.TraceID)

	req, err := http.NewRequest("GET", requestURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should return 200 with empty array (no logs for non-existent node)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	assert.NoError(t, err)

	logs, ok := responseData["logs"].([]interface{})
	assert.True(t, ok, "Logs should be an array")
	assert.Empty(t, logs, "Should return empty array for non-existent node")
}
