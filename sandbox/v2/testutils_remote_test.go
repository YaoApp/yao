//go:build remote

package sandbox_test

import (
	"os"
	"strings"
)

func init() {
	extraNodeProviders = append(extraNodeProviders, remoteNodes)
	extraHostExecProviders = append(extraHostExecProviders, remoteHostExec)
	extraPurgeProviders = append(extraPurgeProviders, remotePurge)
}

func remoteNodes() []nodeConfig {
	addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR")
	if addr == "" {
		return nil
	}
	return []nodeConfig{{Name: "remote", Addr: addr}}
}

func remoteHostExec() []hostExecTarget {
	addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR")
	if addr == "" {
		return nil
	}
	addr = strings.TrimPrefix(addr, "tai://")
	return []hostExecTarget{{Name: "remote", Addr: addr}}
}

func remotePurge() []purgeTarget {
	addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR")
	if addr == "" {
		return nil
	}
	return []purgeTarget{{name: "remote", addr: addr}}
}
