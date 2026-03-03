package common

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// HashFile computes the SHA-256 hash of a file and returns it as "sha256-<hex>".
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256-%x", h.Sum(nil)), nil
}

// HashBytes computes the SHA-256 hash of raw bytes and returns "sha256-<hex>".
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256-%x", h[:])
}

// HashDir walks a directory and returns a map of relative paths to their SHA-256 hashes.
// The relPrefix is prepended to each relative path (use "" for no prefix).
func HashDir(dir, relPrefix string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if relPrefix != "" {
			rel = strings.TrimRight(filepath.ToSlash(relPrefix), "/") + "/" + rel
		}
		hash, err := HashFile(path)
		if err != nil {
			return err
		}
		result[rel] = hash
		return nil
	})
	return result, err
}
