package tai

import (
	"io"

	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/types"
	"github.com/yaoapp/yao/tai/volume"
)

// Type aliases kept at package level for convenience.
type Runtime = types.Runtime
type Ports = types.Ports

// Option configures RegisterLocal.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// WithDataDir sets the workspace root directory for Local mode.
func WithDataDir(dir string) Option {
	return optionFunc(func(c *config) { c.dataDir = dir })
}

// WithVolume injects a custom Volume implementation (useful for testing).
func WithVolume(vol volume.Volume) Option {
	return optionFunc(func(c *config) { c.volume = vol })
}

type config struct {
	dataDir string
	volume  volume.Volume
}

func defaultPorts() Ports {
	return Ports{
		GRPC: 19100,
		HTTP: 8099,
		VNC:  16080,
	}
}

func mergedPorts(p Ports) Ports {
	d := defaultPorts()
	if p.GRPC != 0 {
		d.GRPC = p.GRPC
	}
	if p.HTTP != 0 {
		d.HTTP = p.HTTP
	}
	if p.VNC != 0 {
		d.VNC = p.VNC
	}
	if p.Docker != 0 {
		d.Docker = p.Docker
	}
	if p.K8s != 0 {
		d.K8s = p.K8s
	}
	return d
}

func intOr(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

// RegisterLocal probes the local environment and registers the current host
// as the "local" node. Capabilities are set based on actual availability:
// Docker is probed, HostExec is controlled by YAO_HOST_EXEC env var.
// Always returns true — the local node is always registered (at minimum
// with Volume capability).
func RegisterLocal(opts ...Option) bool {
	reg := registry.Global()
	if reg == nil {
		return false
	}
	if _, ok := reg.Get("local"); ok {
		return true
	}

	cfg := &config{}
	for _, o := range opts {
		o.apply(cfg)
	}

	res, err := DialLocal("", cfg.dataDir, cfg.volume)
	if err != nil {
		return false
	}

	reg.Register(&registry.TaiNode{
		TaiID:  "local",
		Mode:   "local",
		System: res.System,
		Capabilities: types.Capabilities{
			Docker:   res.Runtime != nil,
			HostExec: res.HostExec != nil,
		},
	})
	reg.SetResources("local", res)
	return true
}

// InitLocal initializes the Tai registry and registers the local host as a
// node in a single call. This is the preferred entry point for application
// startup.
//
// Capabilities are determined by probing the environment:
//   - Docker reachable  → Docker capability
//   - YAO_HOST_EXEC=true → HostExec capability (with Policy from env)
//   - Volume is always available
func InitLocal(w io.Writer, logMode string, dataDir string) types.Capabilities {
	registry.InitWithWriter(w, logMode)
	RegisterLocal(WithDataDir(dataDir))
	if meta, ok := registry.Global().Get("local"); ok {
		return meta.Capabilities
	}
	return types.Capabilities{}
}

// GetResources returns the ConnResources for a registered Tai node.
func GetResources(taiID string) (*ConnResources, bool) {
	reg := registry.Global()
	if reg == nil {
		return nil, false
	}
	raw, ok := reg.GetResources(taiID)
	if !ok {
		return nil, false
	}
	res, ok := raw.(*ConnResources)
	return res, ok && res != nil
}

// GetNodeMeta returns the metadata for a registered Tai node by ID.
func GetNodeMeta(taiID string) (*types.NodeMeta, bool) {
	reg := registry.Global()
	if reg == nil {
		return nil, false
	}
	return reg.Get(taiID)
}
