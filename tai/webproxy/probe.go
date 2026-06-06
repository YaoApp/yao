package webproxy

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	probeTimeout = 2 * time.Second
	relayPort    = 2099
)

// probe determines the best connection mode for a given binding target.
// Uses a single Docker client and a single ContainerInspect call.
func probe(opts BindOptions) (ConnMode, string) {
	if opts.TargetID == HostID || opts.ContainerID == "" {
		return ModeDirect, fmt.Sprintf("127.0.0.1:%d", opts.TargetPort)
	}

	// Single Docker inspect for all probes
	info, err := inspectContainer(opts.ContainerID)
	if err != nil {
		return ModeRelay, fmt.Sprintf("127.0.0.1:%d", relayPort)
	}

	// Priority 1: Docker host port mapping
	if addr, ok := checkPortMapping(info, opts.TargetPort); ok {
		return ModeDirect, addr
	}

	// Priority 2: Container IP direct
	if addr, ok := checkContainerIP(info, opts.TargetPort); ok {
		return ModeDirect, addr
	}

	// Priority 3: Relay via 2099
	if addr, ok := checkRelayAccess(info); ok {
		return ModeRelay, addr
	}

	return ModeRelay, fmt.Sprintf("127.0.0.1:%d", relayPort)
}

func inspectContainer(containerID string) (*container.InspectResponse, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	info, err := cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func checkPortMapping(info *container.InspectResponse, targetPort int) (string, bool) {
	if info.NetworkSettings == nil {
		return "", false
	}
	portKey := nat.Port(fmt.Sprintf("%d/tcp", targetPort))
	bindings, ok := info.NetworkSettings.Ports[portKey]
	if !ok || len(bindings) == 0 {
		return "", false
	}

	addr := fmt.Sprintf("127.0.0.1:%s", bindings[0].HostPort)
	if tcpDial(addr) {
		return addr, true
	}
	return "", false
}

func checkContainerIP(info *container.InspectResponse, targetPort int) (string, bool) {
	ip := getContainerIP(info)
	if ip == "" {
		return "", false
	}

	addr := fmt.Sprintf("%s:%d", ip, targetPort)
	if tcpDial(addr) {
		return addr, true
	}
	return "", false
}

func checkRelayAccess(info *container.InspectResponse) (string, bool) {
	if info.NetworkSettings == nil {
		return "", false
	}

	// Check if relay port 2099 is mapped to host
	portKey := nat.Port(fmt.Sprintf("%d/tcp", relayPort))
	bindings, ok := info.NetworkSettings.Ports[portKey]
	if ok && len(bindings) > 0 {
		addr := fmt.Sprintf("127.0.0.1:%s", bindings[0].HostPort)
		if tcpDial(addr) {
			return addr, true
		}
	}

	// Try container IP on relay port
	ip := getContainerIP(info)
	if ip != "" {
		addr := fmt.Sprintf("%s:%d", ip, relayPort)
		if tcpDial(addr) {
			return addr, true
		}
	}

	return "", false
}

func getContainerIP(info *container.InspectResponse) string {
	if info.NetworkSettings == nil {
		return ""
	}
	if info.NetworkSettings.IPAddress != "" {
		return info.NetworkSettings.IPAddress
	}
	for _, net := range info.NetworkSettings.Networks {
		if net.IPAddress != "" {
			return net.IPAddress
		}
	}
	return ""
}

func tcpDial(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, probeTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
