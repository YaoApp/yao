//go:build windows

package agent

import "os/exec"

func setProcAttr(cmd *exec.Cmd) {
	// Windows does not support Setpgid or Pdeathsig.
}
