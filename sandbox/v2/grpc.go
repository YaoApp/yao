package sandbox

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
func BuildGRPCEnv(pool *Pool, sandboxID, access, refresh string, grpcPort int) map[string]string {
	portStr := strconv.Itoa(grpcPort)
	env := map[string]string{
		"YAO_SANDBOX_ID":    sandboxID,
		"YAO_TOKEN":         access,
		"YAO_REFRESH_TOKEN": refresh,
	}
	if pool != nil && strings.Contains(pool.Addr, "tai://") {
		taiHost := strings.TrimPrefix(pool.Addr, "tai://")
		env["YAO_GRPC_TAI"] = "enable"
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("%s:9100", taiHost)
		env["YAO_GRPC_UPSTREAM"] = fmt.Sprintf("127.0.0.1:%s", portStr)
	} else {
		env["YAO_GRPC_ADDR"] = fmt.Sprintf("127.0.0.1:%s", portStr)
	}
	return env
}
