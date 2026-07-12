package sandbox

import (
	"fmt"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/tai/registry"
)

const taiHost = "host.tai.internal"

// resolveGRPCTLS returns true when the node connects directly to Yao
// (not via Tai Gateway) and Yao's gRPC server has TLS enabled.
// Used by injectGRPCAddr to set YAO_GRPC_TLS=true at exec time.
func (m *Manager) resolveGRPCTLS(nodeID string) bool {
	reg := registry.Global()
	if reg == nil {
		return false
	}
	snap, ok := reg.Get(nodeID)
	if !ok {
		return false
	}
	if snap.Mode == "tunnel" || snap.Mode == "cloud" {
		return false
	}
	return config.Conf.GRPC.Cert != "" && config.Conf.GRPC.Key != ""
}

// resolveGRPCAddr returns the current YAO_GRPC_ADDR for a node by looking up
// its live Ports.GRPC from the registry. This ensures exec-time env always
// reflects the latest Tai gRPC port, even after Tai restarts with a new port.
func (m *Manager) resolveGRPCAddr(nodeID string) string {
	reg := registry.Global()
	if reg == nil {
		return ""
	}
	snap, ok := reg.Get(nodeID)
	if !ok {
		return ""
	}
	switch snap.Mode {
	case "local":
		port := config.Conf.GRPC.Port
		if port == 0 {
			port = 9099
		}
		return fmt.Sprintf("%s:%d", taiHost, port)
	case "tunnel", "cloud":
		port := snap.Ports.GRPC
		if port == 0 {
			return ""
		}
		return fmt.Sprintf("%s:%d", taiHost, port)
	default:
		return ""
	}
}

// BuildGRPCEnv builds the gRPC environment variables for a sandbox container.
//
// All containers reach the host via "host.tai.internal" (injected by Tai at
// container creation). The port depends on the mode:
//
//   - local:                   Yao gRPC port (Tai and Yao on the same machine)
//   - tunnel/cloud:     Tai gRPC port (Tai Gateway forwards to Yao)
//
// taiGRPCPort is the Tai node's gRPC port from registration (Ports.GRPC).
func BuildGRPCEnv(mode string, taiGRPCPort int, sandboxID, chatID, workspaceID string) map[string]string {
	env := map[string]string{
		"YAO_SANDBOX_ID": sandboxID,
	}
	if chatID != "" {
		env["CTX_CHAT_ID"] = chatID
	}
	if workspaceID != "" {
		env["CTX_WORKSPACE_ID"] = workspaceID
	}

	switch mode {
	case "local":
		port := config.Conf.GRPC.Port
		if port == 0 {
			port = 9099
		}
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("%s:%d", taiHost, port)
		if config.Conf.GRPC.Cert != "" && config.Conf.GRPC.Key != "" {
			env["YAO_GRPC_TLS"] = "true"
		}

	case "tunnel", "cloud":
		if taiGRPCPort == 0 {
			return env
		}
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("%s:%d", taiHost, taiGRPCPort)

	default:
		port := config.Conf.GRPC.Port
		if port == 0 {
			port = 9099
		}
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("%s:%d", taiHost, port)
		if config.Conf.GRPC.Cert != "" && config.Conf.GRPC.Key != "" {
			env["YAO_GRPC_TLS"] = "true"
		}
	}
	return env
}
