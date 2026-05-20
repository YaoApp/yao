//go:build !windows

package local

import (
	"fmt"
	"os"
	"syscall"
)

func applyOwnership(info os.FileInfo, destPath string) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", destPath)
	}
	return os.Lchown(destPath, int(stat.Uid), int(stat.Gid))
}
