// Package testdata provides helpers to build .yao.zip test fixtures in memory.
package testdata

import (
	"archive/zip"
	"bytes"
	"encoding/json"
)

// Manifest mirrors the pkg.yao structure for test fixture construction.
type Manifest struct {
	Type         string            `json:"type"`
	Scope        string            `json:"scope"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	Dependencies []ManifestDep     `json:"dependencies,omitempty"`
	Engines      map[string]string `json:"engines,omitempty"`
	Keywords     []string          `json:"keywords,omitempty"`
	License      string            `json:"license,omitempty"`
	Author       *ManifestAuthor   `json:"author,omitempty"`
}

// ManifestDep represents a dependency entry in pkg.yao.
type ManifestDep struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ManifestAuthor holds author information.
type ManifestAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// BuildZip creates an in-memory .yao.zip with a package/pkg.yao manifest
// and an optional set of extra files (path relative to package/) -> content.
func BuildZip(manifest *Manifest, extraFiles map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	f, err := w.Create("package/pkg.yao")
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(data); err != nil {
		return nil, err
	}

	for name, content := range extraFiles {
		f, err := w.Create("package/" + name)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
