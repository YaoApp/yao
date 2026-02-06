package sandbox

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// mapToSlice converts map to []string for environment variables
func mapToSlice(m map[string]string) []string {
	if m == nil {
		return nil
	}
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, k+"="+v)
	}
	return result
}

// parseMemory converts string like "2g" to bytes
func parseMemory(s string) int64 {
	if s == "" {
		return 0
	}

	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) < 2 {
		v, _ := strconv.ParseInt(s, 10, 64)
		return v
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	switch unit {
	case 'k':
		return int64(num * 1024)
	case 'm':
		return int64(num * 1024 * 1024)
	case 'g':
		return int64(num * 1024 * 1024 * 1024)
	case 't':
		return int64(num * 1024 * 1024 * 1024 * 1024)
	default:
		// Assume bytes if no unit
		v, _ := strconv.ParseInt(s, 10, 64)
		return v
	}
}

// parseLS parses ls -la output to []FileInfo
// If hasTimeStyle is true, expects GNU ls output with --time-style=+%s (Unix epoch)
// If hasTimeStyle is false, expects BusyBox/basic ls output (date string format)
func parseLS(output string, hasTimeStyle bool) []FileInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var result []FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}

		// Parse ls -la output
		// GNU with --time-style: drwxr-xr-x 2 user group 4096 1234567890 filename
		// BusyBox/basic:         drwxr-xr-x 2 user group 4096 Jan  1 12:00 filename
		fields := strings.Fields(line)

		var minFields int
		if hasTimeStyle {
			minFields = 7 // mode, links, user, group, size, timestamp, name
		} else {
			minFields = 9 // mode, links, user, group, size, month, day, time/year, name
		}

		if len(fields) < minFields {
			continue
		}

		// Parse mode
		modeStr := fields[0]
		if len(modeStr) == 0 {
			continue
		}
		mode := parseLSMode(modeStr)

		// Parse size
		size, _ := strconv.ParseInt(fields[4], 10, 64)

		// Parse timestamp and get filename
		var modTime time.Time
		var name string

		if hasTimeStyle {
			// GNU ls with --time-style=+%s: timestamp is Unix epoch in fields[5]
			timestamp, _ := strconv.ParseInt(fields[5], 10, 64)
			modTime = time.Unix(timestamp, 0)
			name = strings.Join(fields[6:], " ")
		} else {
			// BusyBox/basic ls: date is in fields[5:8] (e.g., "Jan  1 12:00" or "Jan  1  2024")
			// Note: time.Now() is used as fallback since BusyBox date parsing is complex
			modTime = time.Now()
			name = strings.Join(fields[8:], " ")
		}

		// Skip . and ..
		if name == "." || name == ".." {
			continue
		}

		result = append(result, FileInfo{
			Name:    name,
			Size:    size,
			Mode:    mode,
			ModTime: modTime,
			IsDir:   modeStr[0] == 'd',
		})
	}

	return result
}

// parseLSMode parses ls mode string to os.FileMode
func parseLSMode(s string) os.FileMode {
	if len(s) < 10 {
		return 0
	}

	var mode os.FileMode

	// File type
	switch s[0] {
	case 'd':
		mode |= os.ModeDir
	case 'l':
		mode |= os.ModeSymlink
	case 'c':
		mode |= os.ModeCharDevice
	case 'b':
		mode |= os.ModeDevice
	case 'p':
		mode |= os.ModeNamedPipe
	case 's':
		mode |= os.ModeSocket
	}

	// Permissions
	perms := s[1:10]
	permBits := []os.FileMode{
		0400, 0200, 0100, // owner
		0040, 0020, 0010, // group
		0004, 0002, 0001, // other
	}

	for i, b := range perms {
		if b != '-' && i < len(permBits) {
			mode |= permBits[i]
		}
	}

	return mode
}

// parseStat parses stat --format=%n|%s|%f|%Y|%F output to *FileInfo
func parseStat(output string) *FileInfo {
	output = strings.TrimSpace(output)
	parts := strings.Split(output, "|")
	if len(parts) < 5 {
		return nil
	}

	name := parts[0]
	size, _ := strconv.ParseInt(parts[1], 10, 64)
	modeHex, _ := strconv.ParseUint(parts[2], 16, 32)
	timestamp, _ := strconv.ParseInt(parts[3], 10, 64)
	fileType := parts[4]

	return &FileInfo{
		Name:    filepath.Base(name),
		Path:    name,
		Size:    size,
		Mode:    os.FileMode(modeHex),
		ModTime: time.Unix(timestamp, 0),
		IsDir:   strings.Contains(fileType, "directory"),
	}
}

// createTarFromPath creates a tar archive from a host path
func createTarFromPath(hostPath string) (io.ReadCloser, error) {
	// Validate path exists before starting goroutine
	info, err := os.Stat(hostPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)
		var finalErr error

		defer func() {
			tw.Close()
			if finalErr != nil {
				pw.CloseWithError(finalErr)
			} else {
				pw.Close()
			}
		}()

		baseDir := filepath.Dir(hostPath)

		walkFn := func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Get relative path
			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}

			// Create header
			header, err := tar.FileInfoHeader(fi, "")
			if err != nil {
				return err
			}
			header.Name = relPath

			// Handle symlinks
			if fi.Mode()&os.ModeSymlink != 0 {
				link, err := os.Readlink(path)
				if err != nil {
					return err
				}
				header.Linkname = link
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			// Write file content
			if fi.Mode().IsRegular() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				if _, err := io.Copy(tw, f); err != nil {
					return err
				}
			}

			return nil
		}

		if info.IsDir() {
			finalErr = filepath.Walk(hostPath, walkFn)
		} else {
			finalErr = walkFn(hostPath, info, nil)
		}
	}()

	return pr, nil
}

// extractTarToPath extracts a tar archive to a host path
func extractTarToPath(reader io.Reader, hostPath string) error {
	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		target := filepath.Join(hostPath, header.Name)

		// Security check: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(hostPath)) {
			return fmt.Errorf("invalid tar path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			os.Remove(target) // Remove existing symlink if any
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}

	return nil
}

// containerName generates a container name from userID and chatID
func containerName(userID, chatID string) string {
	return fmt.Sprintf("yao-sandbox-%s-%s", userID, chatID)
}
