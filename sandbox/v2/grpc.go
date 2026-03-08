package sandbox

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func createToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateContainerTokens creates an OAuth token pair for a sandbox container.
func CreateContainerTokens(sandboxID, owner string, scopes []string) (access, refresh string, err error) {
	access, err = createToken()
	if err != nil {
		return "", "", err
	}
	refresh, err = createToken()
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// RevokeContainerTokens revokes a refresh token for a sandbox container.
func RevokeContainerTokens(refresh string) error {
	return nil
}

// BuildGRPCEnv builds the gRPC environment variables for a sandbox container.
// Supports tai:// (direct), tunnel:// (NAT traversal), and local modes.
func BuildGRPCEnv(pool *Pool, sandboxID, access, refresh string, grpcPort int) map[string]string {
	portStr := strconv.Itoa(grpcPort)
	env := map[string]string{
		"YAO_SANDBOX_ID":    sandboxID,
		"YAO_TOKEN":         access,
		"YAO_REFRESH_TOKEN": refresh,
	}

	if pool == nil {
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("127.0.0.1:%s", portStr)
		return env
	}

	switch {
	case strings.HasPrefix(pool.Addr, "tunnel://"):
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("127.0.0.1:%d", grpcPort)

	case strings.HasPrefix(pool.Addr, "tai://"):
		u, err := url.Parse(pool.Addr)
		if err != nil {
			env["YAO_GRPC_ADDR"] = fmt.Sprintf("127.0.0.1:%s", portStr)
			return env
		}
		taiHost := u.Hostname()
		taiPort := u.Port()
		if taiPort == "" {
			taiPort = "19100"
		}
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("%s:%s", taiHost, taiPort)

	default:
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("127.0.0.1:%s", portStr)
	}
	return env
}
