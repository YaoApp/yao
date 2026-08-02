package volume

import (
	"os"
	"path/filepath"
	"runtime"
)

// wsBase returns the absolute path to the workspace config directory
// (.workspace/) for a given session, using localStorage's path resolution.
func (l *localStorage) wsBase(sessionID string) (string, error) {
	return l.abs(sessionID, ".workspace")
}

// ensureConfigDirs creates the workspace config directory structure:
//
//	.workspace/git/           (0700)
//	.workspace/git/config     (empty, 0600 — if not exists)
//	.workspace/ssh/keys/      (0700)
//	.workspace/ssh/config     (empty, 0600 — if not exists)
func ensureConfigDirs(wsBase string) error {
	dirs := []string{
		filepath.Join(wsBase, "git"),
		filepath.Join(wsBase, "ssh", "keys"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}

	emptyFiles := []string{
		filepath.Join(wsBase, "git", "config"),
		filepath.Join(wsBase, "ssh", "config"),
	}
	for _, f := range emptyFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			if err := os.WriteFile(f, []byte{}, 0o600); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeSecureFile atomically writes data to path with 0600 permissions.
// On Windows, os.Rename cannot atomically replace an existing file,
// so we os.Remove first.
func writeSecureFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(path)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(path, 0o600)
	}
	return nil
}
