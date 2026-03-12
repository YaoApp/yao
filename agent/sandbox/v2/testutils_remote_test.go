//go:build remote

package sandboxv2_test

import "os"

func init() {
	extraNodeProviders = append(extraNodeProviders, agentRemoteNodes)
}

func agentRemoteNodes() []nodeConfig {
	addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR")
	if addr == "" {
		return nil
	}
	return []nodeConfig{{Name: "remote", Addr: addr}}
}
