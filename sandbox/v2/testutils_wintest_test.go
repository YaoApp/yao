//go:build wintest

package sandbox_test

import "os"

func init() {
	extraHostExecProviders = append(extraHostExecProviders, winHostExec)
}

func winHostExec() []hostExecTarget {
	var targets []hostExecTarget
	if addr := os.Getenv("TAI_TEST_WIN_HOSTEXEC_LINUX"); addr != "" {
		targets = append(targets, hostExecTarget{Name: "win-linux", Addr: addr})
	}
	if addr := os.Getenv("TAI_TEST_WIN_HOSTEXEC_NATIVE"); addr != "" {
		targets = append(targets, hostExecTarget{Name: "win-native", Addr: addr, IsWinNative: true})
	}
	return targets
}
