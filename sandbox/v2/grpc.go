package sandbox

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/yaoapp/yao/config"
)

// BuildGRPCEnv builds the gRPC environment variables for a sandbox container
// based on the Tai node's mode and address from the registry.
//
// mode is the TaiNode.Mode ("local", "direct", "tunnel").
// addr is the TaiNode.Addr (e.g. "tai://host:port" for direct mode).
// sandboxID is the container's sandbox identifier.
//
// The Yao gRPC port is read from config.Conf.GRPC.Port.
func BuildGRPCEnv(mode, addr, sandboxID string) map[string]string {
	grpcPort := config.Conf.GRPC.Port
	if grpcPort == 0 {
		grpcPort = 9099
	}
	portStr := strconv.Itoa(grpcPort)

	env := map[string]string{
		"YAO_SANDBOX_ID": sandboxID,
	}

	switch mode {
	case "local":
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("host.docker.internal:%s", portStr)

	case "tunnel":
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("127.0.0.1:%s", portStr)

	case "direct":
		u, err := url.Parse(addr)
		if err != nil || u.Hostname() == "" {
			env["YAO_GRPC_ADDR"] = fmt.Sprintf("host.docker.internal:%s", portStr)
			return env
		}
		taiHost := u.Hostname()
		taiPort := u.Port()
		if taiPort == "" {
			taiPort = "19100"
		}
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("%s:%s", taiHost, taiPort)

	default:
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("host.docker.internal:%s", portStr)
	}
	return env
}
