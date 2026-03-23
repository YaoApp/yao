package sandbox

import (
	"context"
	"fmt"
	"log"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	tairuntime "github.com/yaoapp/yao/tai/runtime"
	taitypes "github.com/yaoapp/yao/tai/types"
	"github.com/yaoapp/yao/workspace"
)

// Manager manages sandbox lifecycle. Node connections are delegated to tai/registry.
// Idle-timeout and lifecycle enforcement is handled by the sandbox watcher (watcher.go).
type Manager struct {
	boxes sync.Map
}

func newManager() *Manager {
	return &Manager{}
}

// Start discovers existing containers from all registered nodes and rebuilds
// the boxes map. The local node must already be registered by tai.InitLocal()
// before Start is called.
func (m *Manager) Start(ctx context.Context) error {
	reg := registry.Global()
	if reg == nil {
		return nil
	}

	for _, snap := range reg.List() {
		res, err := m.getNode(snap.TaiID)
		if err != nil {
			continue
		}
		m.recoverBoxes(ctx, snap.TaiID, res)
	}

	return nil
}

// Nodes returns the list of registered Tai nodes from the registry.
func (m *Manager) Nodes() []taitypes.NodeMeta {
	reg := registry.Global()
	if reg == nil {
		return nil
	}
	return reg.List()
}

// Heartbeat updates the box's last heartbeat timestamp.
func (m *Manager) Heartbeat(sandboxID string, active bool, processCount int) error {
	v, ok := m.boxes.Load(sandboxID)
	if !ok {
		return ErrNotFound
	}
	b := v.(*Box)
	if active {
		b.lastHeartbeat.Store(time.Now().UnixMilli())
	}
	b.processCount.Store(int32(processCount))
	return nil
}

// Host returns a Host handle for executing commands on the Tai host machine.
func (m *Manager) Host(_ context.Context, nodeID string) (*Host, error) {
	if nodeID == "" {
		return nil, ErrNodeMissing
	}

	res, err := m.getNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("sandbox: connect node %q: %w", nodeID, err)
	}

	if res.HostExec == nil {
		return nil, fmt.Errorf("sandbox: node %q has no host_exec capability", nodeID)
	}

	sys := SystemInfo{
		OS:       res.System.OS,
		Arch:     res.System.Arch,
		Hostname: res.System.Hostname,
		NumCPU:   res.System.NumCPU,
		TotalMem: res.System.TotalMem,
		Shell:    res.System.Shell,
		TempDir:  res.System.TempDir,
	}

	return &Host{nodeID: nodeID, system: sys, manager: m}, nil
}

// Create creates and starts a new sandbox.
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Box, error) {
	if opts.Image == "" {
		return nil, fmt.Errorf("sandbox: image is required")
	}

	nodeID := opts.NodeID

	if opts.WorkspaceID != "" {
		if wsm := workspace.M(); wsm != nil {
			node, err := wsm.NodeForWorkspace(ctx, opts.WorkspaceID)
			if err != nil {
				targetNode := nodeID
				if targetNode == "" {
					if nodes := wsm.Nodes(); len(nodes) > 0 {
						for _, n := range nodes {
							if n.Online {
								targetNode = n.Name
								break
							}
						}
					}
				}
				if targetNode == "" {
					return nil, fmt.Errorf("sandbox: resolve workspace %q: no available node", opts.WorkspaceID)
				}
				wsID := opts.WorkspaceID
				if wsID == opts.Owner {
					wsID = workspace.DefaultWorkspaceID(opts.Owner, targetNode)
				}
				_, err = wsm.Create(ctx, workspace.CreateOptions{
					ID:    wsID,
					Name:  defaultWorkspaceName(opts.Locale),
					Owner: opts.Owner,
					Node:  targetNode,
				})
				if err != nil {
					return nil, fmt.Errorf("sandbox: auto-create workspace %q: %w", wsID, err)
				}
				opts.WorkspaceID = wsID
				nodeID = targetNode
			} else {
				nodeID = node
				if nodeID == "local" {
					if ws, e := wsm.Get(ctx, opts.WorkspaceID); e == nil && ws.Owner != "" && ws.Owner != opts.Owner {
						return nil, fmt.Errorf("sandbox: no permission to mount workspace %q", opts.WorkspaceID)
					}
				}
			}
		}
	}

	if nodeID == "" {
		return nil, ErrNodeMissing
	}

	id := opts.ID
	if id == "" {
		id = fmt.Sprintf("sb-%d", time.Now().UnixNano())
	}

	res, err := m.getNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("sandbox: connect node %q: %w", nodeID, err)
	}

	if res.Runtime == nil {
		return nil, fmt.Errorf("sandbox: node %q has no container runtime", nodeID)
	}

	sys := inferSystemInfo(ctx, res, opts.Image)

	taiOpts := m.buildTaiCreateOptions(opts, nodeID, id, sys)

	containerID, err := res.Runtime.Create(ctx, taiOpts)
	if err != nil {
		return nil, fmt.Errorf("sandbox: create container: %w", err)
	}

	if err := res.Runtime.Start(ctx, containerID); err != nil {
		res.Runtime.Remove(ctx, containerID, true)
		return nil, fmt.Errorf("sandbox: start container: %w", err)
	}

	policy := opts.Policy
	if policy == "" {
		policy = Session
	}

	boxWorkDir := opts.WorkDir
	if boxWorkDir == "" {
		boxWorkDir = "/workspace"
	}

	box := &Box{
		id:           id,
		containerID:  containerID,
		nodeID:       nodeID,
		owner:        opts.Owner,
		policy:       policy,
		labels:       opts.Labels,
		idleTimeoutD: opts.IdleTimeout,
		maxLifetimeD: opts.MaxLifetime,
		stopTimeoutD: opts.StopTimeout,
		createdAt:    time.Now(),
		manager:      m,
		vnc:          opts.VNC,
		image:        opts.Image,
		workspaceID:  opts.WorkspaceID,
		workDir:      boxWorkDir,
		displayName:  opts.DisplayName,
		system:       sys,
	}
	box.status.Store("running")
	box.lastCall.Store(time.Now().UnixMilli())

	m.boxes.Store(id, box)
	return box, nil
}

// Get returns an existing sandbox by ID.
func (m *Manager) Get(_ context.Context, id string) (*Box, error) {
	v, ok := m.boxes.Load(id)
	if !ok {
		return nil, ErrNotFound
	}
	return v.(*Box), nil
}

// GetOrCreate returns existing sandbox by ID or creates a new one.
func (m *Manager) GetOrCreate(ctx context.Context, opts CreateOptions) (*Box, error) {
	if opts.ID != "" {
		if v, ok := m.boxes.Load(opts.ID); ok {
			return v.(*Box), nil
		}
	}
	return m.Create(ctx, opts)
}

// List returns all sandboxes, optionally filtered.
func (m *Manager) List(_ context.Context, opts ListOptions) ([]*Box, error) {
	var result []*Box
	m.boxes.Range(func(_, value any) bool {
		b := value.(*Box)
		if opts.Owner != "" && b.owner != opts.Owner {
			return true
		}
		if opts.NodeID != "" && b.nodeID != opts.NodeID {
			return true
		}
		if len(opts.Labels) > 0 {
			for k, v := range opts.Labels {
				if b.labels[k] != v {
					return true
				}
			}
		}
		result = append(result, b)
		return true
	})
	return result, nil
}

// StartBox starts a stopped sandbox and updates its lastCall timestamp.
func (m *Manager) StartBox(ctx context.Context, id string) error {
	v, ok := m.boxes.Load(id)
	if !ok {
		return ErrNotFound
	}
	b := v.(*Box)

	res, err := m.getNode(b.nodeID)
	if err != nil {
		return err
	}
	if res.Runtime == nil {
		return fmt.Errorf("sandbox: node %q has no container runtime", b.nodeID)
	}

	if err := res.Runtime.Start(ctx, b.containerID); err != nil {
		return fmt.Errorf("sandbox: start container %s: %w", b.containerID, err)
	}

	b.status.Store("running")
	b.touch()
	return nil
}

// Remove force-removes a sandbox (SIGKILL + delete).
func (m *Manager) Remove(ctx context.Context, id string) error {
	v, ok := m.boxes.Load(id)
	if !ok {
		return ErrNotFound
	}
	b := v.(*Box)

	res, err := m.getNode(b.nodeID)
	if err == nil && res.Runtime != nil {
		res.Runtime.Remove(ctx, b.containerID, true)
	}

	m.boxes.Delete(id)
	return nil
}

// Close is a no-op; lifecycle management is handled by the sandbox watcher.
func (m *Manager) Close() error {
	return nil
}

func (m *Manager) getNode(name string) (*tai.ConnResources, error) {
	res, ok := tai.GetResources(name)
	if !ok {
		return nil, ErrNodeNotFound
	}
	return res, nil
}

func (m *Manager) buildTaiCreateOptions(opts CreateOptions, nodeID, sandboxID string, sys SystemInfo) tairuntime.CreateOptions {
	env := make(map[string]string)

	reg := registry.Global()
	if reg != nil {
		if snap, ok := reg.Get(nodeID); ok {
			grpcEnv := BuildGRPCEnv(snap.Mode, snap.Ports.GRPC, sandboxID)
			for k, v := range grpcEnv {
				env[k] = v
			}
		}
	}

	for k, v := range opts.Env {
		env[k] = v
	}

	labels := map[string]string{
		"managed-by":      "yao-sandbox",
		"sandbox-id":      sandboxID,
		"sandbox-owner":   opts.Owner,
		"sandbox-node-id": nodeID,
		"sandbox-policy":  string(opts.Policy),
	}
	if opts.VNC {
		labels["sandbox-vnc"] = "true"
	}
	if opts.WorkspaceID != "" {
		labels["workspace-id"] = opts.WorkspaceID
	}
	if opts.DisplayName != "" {
		labels["sandbox-display-name"] = opts.DisplayName
	}
	if sys.OS != "" {
		labels["sandbox-sys-os"] = sys.OS
	}
	if sys.Arch != "" {
		labels["sandbox-sys-arch"] = sys.Arch
	}
	if sys.Hostname != "" {
		labels["sandbox-sys-hostname"] = sys.Hostname
	}
	if sys.NumCPU > 0 {
		labels["sandbox-sys-numcpu"] = strconv.Itoa(sys.NumCPU)
	}
	if sys.TotalMem > 0 {
		labels["sandbox-sys-totalmem"] = strconv.FormatInt(sys.TotalMem, 10)
	}
	if sys.Shell != "" {
		labels["sandbox-sys-shell"] = sys.Shell
	}
	for k, v := range opts.Labels {
		labels[k] = v
	}

	workDir := opts.WorkDir
	if workDir == "" {
		workDir = "/workspace"
	}

	cmd := []string{"sh", "-c", "trap 'exit 0' TERM; while :; do sleep 86400 & wait $!; done"}

	var ports []tairuntime.PortMapping
	for _, p := range opts.Ports {
		ports = append(ports, tairuntime.PortMapping{
			ContainerPort: p.ContainerPort,
			HostPort:      p.HostPort,
			HostIP:        p.HostIP,
			Protocol:      p.Protocol,
		})
	}

	var binds []string
	if opts.WorkspaceID != "" {
		if wsm := workspace.M(); wsm != nil {
			mountPath := opts.MountPath
			if mountPath == "" {
				mountPath = "/workspace"
			}
			mode := opts.MountMode
			if mode == "" {
				mode = "rw"
			}
			hostPath, _ := wsm.MountPath(context.Background(), opts.WorkspaceID)
			if hostPath != "" {
				binds = append(binds, fmt.Sprintf("%s:%s:%s", hostPath, mountPath, mode))
			}
		}
	}

	return tairuntime.CreateOptions{
		Name:       sandboxID,
		Image:      opts.Image,
		Cmd:        cmd,
		Env:        env,
		Binds:      binds,
		WorkingDir: workDir,
		User:       opts.User,
		Memory:     opts.Memory,
		CPUs:       opts.CPUs,
		VNC:        opts.VNC,
		Ports:      ports,
		Labels:     labels,
	}
}

func (m *Manager) recoverBoxes(ctx context.Context, nodeID string, res *tai.ConnResources) {
	if res.Runtime == nil {
		return
	}
	containers, err := res.Runtime.List(ctx, tairuntime.ListOptions{
		All:    true,
		Labels: map[string]string{"managed-by": "yao-sandbox"},
	})
	if err != nil {
		return
	}

	for _, c := range containers {
		sandboxID := c.Labels["sandbox-id"]
		if sandboxID == "" {
			continue
		}
		if _, loaded := m.boxes.Load(sandboxID); loaded {
			continue
		}

		cid := c.ID
		if c.Name != "" {
			cid = c.Name
		}
		hasVNC := c.Labels["sandbox-vnc"] == "true"
		if !hasVNC {
			for _, p := range c.Ports {
				if p.ContainerPort == 5900 || p.ContainerPort == 6080 {
					hasVNC = true
					break
				}
			}
		}
		sys := systemInfoFromLabels(c.Labels)
		if sys.OS == "" {
			sys = inferSystemInfo(ctx, res, c.Image)
		}
		policy := LifecyclePolicy(c.Labels["sandbox-policy"])
		box := &Box{
			id:          sandboxID,
			containerID: cid,
			nodeID:      c.Labels["sandbox-node-id"],
			owner:       c.Labels["sandbox-owner"],
			policy:      policy,
			labels:      c.Labels,
			createdAt:   time.Now(),
			image:       c.Image,
			workspaceID: c.Labels["workspace-id"],
			vnc:         hasVNC,
			workDir:     "/workspace",
			displayName: c.Labels["sandbox-display-name"],
			system:      sys,
			manager:     m,
		}
		switch policy {
		case Session:
			box.idleTimeoutD = DefaultSessionIdleTimeout
		case LongRunning:
			box.idleTimeoutD = DefaultLongRunningIdleTimeout
		}
		box.lastCall.Store(time.Now().UnixMilli())
		m.boxes.Store(sandboxID, box)
	}
}

func defaultWorkspaceName(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		return "默认工作区"
	}
	return "Default Workspace"
}

// inferSystemInfo derives static SystemInfo for a container from image metadata
// and Tai host resources. OS/Arch/Shell come from the image; Hostname/NumCPU/TotalMem
// come from the Tai host.
func inferSystemInfo(ctx context.Context, res *tai.ConnResources, imageRef string) SystemInfo {
	sys := SystemInfo{
		Hostname: res.System.Hostname,
		NumCPU:   res.System.NumCPU,
		TotalMem: res.System.TotalMem,
	}

	if res.Image != nil {
		meta, err := res.Image.Inspect(ctx, imageRef)
		if err != nil {
			log.Printf("[sandbox/v2] image inspect %q: %v (using fallback)", imageRef, err)
		}
		if meta != nil {
			sys.OS = meta.OS
			sys.Arch = meta.Arch
			sys.Shell = meta.Shell
			return sys
		}
	}

	sys.OS = "linux"
	sys.Arch = goruntime.GOARCH
	sys.Shell = "bash"
	return sys
}

// systemInfoFromLabels restores SystemInfo from Docker container labels that
// were persisted at creation time, so recovery doesn't depend on the Tai node
// being connected.
func systemInfoFromLabels(labels map[string]string) SystemInfo {
	numCPU, _ := strconv.Atoi(labels["sandbox-sys-numcpu"])
	totalMem, _ := strconv.ParseInt(labels["sandbox-sys-totalmem"], 10, 64)
	return SystemInfo{
		OS:       labels["sandbox-sys-os"],
		Arch:     labels["sandbox-sys-arch"],
		Hostname: labels["sandbox-sys-hostname"],
		NumCPU:   numCPU,
		TotalMem: totalMem,
		Shell:    labels["sandbox-sys-shell"],
	}
}

// ImageExists reports whether the given image ref exists on the target node.
func (m *Manager) ImageExists(ctx context.Context, nodeID, ref string) (bool, error) {
	res, err := m.getNode(nodeID)
	if err != nil {
		return false, err
	}
	if res.Image == nil {
		return true, nil
	}
	return res.Image.Exists(ctx, ref)
}

// PullImage pulls an image to the target node, returning a channel of
// real-time progress events.
func (m *Manager) PullImage(ctx context.Context, nodeID, ref string, opts ImagePullOptions) (<-chan tairuntime.PullProgress, error) {
	res, err := m.getNode(nodeID)
	if err != nil {
		return nil, err
	}
	if res.Image == nil {
		return nil, nil
	}
	pullOpts := tairuntime.PullOptions{}
	if opts.Auth != nil {
		pullOpts.Auth = &tairuntime.RegistryAuth{
			Username: opts.Auth.Username,
			Password: opts.Auth.Password,
			Server:   opts.Auth.Server,
		}
	}
	return res.Image.Pull(ctx, ref, pullOpts)
}

// EnsureImage checks whether the image exists on the node; if not, it
// pulls the image and blocks until the pull completes.
func (m *Manager) EnsureImage(ctx context.Context, nodeID, ref string, opts ImagePullOptions) error {
	exists, err := m.ImageExists(ctx, nodeID, ref)
	if err != nil {
		return fmt.Errorf("image exists check: %w", err)
	}
	if exists {
		return nil
	}

	ch, err := m.PullImage(ctx, nodeID, ref, opts)
	if err != nil {
		return fmt.Errorf("image pull: %w", err)
	}
	if ch == nil {
		return nil
	}

	for p := range ch {
		if p.Error != "" {
			return fmt.Errorf("image pull %q: %s", ref, p.Error)
		}
	}
	return nil
}
