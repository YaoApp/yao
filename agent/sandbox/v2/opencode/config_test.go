package opencode

import (
	"encoding/json"
	"testing"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
)

func TestBuildOpenCodeConfig_Defaults(t *testing.T) {
	req := &types.PrepareRequest{
		AssistantID: "test-assistant",
		Config:      &types.SandboxConfig{},
	}

	data := buildOpenCodeConfig(req, nil)
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if cfg["autoupdate"] != false {
		t.Errorf("autoupdate should be false, got %v", cfg["autoupdate"])
	}
	if cfg["snapshot"] != false {
		t.Errorf("snapshot should be false, got %v", cfg["snapshot"])
	}
	if cfg["share"] != "disabled" {
		t.Errorf("share should be disabled, got %v", cfg["share"])
	}

	instructions, ok := cfg["instructions"].([]any)
	if !ok || len(instructions) == 0 {
		t.Fatal("instructions should be a non-empty array")
	}
	if instructions[0] != ".yao/assistants/test-assistant/system-prompt.md" {
		t.Errorf("instructions[0] = %q, want .yao/assistants/test-assistant/system-prompt.md", instructions[0])
	}

	watcher, ok := cfg["watcher"].(map[string]any)
	if !ok {
		t.Fatal("watcher should be a map")
	}
	ignore, ok := watcher["ignore"].([]any)
	if !ok || len(ignore) < 2 {
		t.Errorf("watcher.ignore should have at least 2 entries, got %v", ignore)
	}
}

func TestBuildOpenCodeConfig_NoAssistantID(t *testing.T) {
	req := &types.PrepareRequest{
		AssistantID: "",
		Config:      &types.SandboxConfig{},
	}

	data := buildOpenCodeConfig(req, nil)
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	instructions := cfg["instructions"].([]any)
	if instructions[0] != ".opencode/system-prompt.md" {
		t.Errorf("instructions[0] = %q, want .opencode/system-prompt.md", instructions[0])
	}
}

func TestBuildOpenCodeConfig_WithMCP(t *testing.T) {
	req := &types.PrepareRequest{
		AssistantID: "test",
		Config:      &types.SandboxConfig{},
	}
	servers := []types.MCPServer{
		{ServerID: "my-server"},
		{ServerID: "another-server"},
	}

	data := buildOpenCodeConfig(req, servers)
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	mcp, ok := cfg["mcp"].(map[string]any)
	if !ok {
		t.Fatal("mcp should be a map")
	}
	if _, exists := mcp["my-server"]; !exists {
		t.Error("mcp should contain my-server")
	}
	if _, exists := mcp["another-server"]; !exists {
		t.Error("mcp should contain another-server")
	}
}

func TestBuildMCPConfig_Default(t *testing.T) {
	result := buildMCPConfig(nil)
	if _, ok := result["yao"]; !ok {
		t.Error("empty server list should produce default 'yao' entry")
	}
}

func TestBuildMCPConfig_WithServers(t *testing.T) {
	servers := []types.MCPServer{
		{ServerID: "server-a"},
		{ServerID: "server-b"},
		{ServerID: ""},
	}
	result := buildMCPConfig(servers)
	if _, ok := result["server-a"]; !ok {
		t.Error("should contain server-a")
	}
	if _, ok := result["server-b"]; !ok {
		t.Error("should contain server-b")
	}
	if len(result) != 2 {
		t.Errorf("should only have 2 entries (empty ID skipped), got %d", len(result))
	}

	serverA := result["server-a"].(map[string]any)
	cmd := serverA["command"].([]string)
	if len(cmd) != 3 || cmd[0] != "tai" || cmd[1] != "mcp" || cmd[2] != "server-a" {
		t.Errorf("command should be [tai mcp server-a], got %v", cmd)
	}
}
