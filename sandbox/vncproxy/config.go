package vncproxy

import (
	"os"
	"strconv"
	"time"
)

// Config holds VNC proxy configuration
type Config struct {
	// Network settings
	DockerNetwork       string `json:"docker_network,omitempty"`        // Docker network name (default: bridge)
	ContainerNoVNCPort  int    `json:"container_novnc_port,omitempty"`  // noVNC port inside container (default: 6080)
	ContainerVNCPort    int    `json:"container_vnc_port,omitempty"`    // VNC port inside container (default: 5900)
	ContainerNamePrefix string `json:"container_name_prefix,omitempty"` // Container name prefix (default: yao-sandbox-)

	// Cache settings
	IPCacheTTL time.Duration `json:"ip_cache_ttl,omitempty"` // IP cache TTL (default: 30s)

	// VNC status check
	VNCCheckTimeout time.Duration `json:"vnc_check_timeout,omitempty"` // Timeout for VNC ready check (default: 2s)
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DockerNetwork:       "bridge",
		ContainerNoVNCPort:  6080,
		ContainerVNCPort:    5900,
		ContainerNamePrefix: "yao-sandbox-",
		IPCacheTTL:          30 * time.Second,
		VNCCheckTimeout:     2 * time.Second,
	}
}

// Init initializes config from environment variables
func (c *Config) Init() {
	if env := os.Getenv("YAO_VNC_DOCKER_NETWORK"); env != "" {
		c.DockerNetwork = env
	} else if c.DockerNetwork == "" {
		c.DockerNetwork = "bridge"
	}

	if env := os.Getenv("YAO_VNC_CONTAINER_NOVNC_PORT"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			c.ContainerNoVNCPort = v
		}
	} else if c.ContainerNoVNCPort == 0 {
		c.ContainerNoVNCPort = 6080
	}

	if env := os.Getenv("YAO_VNC_CONTAINER_VNC_PORT"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			c.ContainerVNCPort = v
		}
	} else if c.ContainerVNCPort == 0 {
		c.ContainerVNCPort = 5900
	}

	if env := os.Getenv("YAO_VNC_CONTAINER_NAME_PREFIX"); env != "" {
		c.ContainerNamePrefix = env
	} else if c.ContainerNamePrefix == "" {
		c.ContainerNamePrefix = "yao-sandbox-"
	}

	if env := os.Getenv("YAO_VNC_IP_CACHE_TTL"); env != "" {
		if v, err := time.ParseDuration(env); err == nil && v > 0 {
			c.IPCacheTTL = v
		}
	} else if c.IPCacheTTL == 0 {
		c.IPCacheTTL = 30 * time.Second
	}

	if env := os.Getenv("YAO_VNC_CHECK_TIMEOUT"); env != "" {
		if v, err := time.ParseDuration(env); err == nil && v > 0 {
			c.VNCCheckTimeout = v
		}
	} else if c.VNCCheckTimeout == 0 {
		c.VNCCheckTimeout = 2 * time.Second
	}
}
