package sandbox

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config holds sandbox configuration
type Config struct {
	Image         string        `json:"image,omitempty"`          // Docker image, default: yao/sandbox-claude:latest
	WorkspaceRoot string        `json:"workspace_root,omitempty"` // Host workspace root directory
	IPCDir        string        `json:"ipc_dir,omitempty"`        // IPC socket directory
	MaxContainers int           `json:"max_containers,omitempty"` // Maximum concurrent containers
	IdleTimeout   time.Duration `json:"idle_timeout,omitempty"`   // Idle timeout before stopping container
	MaxMemory     string        `json:"max_memory,omitempty"`     // Memory limit, e.g., "2g"
	MaxCPU        float64       `json:"max_cpu,omitempty"`        // CPU limit, e.g., 1.0

	// Container internal paths
	ContainerWorkDir   string `json:"container_workdir,omitempty"`    // Container working directory, default: /workspace
	ContainerIPCSocket string `json:"container_ipc_socket,omitempty"` // Container IPC socket path, default: /tmp/yao.sock
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		Image:              "yaoapp/sandbox-claude:latest",
		MaxContainers:      100,
		IdleTimeout:        30 * time.Minute,
		MaxMemory:          "2g",
		MaxCPU:             1.0,
		ContainerWorkDir:   "/workspace",
		ContainerIPCSocket: "/tmp/yao.sock",
	}
}

// Init initializes the config with defaults based on environment variables and data root
func (c *Config) Init(dataRoot string) {
	// Image
	if env := os.Getenv("YAO_SANDBOX_IMAGE"); env != "" {
		c.Image = env
	} else if c.Image == "" {
		c.Image = "yaoapp/sandbox-claude:latest"
	}

	// Workspace root
	if env := os.Getenv("YAO_SANDBOX_WORKSPACE"); env != "" {
		c.WorkspaceRoot = env
	} else if c.WorkspaceRoot == "" {
		c.WorkspaceRoot = filepath.Join(dataRoot, "sandbox", "workspace")
	}

	// IPC directory
	if env := os.Getenv("YAO_SANDBOX_IPC"); env != "" {
		c.IPCDir = env
	} else if c.IPCDir == "" {
		c.IPCDir = filepath.Join(dataRoot, "sandbox", "ipc")
	}

	// Max containers - set default first if zero, then try env override
	if c.MaxContainers == 0 {
		c.MaxContainers = 100
	}
	if env := os.Getenv("YAO_SANDBOX_MAX"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			c.MaxContainers = v
		}
		// Invalid env value: keep existing/default value
	}

	// Idle timeout - set default first if zero, then try env override
	if c.IdleTimeout == 0 {
		c.IdleTimeout = 30 * time.Minute
	}
	if env := os.Getenv("YAO_SANDBOX_IDLE_TIMEOUT"); env != "" {
		if v, err := time.ParseDuration(env); err == nil && v > 0 {
			c.IdleTimeout = v
		}
		// Invalid env value: keep existing/default value
	}

	// Max memory
	if env := os.Getenv("YAO_SANDBOX_MEMORY"); env != "" {
		c.MaxMemory = env
	} else if c.MaxMemory == "" {
		c.MaxMemory = "2g"
	}

	// Max CPU - set default first if zero, then try env override
	if c.MaxCPU == 0 {
		c.MaxCPU = 1.0
	}
	if env := os.Getenv("YAO_SANDBOX_CPU"); env != "" {
		if v, err := strconv.ParseFloat(env, 64); err == nil && v > 0 {
			c.MaxCPU = v
		}
		// Invalid env value: keep existing/default value
	}

	// Container internal paths
	if env := os.Getenv("YAO_SANDBOX_CONTAINER_WORKDIR"); env != "" {
		c.ContainerWorkDir = env
	} else if c.ContainerWorkDir == "" {
		c.ContainerWorkDir = "/workspace"
	}

	if env := os.Getenv("YAO_SANDBOX_CONTAINER_IPC"); env != "" {
		c.ContainerIPCSocket = env
	} else if c.ContainerIPCSocket == "" {
		c.ContainerIPCSocket = "/tmp/yao.sock"
	}
}
