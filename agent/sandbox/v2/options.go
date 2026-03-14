package sandboxv2

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// resolveEnvRef resolves $ENV.XXX references to os.Getenv("XXX").
func resolveEnvRef(value string) string {
	if strings.HasPrefix(value, "$ENV.") {
		return os.Getenv(value[5:])
	}
	return value
}

// BuildCreateOptions converts a SandboxConfig into the V2 infrastructure
// CreateOptions. An optional connector is used to inject OPENAI_PROXY_*
// environment variables when the connector is OpenAI-compatible (non-Anthropic).
func BuildCreateOptions(cfg *types.SandboxConfig, identifier, ownerID, workspaceID string, conn ...connector.Connector) (infra.CreateOptions, error) {
	opts := infra.CreateOptions{
		ID:          identifier,
		Owner:       ownerID,
		Image:       cfg.Computer.Image,
		WorkDir:     cfg.Computer.WorkDir,
		User:        cfg.Computer.User,
		MountPath:   cfg.Computer.MountPath,
		MountMode:   cfg.Computer.MountMode,
		WorkspaceID: workspaceID,
		Labels:      cfg.Labels,
		DisplayName: cfg.DisplayName,
	}

	if opts.Labels == nil {
		opts.Labels = make(map[string]string)
	}

	// Lifecycle policy
	switch cfg.Lifecycle {
	case "oneshot":
		opts.Policy = infra.OneShot
	case "session":
		opts.Policy = infra.Session
	case "longrunning":
		opts.Policy = infra.LongRunning
	case "persistent":
		opts.Policy = infra.Persistent
	default:
		opts.Policy = infra.OneShot
	}

	// Timeouts
	if cfg.IdleTimeout != "" {
		d, err := time.ParseDuration(cfg.IdleTimeout)
		if err != nil {
			return opts, fmt.Errorf("idle_timeout: %w", err)
		}
		opts.IdleTimeout = d
	}
	if cfg.MaxLifetime != "" {
		d, err := time.ParseDuration(cfg.MaxLifetime)
		if err != nil {
			return opts, fmt.Errorf("max_lifetime: %w", err)
		}
		opts.MaxLifetime = d
	}
	if cfg.StopTimeout != "" {
		d, err := time.ParseDuration(cfg.StopTimeout)
		if err != nil {
			return opts, fmt.Errorf("stop_timeout: %w", err)
		}
		opts.StopTimeout = d
	}

	// Memory (string like "4g" → bytes)
	if cfg.Computer.Memory != "" {
		mem, err := parseMemory(cfg.Computer.Memory)
		if err != nil {
			return opts, fmt.Errorf("memory: %w", err)
		}
		opts.Memory = mem
	}

	opts.CPUs = cfg.Computer.CPUs

	// VNC
	opts.VNC = cfg.Computer.VNC.Enabled

	// Ports
	for _, p := range cfg.Computer.Ports {
		opts.Ports = append(opts.Ports, infra.PortMapping{
			ContainerPort: p.Port,
			HostPort:      p.HostPort,
			Protocol:      p.Protocol,
		})
	}

	// NodeID (host mode pre-selection)
	if cfg.NodeID != "" {
		opts.NodeID = cfg.NodeID
	}

	// Merge environment + secrets into CreateOptions.Env.
	// Secrets override environment for same-name keys.
	// $ENV.XXX references are resolved at runtime.
	envSize := len(cfg.Environment) + len(cfg.Secrets)
	if envSize > 0 {
		opts.Env = make(map[string]string, envSize)
		for k, v := range cfg.Environment {
			opts.Env[k] = resolveEnvRef(v)
		}
		for k, v := range cfg.Secrets {
			opts.Env[k] = resolveEnvRef(v)
		}
	}

	if opts.Env == nil {
		opts.Env = make(map[string]string)
	}

	// Inject OPENAI_PROXY_* when connector is OpenAI-compatible (non-Anthropic).
	// The a2o proxy inside the container translates Anthropic API → OpenAI API.
	if len(conn) > 0 && conn[0] != nil && !conn[0].Is(connector.ANTHROPIC) {
		injectProxyEnv(opts.Env, conn[0])
	}

	// Inject VNC_* environment variables from config.
	if cfg.Computer.VNC.Enabled {
		opts.Env["VNC_ENABLED"] = "true"
		if cfg.Computer.VNC.Password != "" {
			opts.Env["VNC_PASSWORD"] = resolveEnvRef(cfg.Computer.VNC.Password)
		}
		if cfg.Computer.VNC.Resolution != "" {
			opts.Env["VNC_RESOLUTION"] = cfg.Computer.VNC.Resolution
		}
		if cfg.Computer.VNC.ViewOnly {
			opts.Env["VNC_VIEW_ONLY"] = "true"
		}
	}

	return opts, nil
}

// injectProxyEnv extracts backend URL, model, and API key from an
// OpenAI-compatible connector's settings and writes them as OPENAI_PROXY_*
// environment variables into env.
func injectProxyEnv(env map[string]string, conn connector.Connector) {
	settings := conn.Setting()
	if settings == nil {
		return
	}

	if host, ok := settings["host"].(string); ok && host != "" {
		env["OPENAI_PROXY_BACKEND"] = host
	}
	if model, ok := settings["model"].(string); ok && model != "" {
		env["OPENAI_PROXY_MODEL"] = model
	}
	if key, ok := settings["key"].(string); ok && key != "" {
		env["OPENAI_PROXY_API_KEY"] = key
	}

	// Forward extra options as JSON.
	extra := make(map[string]interface{})
	for k, v := range settings {
		switch k {
		case "host", "model", "key", "proxy", "type":
			continue
		default:
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		if data, err := json.Marshal(extra); err == nil {
			env["OPENAI_PROXY_OPTIONS"] = string(data)
		}
	}
}

// parseMemory converts a human-readable memory string to bytes.
// Supported formats: "4GB", "4G", "4g", "512MB", "512M", "512m", "1024KB", "1024K", "1024".
func parseMemory(s string) (int64, error) {
	if len(s) == 0 {
		return 0, nil
	}

	upper := strings.ToUpper(s)
	var num string
	var multiplier int64

	switch {
	case strings.HasSuffix(upper, "GB"):
		num = s[:len(s)-2]
		multiplier = 1 << 30
	case strings.HasSuffix(upper, "MB"):
		num = s[:len(s)-2]
		multiplier = 1 << 20
	case strings.HasSuffix(upper, "KB"):
		num = s[:len(s)-2]
		multiplier = 1 << 10
	case strings.HasSuffix(upper, "TB"):
		num = s[:len(s)-2]
		multiplier = 1 << 40
	case strings.HasSuffix(upper, "G"):
		num = s[:len(s)-1]
		multiplier = 1 << 30
	case strings.HasSuffix(upper, "M"):
		num = s[:len(s)-1]
		multiplier = 1 << 20
	case strings.HasSuffix(upper, "K"):
		num = s[:len(s)-1]
		multiplier = 1 << 10
	case strings.HasSuffix(upper, "T"):
		num = s[:len(s)-1]
		multiplier = 1 << 40
	default:
		num = s
		multiplier = 1
	}

	var val float64
	if _, err := fmt.Sscanf(num, "%f", &val); err != nil {
		return 0, fmt.Errorf("invalid memory value %q", s)
	}
	return int64(val * float64(multiplier)), nil
}
