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

// TestGetNodes tests the get all nodes API endpoint
func TestGetNodes(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Test GET /traces/:traceID/nodes
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/nodes", data.ServerURL, data.BaseURL, data.TraceID)

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
	assert.NotNil(t, responseData["nodes"], "Should have nodes field")
	assert.NotNil(t, responseData["count"], "Should have count field")

	nodes, ok := responseData["nodes"].([]interface{})
	assert.True(t, ok, "Nodes should be an array")
	assert.NotEmpty(t, nodes, "Nodes array should not be empty")

	count := int(responseData["count"].(float64))
	assert.Equal(t, 3, count, "Should have 3 nodes (3 child nodes created)")
	assert.Equal(t, count, len(nodes), "Count should match array length")

	// Verify node structure and metadata
	metadataFound := 0
	typeFound := 0
	for _, n := range nodes {
		node, ok := n.(map[string]interface{})
		assert.True(t, ok, "Each node should be an object")
		assert.NotEmpty(t, node["id"], "Node should have ID")
		assert.NotNil(t, node["label"], "Node should have label")
		assert.NotNil(t, node["status"], "Node should have status")
		assert.NotNil(t, node["created_at"], "Node should have created_at")

		// Verify type field is present
		if node["type"] != nil {
			nodeType, ok := node["type"].(string)
			assert.True(t, ok, "Type should be a string")
			assert.NotEmpty(t, nodeType, "Type should not be empty")
			typeFound++
		}

		// Verify parent_ids field (should be array or null)
		if node["parent_ids"] != nil {
			_, ok := node["parent_ids"].([]interface{})
			assert.True(t, ok, "parent_ids should be an array")
		}

		// Check if metadata is present (should be for all our test nodes)
		if node["metadata"] != nil {
			metadata, ok := node["metadata"].(map[string]interface{})
			assert.True(t, ok, "Metadata should be a map")
			if nodeOrder, exists := metadata["node_order"]; exists {
				assert.NotNil(t, nodeOrder, "node_order should exist in metadata")
				metadataFound++
			}
		}
	}

	assert.Equal(t, 3, typeFound, "All 3 nodes should have type field")

	assert.Equal(t, 3, metadataFound, "All 3 nodes should have metadata with node_order")

	t.Logf("Retrieved %d nodes for trace %s (all with metadata)", count, data.TraceID)
}

// TestGetNodeByID tests the get single node API endpoint
func TestGetNodeByID(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Test GET /traces/:traceID/nodes/:nodeID with Node1
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/nodes/%s", data.ServerURL, data.BaseURL, data.TraceID, data.Node1ID)

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
	assert.Equal(t, data.Node1ID, responseData["id"], "Node ID should match")
	assert.Equal(t, "First Node", responseData["label"], "Node label should match")
	assert.Equal(t, "agent", responseData["type"], "Node type should match")
	assert.Equal(t, "icon1", responseData["icon"], "Node icon should match")
	assert.Equal(t, "First test node", responseData["description"], "Node description should match")

	// Verify parent_ids field (should be array or null)
	if responseData["parent_ids"] != nil {
		parentIDs, ok := responseData["parent_ids"].([]interface{})
		assert.True(t, ok, "parent_ids should be an array")
		t.Logf("Node has %d parent(s)", len(parentIDs))
	}

	// Verify metadata is present and correct
	assert.NotNil(t, responseData["metadata"], "Metadata should be present")
	metadata, ok := responseData["metadata"].(map[string]interface{})
	assert.True(t, ok, "Metadata should be a map")
	assert.Equal(t, float64(1), metadata["node_order"], "Metadata node_order should be 1")

	// Verify input and output are present
	assert.NotNil(t, responseData["input"], "Input should be present")
	assert.NotNil(t, responseData["output"], "Output should be present")

	t.Logf("Retrieved node %s (type: %s) from trace %s with metadata: %+v", data.Node1ID, responseData["type"], data.TraceID, metadata)
}

// TestGetNodeByIDNotFound tests getting a non-existent node
func TestGetNodeByIDNotFound(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Try to get non-existent node
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/nodes/nonexistent", data.ServerURL, data.BaseURL, data.TraceID)

	req, err := http.NewRequest("GET", requestURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Expected status code 404 for non-existent node")
}
