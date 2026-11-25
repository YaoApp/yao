package types

import "time"

// Hardware is the structured hardware snapshot reported by Tai at registration.
type Hardware struct {
	OS                string             `json:"os,omitempty"`
	Arch              string             `json:"arch,omitempty"`
	CPUCores          uint32             `json:"cpu_cores,omitempty"`
	MemoryMB          uint64             `json:"memory_mb,omitempty"`
	DiskFreeMB        uint64             `json:"disk_free_mb,omitempty"`
	GPUs              []GPU              `json:"gpus,omitempty"`
	Capabilities      []string           `json:"capabilities,omitempty"`
	ContainerRuntimes []ContainerRuntime `json:"container_runtimes,omitempty"`
	TaiVersion        string             `json:"tai_version,omitempty"`
}

// GPU is one GPU reported by Tai.
type GPU struct {
	Name   string `json:"name"`
	VRAMMB uint64 `json:"vram_mb"`
	Driver string `json:"driver,omitempty"`
}

// ContainerRuntime is one container runtime reported by Tai.
type ContainerRuntime struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Socket  string `json:"socket,omitempty"`
}

// Network is one network address reported by Tai. It mirrors ComputerNetwork.
type Network struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Kind         string `json:"kind"`
	Family       string `json:"family"`
	Address      string `json:"address"`
	PrefixLength uint32 `json:"prefix_length,omitempty"`
	Primary      bool   `json:"primary,omitempty"`
}

// ComputerSnapshot is a full hardware/network observation from Tai.
// A non-nil Snapshot on CreateOrUpdate replaces the persisted snapshot so a
// reconnect cannot leave stale hardware values behind.
type ComputerSnapshot struct {
	OS                string
	Arch              string
	CPUCores          uint32
	MemoryMB          uint64
	DiskFreeMB        uint64
	GPUs              []GPU
	Capabilities      []string
	ContainerRuntimes []ContainerRuntime
	TaiVersion        string
	IP                string
	Networks          []Network
	LastSeenAt        time.Time
}
