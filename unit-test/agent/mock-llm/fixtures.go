package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	fixtureStore = map[string][]byte{}
	fixtureMu    sync.RWMutex
)

// LoadFixtures walks the fixture directory and loads all .json files into memory.
// Files are keyed by their relative path without extension: "openai/simple-chat" for "openai/simple-chat.json".
func LoadFixtures(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("fixture dir %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read fixture %s: %w", path, err)
		}

		rel, _ := filepath.Rel(dir, path)
		key := strings.TrimSuffix(rel, ".json")
		key = filepath.ToSlash(key)

		fixtureMu.Lock()
		fixtureStore[key] = data
		fixtureMu.Unlock()
		return nil
	})
}

func GetFixture(key string) ([]byte, bool) {
	fixtureMu.RLock()
	defer fixtureMu.RUnlock()
	data, ok := fixtureStore[key]
	return data, ok
}
