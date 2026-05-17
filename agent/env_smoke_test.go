package agent_test

import (
	"net/http"
	"os"
	"testing"
	"time"
)

// TestEnvironmentReady validates that the agent test environment is fully
// operational before any other tests run. It checks:
//   - YAO_AGENT_TEST_APPLICATION points to a valid app directory
//   - mock-llm is reachable and healthy
//   - yao-server /.well-known/yao responds 200
//
// This test should Fatal (not Skip) on missing env — run bin/test-agent start first.
func TestEnvironmentReady(t *testing.T) {
	appDir := os.Getenv("YAO_AGENT_TEST_APPLICATION")
	if appDir == "" {
		t.Fatal("YAO_AGENT_TEST_APPLICATION is not set — run bin/test-agent start first")
	}

	if _, err := os.Stat(appDir + "/app.yao"); os.IsNotExist(err) {
		t.Fatalf("app.yao not found in %s", appDir)
	}

	mockHost := os.Getenv("MOCK_LLM_HOST")
	if mockHost == "" {
		t.Fatal("MOCK_LLM_HOST is not set")
	}

	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(mockHost + "/healthz")
	if err != nil {
		t.Fatalf("mock-llm health check failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("mock-llm returned status %d", resp.StatusCode)
	}

	yaoPort := os.Getenv("YAO_PORT")
	if yaoPort == "" {
		yaoPort = "6099"
	}
	resp2, err := client.Get("http://127.0.0.1:" + yaoPort + "/.well-known/yao")
	if err != nil {
		t.Fatalf("yao-server health check failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("yao-server /.well-known/yao returned status %d", resp2.StatusCode)
	}

	t.Log("All environment checks passed")
}
