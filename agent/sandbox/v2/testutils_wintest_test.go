//go:build wintest

package sandboxv2_test

import "os"

func init() {
	extraHostExecProviders = append(extraHostExecProviders, agentWinHostExec)
}

func agentWinHostExec() []hostTarget {
	var targets []hostTarget
	if addr := os.Getenv("TAI_TEST_WIN_HOSTEXEC_LINUX"); addr != "" {
		targets = append(targets, hostTarget{Name: "win-linux", Addr: addr})
	}
	if addr := os.Getenv("TAI_TEST_WIN_HOSTEXEC_NATIVE"); addr != "" {
		targets = append(targets, hostTarget{Name: "win-native", Addr: addr})
	}
	return targets
}
