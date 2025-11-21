package trace_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGetEvents tests the events API endpoint
func TestGetEvents(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Test GET /traces/:traceID/events
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/events", data.ServerURL, data.BaseURL, data.TraceID)

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
	assert.NotNil(t, responseData["events"], "Should have events field")

	events, ok := responseData["events"].([]interface{})
	assert.True(t, ok, "Events should be an array")
	assert.NotEmpty(t, events, "Events array should not be empty")

	t.Logf("Retrieved %d events for trace %s", len(events), data.TraceID)

	// Verify event types
	eventTypes := make(map[string]bool)
	for _, e := range events {
		event, ok := e.(map[string]interface{})
		if ok {
			eventType, _ := event["type"].(string)
			eventTypes[eventType] = true
		}
	}

	assert.True(t, eventTypes["init"], "Should have init event")
	assert.True(t, eventTypes["node_start"], "Should have node_start events")
	assert.True(t, eventTypes["node_complete"], "Should have node_complete events")
	assert.True(t, eventTypes["space_created"], "Should have space_created event")
}

// TestGetEventsNotFound tests getting events for non-existent trace
func TestGetEventsNotFound(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Try to get events for non-existent trace
	requestURL := fmt.Sprintf("%s%s/trace/traces/nonexistent/events", data.ServerURL, data.BaseURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Expected status code 404 for non-existent trace")
}

// TestGetEventsUnauthorized tests getting events without authentication
func TestGetEventsUnauthorized(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Try to get events without token
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/events", data.ServerURL, data.BaseURL, data.TraceID)

	req, err := http.NewRequest("GET", requestURL, nil)
	assert.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Expected status code 401 without authentication")
}

// TestGetEventsSSE tests the events API endpoint in SSE streaming mode
func TestGetEventsSSE(t *testing.T) {
	data := prepareTestTrace(t)
	defer cleanupTestTrace(t, data)

	// Test GET /traces/:traceID/events?stream=true
	requestURL := fmt.Sprintf("%s%s/trace/traces/%s/events?stream=true", data.ServerURL, data.BaseURL, data.TraceID)

	req, err := http.NewRequest("GET", requestURL, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+data.TokenInfo.AccessToken)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Verify SSE response headers
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"), "Expected text/event-stream content type")
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"), "Expected no-cache")
	assert.Equal(t, "keep-alive", resp.Header.Get("Connection"), "Expected keep-alive connection")

	// Read SSE events
	scanner := bufio.NewScanner(resp.Body)
	events := make([]map[string]interface{}, 0)
	var currentEvent map[string]interface{}
	eventCount := 0
	maxEvents := 50 // Limit to prevent infinite loop

	for scanner.Scan() && eventCount < maxEvents {
		line := scanner.Text()

		// SSE format: "data: {...}"
		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")

			// Check for [DONE] marker
			if dataStr == "[DONE]" {
				t.Log("Received [DONE] marker, stream completed")
				break
			}

			// Parse JSON event data
			var eventData map[string]interface{}
			if err := json.Unmarshal([]byte(dataStr), &eventData); err != nil {
				t.Logf("Failed to parse event data: %s, error: %v", dataStr, err)
				continue
			}

			currentEvent = eventData
		} else if line == "" && currentEvent != nil {
			// Empty line marks end of an event
			events = append(events, currentEvent)
			eventCount++
			currentEvent = nil
		}
	}

	assert.NoError(t, scanner.Err(), "Should not have scanner errors")
	assert.NotEmpty(t, events, "Should receive at least one SSE event")

	t.Logf("Received %d SSE events for trace %s", len(events), data.TraceID)

	// Verify event structure and types
	eventTypes := make(map[string]int)
	for i, event := range events {
		// Verify required fields
		assert.NotNil(t, event["type"], "Event %d should have type field", i)
		assert.NotNil(t, event["trace_id"], "Event %d should have trace_id field", i)
		assert.NotNil(t, event["timestamp"], "Event %d should have timestamp field", i)

		// Verify TraceID matches
		if traceID, ok := event["trace_id"].(string); ok {
			assert.Equal(t, data.TraceID, traceID, "Event %d trace_id should match", i)
		}

		// Count event types
		if eventType, ok := event["type"].(string); ok {
			eventTypes[eventType]++
		}
	}

	// Verify expected event types
	assert.Greater(t, eventTypes["init"], 0, "Should have at least one init event")
	assert.Greater(t, eventTypes["node_start"], 0, "Should have at least one node_start event")
	assert.Greater(t, eventTypes["node_complete"], 0, "Should have at least one node_complete event")
	assert.Greater(t, eventTypes["complete"], 0, "Should have at least one complete event")

	// Log event type distribution
	t.Logf("Event type distribution: %+v", eventTypes)

	// Verify event order: init should be first
	if len(events) > 0 {
		firstEventType, _ := events[0]["type"].(string)
		assert.Equal(t, "init", firstEventType, "First event should be init")
	}

	// Verify complete event is last (before [DONE])
	if len(events) > 1 {
		lastEventType, _ := events[len(events)-1]["type"].(string)
		assert.Equal(t, "complete", lastEventType, "Last event should be complete")
	}
}
