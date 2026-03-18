package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	sandboxTypes "github.com/yaoapp/yao/agent/sandbox/v2/types"
)

// LoadSandboxConfig reads a sandbox.yao file (JSON or YAML) and returns
// the V2 SandboxConfig. Called during Assistant.Load().
func LoadSandboxConfig(filePath string) (*sandboxTypes.SandboxConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read sandbox config %s: %w", filePath, err)
	}

	var cfg sandboxTypes.SandboxConfig
	if err := application.Parse(filepath.Base(filePath), data, &cfg); err != nil {
		return nil, fmt.Errorf("parse sandbox config: %w", err)
	}

	if cfg.Version != sandboxTypes.SandboxVersionV2 {
		return nil, fmt.Errorf("sandbox.yao version must be %q, got %q", sandboxTypes.SandboxVersionV2, cfg.Version)
	}

	return &cfg, nil
}

// ToSandboxV2 converts a generic value (typically map[string]any from DSL
// parsing) into a V2 SandboxConfig.
func ToSandboxV2(v any) (*sandboxTypes.SandboxConfig, error) {
	if v == nil {
		return nil, nil
	}

	switch sb := v.(type) {
	case *sandboxTypes.SandboxConfig:
		return sb, nil
	case sandboxTypes.SandboxConfig:
		return &sb, nil
	default:
		raw, err := jsoniter.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("sandbox v2 format error: %w", err)
		}
		var cfg sandboxTypes.SandboxConfig
		if err := jsoniter.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("sandbox v2 format error: %w", err)
		}
		return &cfg, nil
	}
}

// ComputeConfigHash computes a SHA-256 fingerprint of the sandbox configuration,
// MCP servers, and skills directory. Used for hot-reload detection in prepare
// step "once" logic.
func ComputeConfigHash(cfg *sandboxTypes.SandboxConfig, mcpServers []MCPServerConfig, skillsDir string) string {
	h := sha256.New()

	raw, _ := json.Marshal(cfg)
	h.Write(raw)

	if len(mcpServers) > 0 {
		mcpRaw, _ := json.Marshal(mcpServers)
		h.Write(mcpRaw)
	}

	if skillsDir != "" {
		h.Write([]byte(skillsDir))
		entries, err := os.ReadDir(skillsDir)
		if err == nil {
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				names = append(names, e.Name())
			}
			sort.Strings(names)
			for _, n := range names {
				h.Write([]byte(n))
			}
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
