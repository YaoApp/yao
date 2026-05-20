//go:build windows

package local

import "os"

func applyOwnership(_ os.FileInfo, _ string) error {
	return nil
}
