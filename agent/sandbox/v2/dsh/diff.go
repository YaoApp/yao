package dsh

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/yaoapp/yao/agent/output/message"
)

// metaDiffs is the parsed form of the "diffs" array inside a tool_result's meta object.
type metaDiffs struct {
	Diffs []metaDiffEntry `json:"diffs"`
}

// metaDiffEntry represents one file change from DSH meta.diffs.
type metaDiffEntry struct {
	Path    string  `json:"path"`
	OldText *string `json:"oldText"` // nil for newly created files
	NewText string  `json:"newText"`
}

// computeFilePatches parses meta JSON, computes unified diffs, and returns FilePatch values.
func computeFilePatches(metaRaw json.RawMessage) []*message.FilePatch {
	var md metaDiffs
	if err := json.Unmarshal(metaRaw, &md); err != nil {
		return nil
	}
	if len(md.Diffs) == 0 {
		return nil
	}

	patches := make([]*message.FilePatch, 0, len(md.Diffs))
	for _, d := range md.Diffs {
		fp := buildFilePatch(d)
		if fp != nil {
			patches = append(patches, fp)
		}
	}
	return patches
}

// buildFilePatch creates a FilePatch from a single meta diff entry.
func buildFilePatch(d metaDiffEntry) *message.FilePatch {
	if d.Path == "" {
		return nil
	}

	status := "modified"
	oldLines := []string{}
	if d.OldText == nil {
		status = "created"
	} else {
		oldLines = splitLines(*d.OldText)
	}
	newLines := splitLines(d.NewText)

	patch, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        oldLines,
		B:        newLines,
		FromFile: "a/" + filepath.ToSlash(d.Path),
		ToFile:   "b/" + filepath.ToSlash(d.Path),
		Context:  3,
	})
	if err != nil || patch == "" {
		return nil
	}

	adds, dels := countDiffLines(patch)
	return &message.FilePatch{
		Path:      d.Path,
		Status:    status,
		Patch:     patch,
		Additions: adds,
		Deletions: dels,
	}
}

// countDiffLines counts added (+) and deleted (-) lines in a unified diff string.
func countDiffLines(patch string) (adds, dels int) {
	for _, line := range strings.Split(patch, "\n") {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '+':
			if !strings.HasPrefix(line, "+++") {
				adds++
			}
		case '-':
			if !strings.HasPrefix(line, "---") {
				dels++
			}
		}
	}
	return
}

// splitLines splits text into lines preserving the format go-difflib expects.
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
