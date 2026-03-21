package types

import (
	"encoding/json"
	"fmt"
)

const (
	SandboxVersionV1 = "1.0"
	SandboxVersionV2 = "2.0"
)

// SandboxConfig is the V2 sandbox configuration loaded from sandbox.yao or
// the package.yao "sandbox" block when version == "2.0".
type SandboxConfig struct {
	Version     string            `json:"version" yaml:"version"`
	Computer    ComputerConfig    `json:"computer" yaml:"computer"`
	Runner      RunnerConfig      `json:"runner" yaml:"runner"`
	Lifecycle   string            `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`
	IdleTimeout string            `json:"idle_timeout,omitempty" yaml:"idle_timeout,omitempty"`
	MaxLifetime string            `json:"max_lifetime,omitempty" yaml:"max_lifetime,omitempty"`
	StopTimeout string            `json:"stop_timeout,omitempty" yaml:"stop_timeout,omitempty"`
	Prepare     []PrepareStep     `json:"prepare,omitempty" yaml:"prepare,omitempty"`
	Environment map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`
	Secrets     map[string]string `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Filter      *ComputerFilter   `json:"filter,omitempty" yaml:"filter,omitempty"`

	// Populated by the framework at runtime (never serialized).
	Owner       string            `json:"-" yaml:"-"`
	ID          string            `json:"-" yaml:"-"`
	Labels      map[string]string `json:"-" yaml:"-"`
	NodeID      string            `json:"-" yaml:"-"`
	Kind        string            `json:"-" yaml:"-"`
	WorkspaceID string            `json:"-" yaml:"-"`
	DisplayName string            `json:"-" yaml:"-"`
}

// StringOrArray accepts both a single string and an array of strings in JSON/YAML.
//
//	"host"          → ["host"]
//	["host", "box"]  → ["host", "box"]
type StringOrArray []string

func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = arr
		return nil
	}
	return fmt.Errorf("StringOrArray: expected a string or an array of strings")
}

// ComputerFilter defines the query parameters for GET /computer/options.
// Declared in DSL sandbox.filter; frontend passes it through to the API.
type ComputerFilter struct {
	Kind    StringOrArray     `json:"kind,omitempty" yaml:"kind,omitempty"`
	Image   string            `json:"image,omitempty" yaml:"image,omitempty"`
	VNC     *bool             `json:"vnc,omitempty" yaml:"vnc,omitempty"`
	OS      string            `json:"os,omitempty" yaml:"os,omitempty"`
	Arch    string            `json:"arch,omitempty" yaml:"arch,omitempty"`
	MinCPUs float64           `json:"min_cpus,omitempty" yaml:"min_cpus,omitempty"`
	MinMem  string            `json:"min_mem,omitempty" yaml:"min_mem,omitempty"`
	Labels  map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// ComputerConfig describes the execution environment (container or host).
type ComputerConfig struct {
	Image     string    `json:"image,omitempty" yaml:"image,omitempty"`
	VNC       VNCConfig `json:"vnc,omitempty" yaml:"vnc,omitempty"`
	Memory    string    `json:"memory,omitempty" yaml:"memory,omitempty"`
	CPUs      float64   `json:"cpus,omitempty" yaml:"cpus,omitempty"`
	Ports     PortList  `json:"ports,omitempty" yaml:"ports,omitempty"`
	User      string    `json:"user,omitempty" yaml:"user,omitempty"`
	WorkDir   string    `json:"work_dir,omitempty" yaml:"work_dir,omitempty"`
	MountPath string    `json:"mount_path,omitempty" yaml:"mount_path,omitempty"`
	MountMode string    `json:"mount_mode,omitempty" yaml:"mount_mode,omitempty"`
}

// RunnerConfig identifies which Runner to use and how.
type RunnerConfig struct {
	Name    string         `json:"name" yaml:"name"`
	Mode    string         `json:"mode,omitempty" yaml:"mode,omitempty"`
	Options map[string]any `json:"options,omitempty" yaml:"options,omitempty"`
}

// PrepareStep is a single action executed during Runner.Prepare.
type PrepareStep struct {
	Action      string `json:"action" yaml:"action"`
	Once        bool   `json:"once,omitempty" yaml:"once,omitempty"`
	IgnoreError bool   `json:"ignore_error,omitempty" yaml:"ignore_error,omitempty"`

	// action=copy
	Src string `json:"src,omitempty" yaml:"src,omitempty"`
	Dst string `json:"dst,omitempty" yaml:"dst,omitempty"`

	// action=exec
	Cmd        string `json:"cmd,omitempty" yaml:"cmd,omitempty"`
	Background bool   `json:"background,omitempty" yaml:"background,omitempty"`

	// action=file (internal use by Runner.Prepare)
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
	Content []byte `json:"-" yaml:"-"`

	// action=process (reserved)
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Args []any  `json:"args,omitempty" yaml:"args,omitempty"`
}

// ---------------------------------------------------------------------------
// VNCConfig — supports both bool and object in JSON/YAML:
//   true → VNCConfig{Enabled: true}
//   {"enabled": true, "password": "xxx"} → full struct
// ---------------------------------------------------------------------------

type VNCConfig struct {
	Enabled    bool   `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	ViewOnly   bool   `json:"view_only,omitempty" yaml:"view_only,omitempty"`
	Password   string `json:"password,omitempty" yaml:"password,omitempty"`
	Resolution string `json:"resolution,omitempty" yaml:"resolution,omitempty"`
}

func (v *VNCConfig) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		v.Enabled = b
		return nil
	}
	type alias VNCConfig
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*v = VNCConfig(a)
	return nil
}

// ---------------------------------------------------------------------------
// PortList — supports both int array and object array in JSON:
//   [3000, 8080] → []PortMapping{{Port: 3000}, {Port: 8080}}
//   [{"port": 3000, "host_port": 9000}] → full structs
// ---------------------------------------------------------------------------

type PortList []PortMapping

type PortMapping struct {
	Port     int    `json:"port" yaml:"port"`
	HostPort int    `json:"host_port,omitempty" yaml:"host_port,omitempty"`
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

func (p *PortList) UnmarshalJSON(data []byte) error {
	var ints []int
	if err := json.Unmarshal(data, &ints); err == nil {
		out := make(PortList, len(ints))
		for i, port := range ints {
			out[i] = PortMapping{Port: port}
		}
		*p = out
		return nil
	}
	var objs []PortMapping
	if err := json.Unmarshal(data, &objs); err != nil {
		return fmt.Errorf("ports: expected int array or object array: %w", err)
	}
	*p = objs
	return nil
}
