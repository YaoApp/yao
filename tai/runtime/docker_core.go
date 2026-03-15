package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// dockerCore contains Docker SDK operations shared by both Local and Docker (via Tai) runtimes.
type dockerCore struct {
	cli *client.Client
}

func (d *dockerCore) create(ctx context.Context, opts CreateOptions, addVNCPorts bool) (string, error) {
	cfg := &container.Config{
		Image:      opts.Image,
		Cmd:        opts.Cmd,
		Env:        envSlice(opts.Env),
		WorkingDir: opts.WorkingDir,
		Labels:     opts.Labels,
		User:       opts.User,
	}

	hostCfg := &container.HostConfig{
		Binds:      normalizeBinds(opts.Binds),
		ExtraHosts: []string{"host.tai.internal:host-gateway"},
	}

	if opts.Memory > 0 {
		hostCfg.Resources.Memory = opts.Memory
	}
	if opts.CPUs > 0 {
		hostCfg.Resources.NanoCPUs = int64(opts.CPUs * 1e9)
	}

	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, p := range opts.Ports {
		cp := nat.Port(fmt.Sprintf("%d/%s", p.ContainerPort, proto(p.Protocol)))
		exposedPorts[cp] = struct{}{}
		portBindings[cp] = []nat.PortBinding{{
			HostIP:   hostIP(p.HostIP),
			HostPort: portStr(p.HostPort),
		}}
	}

	// tai relay daemon port — always mapped so the host Tai server can
	// forward arbitrary container ports through the relay.
	relayPort := nat.Port("2099/tcp")
	exposedPorts[relayPort] = struct{}{}
	portBindings[relayPort] = []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: ""}}

	if opts.VNC {
		hostCfg.CapAdd = append(hostCfg.CapAdd, "SYS_ADMIN")
		shmSize := opts.Memory / 4
		if shmSize < 256*1024*1024 {
			shmSize = 256 * 1024 * 1024
		}
		hostCfg.ShmSize = shmSize
		cfg.Env = append(cfg.Env, "VNC_ENABLED=true")

		if addVNCPorts {
			for _, p := range []int{6080, 5900} {
				cp := nat.Port(fmt.Sprintf("%d/tcp", p))
				exposedPorts[cp] = struct{}{}
				portBindings[cp] = []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: ""}}
			}
		}
	}

	if len(exposedPorts) > 0 {
		cfg.ExposedPorts = exposedPorts
		hostCfg.PortBindings = portBindings
	}

	resp, err := d.cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, opts.Name)
	if err != nil {
		return "", fmt.Errorf("create: %w", err)
	}
	return resp.ID, nil
}

func (d *dockerCore) start(ctx context.Context, id string) error {
	return d.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (d *dockerCore) stop(ctx context.Context, id string, timeoutSec int) error {
	return d.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeoutSec})
}

func (d *dockerCore) remove(ctx context.Context, id string, force bool) error {
	return d.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force, RemoveVolumes: true})
}

func (d *dockerCore) exec(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
	execCfg := container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   opts.WorkDir,
		Env:          envSlice(opts.Env),
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := d.cli.ContainerExecCreate(ctx, id, execCfg)
	if err != nil {
		return nil, fmt.Errorf("exec create: %w", err)
	}

	resp, err := d.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("exec attach: %w", err)
	}
	defer resp.Close()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, resp.Reader); err != nil && err != io.EOF {
		return nil, fmt.Errorf("exec read: %w", err)
	}

	inspect, err := d.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("exec inspect: %w", err)
	}

	return &ExecResult{
		ExitCode: inspect.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

func (d *dockerCore) execStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*StreamHandle, error) {
	execCfg := container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   opts.WorkDir,
		Env:          envSlice(opts.Env),
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := d.cli.ContainerExecCreate(ctx, id, execCfg)
	if err != nil {
		return nil, fmt.Errorf("exec create: %w", err)
	}

	resp, err := d.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("exec attach: %w", err)
	}

	execCtx, execCancel := context.WithCancel(ctx)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	// Pump user writes into the multiplexed connection.
	// Closing stdinW sends EOF to the container stdin without
	// tearing down the underlying connection (which carries stdout/stderr).
	go func() {
		io.Copy(resp.Conn, stdinR)
		resp.CloseWrite()
	}()

	go func() {
		_, _ = stdcopy.StdCopy(stdoutW, stderrW, resp.Reader)
		stdoutW.Close()
		stderrW.Close()
	}()

	return &StreamHandle{
		Stdin:  stdinW,
		Stdout: stdoutR,
		Stderr: stderrR,
		Wait: func() (int, error) {
			for {
				inspect, err := d.cli.ContainerExecInspect(execCtx, execResp.ID)
				if err != nil {
					return -1, fmt.Errorf("exec inspect: %w", err)
				}
				if !inspect.Running {
					return inspect.ExitCode, nil
				}
			}
		},
		Cancel: func() {
			execCancel()
			resp.Close()
		},
	}, nil
}

func (d *dockerCore) inspect(ctx context.Context, id string) (*ContainerInfo, error) {
	info, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	ci := &ContainerInfo{
		ID:     info.ID,
		Name:   strings.TrimPrefix(info.Name, "/"),
		Image:  info.Config.Image,
		Status: info.State.Status,
		Labels: info.Config.Labels,
	}

	if info.NetworkSettings != nil {
		for _, net := range info.NetworkSettings.Networks {
			if net.IPAddress != "" {
				ci.IP = net.IPAddress
				break
			}
		}
		for portProto, bindings := range info.NetworkSettings.Ports {
			parts := strings.SplitN(string(portProto), "/", 2)
			cp, _ := strconv.Atoi(parts[0])
			protocol := "tcp"
			if len(parts) > 1 {
				protocol = parts[1]
			}
			for _, b := range bindings {
				hp, _ := strconv.Atoi(b.HostPort)
				ci.Ports = append(ci.Ports, PortMapping{
					ContainerPort: cp,
					HostPort:      hp,
					HostIP:        b.HostIP,
					Protocol:      protocol,
				})
			}
		}
	}
	return ci, nil
}

func (d *dockerCore) list(ctx context.Context, opts ListOptions) ([]ContainerInfo, error) {
	listOpts := container.ListOptions{All: opts.All}
	if len(opts.Labels) > 0 {
		f := filters.NewArgs()
		for k, v := range opts.Labels {
			f.Add("label", k+"="+v)
		}
		listOpts.Filters = f
	}

	containers, err := d.cli.ContainerList(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		ci := ContainerInfo{
			ID:     c.ID,
			Name:   name,
			Image:  c.Image,
			Status: c.State,
			Labels: c.Labels,
		}
		for _, p := range c.Ports {
			ci.Ports = append(ci.Ports, PortMapping{
				ContainerPort: int(p.PrivatePort),
				HostPort:      int(p.PublicPort),
				HostIP:        p.IP,
				Protocol:      p.Type,
			})
		}
		result = append(result, ci)
	}
	return result, nil
}

// normalizeBinds converts Windows-style host paths in Docker bind-mount
// specifications to WSL2 mount paths that Docker (running in WSL2) accepts.
// e.g. "D:\volumes\ws-abc:/workspace:rw" -> "/mnt/d/volumes/ws-abc:/workspace:rw"
//
// Detection is based on the path content (drive-letter prefix), not runtime.GOOS,
// because the path may originate from a remote Tai node (Windows) while Yao
// runs on macOS/Linux.
func normalizeBinds(binds []string) []string {
	if len(binds) == 0 {
		return binds
	}
	out := make([]string, len(binds))
	changed := false
	for i, b := range binds {
		out[i] = normalizeWindowsBind(b)
		if out[i] != b {
			changed = true
		}
	}
	if !changed {
		return binds
	}
	return out
}

// normalizeWindowsBind handles a single bind spec "hostPath:containerPath[:mode]".
// When Yao runs on Windows and Docker runs in WSL2, Windows paths like
// "D:\volumes\ws-abc" must be converted to "/mnt/d/volumes/ws-abc" because
// WSL2 mounts Windows drives under /mnt/<lowercase-letter>/.
func normalizeWindowsBind(bind string) string {
	if len(bind) < 3 {
		return bind
	}

	// Detect drive-letter prefix: "X:\" or "X:/"
	if bind[1] != ':' || (bind[2] != '\\' && bind[2] != '/') {
		return bind
	}

	// Find the next colon after the drive letter colon (the bind separator)
	idx := strings.Index(bind[2:], ":")
	if idx < 0 {
		return bind
	}
	hostPath := bind[:2+idx]
	rest := bind[2+idx:] // starts with ":"

	// Convert "D:\foo\bar" -> "/mnt/d/foo/bar"
	drive := strings.ToLower(string(hostPath[0]))
	tail := strings.ReplaceAll(hostPath[2:], `\`, `/`)
	return "/mnt/" + drive + tail + rest
}
