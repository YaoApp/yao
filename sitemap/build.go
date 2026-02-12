package sitemap

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// BuildOpen creates a new sitemap writer. Returns a UUID handle string.
// The caller uses this handle for subsequent Write and Close operations.
func BuildOpen(opts *BuildOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("build options are required")
	}
	if opts.Dir == "" {
		return "", fmt.Errorf("output directory (dir) is required")
	}

	// Ensure the output directory exists
	absDir, err := filepath.Abs(opts.Dir)
	if err != nil {
		return "", fmt.Errorf("invalid dir path: %s", err.Error())
	}
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %s", err.Error())
	}

	id := uuid.NewString()
	writer := &sitemapWriter{
		id:        id,
		dir:       absDir,
		baseURL:   opts.BaseURL,
		count:     0,
		total:     0,
		fileIndex: 0,
		create:    time.Now().Unix(),
	}

	openWriters.Store(id, writer)
	return id, nil
}

// BuildWrite writes a batch of URLs to the sitemap.
// Automatically splits into new files when MaxURLsPerFile (50,000) is reached.
func BuildWrite(handle string, urls []URL) error {
	v, ok := openWriters.Load(handle)
	if !ok {
		return fmt.Errorf("sitemap writer %s not found", handle)
	}
	w := v.(*sitemapWriter)

	for _, u := range urls {
		// Check if we need a new file
		if w.currentFile == nil || w.count >= MaxURLsPerFile {
			if err := w.rotateFile(); err != nil {
				return err
			}
		}

		// Encode the <url> element
		if err := w.encoder.Encode(u); err != nil {
			return fmt.Errorf("failed to encode URL: %s", err.Error())
		}
		w.count++
		w.total++
	}

	return nil
}

// BuildClose finalizes the sitemap output. Closes the current file,
// generates a sitemap index if more than one file was created, and removes
// the handle from openWriters.
func BuildClose(handle string) (*BuildResult, error) {
	v, ok := openWriters.Load(handle)
	if !ok {
		return nil, fmt.Errorf("sitemap writer %s not found", handle)
	}
	w := v.(*sitemapWriter)

	// Close the current file if open
	if err := w.closeCurrentFile(); err != nil {
		return nil, err
	}

	result := &BuildResult{
		Files: w.files,
		Total: w.total,
	}

	// Generate sitemap index if more than one file
	if len(w.files) > 1 {
		indexPath, err := w.generateIndex()
		if err != nil {
			return nil, err
		}
		result.Index = indexPath
	}

	// Clean up
	openWriters.Delete(handle)
	return result, nil
}

// ==================== Internal Methods ====================

// rotateFile closes the current file (if open) and opens a new one.
func (w *sitemapWriter) rotateFile() error {
	// Close existing file first
	if w.currentFile != nil {
		if err := w.closeCurrentFile(); err != nil {
			return err
		}
	}

	w.fileIndex++
	w.count = 0

	filename := fmt.Sprintf("sitemap_%d.xml", w.fileIndex)
	filePath := filepath.Join(w.dir, filename)

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create sitemap file %s: %s", filePath, err.Error())
	}
	w.currentFile = f
	w.files = append(w.files, filePath)

	// Write XML declaration
	if _, err := f.WriteString(xml.Header); err != nil {
		return fmt.Errorf("failed to write XML header: %s", err.Error())
	}

	// Write opening <urlset> tag with all namespaces using EncodeToken
	w.encoder = xml.NewEncoder(f)
	w.encoder.Indent("", "  ")

	start := xml.StartElement{
		Name: xml.Name{Space: "", Local: "urlset"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: NSSitemap},
			{Name: xml.Name{Local: "xmlns:image"}, Value: NSImage},
			{Name: xml.Name{Local: "xmlns:video"}, Value: NSVideo},
			{Name: xml.Name{Local: "xmlns:news"}, Value: NSNews},
		},
	}
	if err := w.encoder.EncodeToken(start); err != nil {
		return fmt.Errorf("failed to write urlset start tag: %s", err.Error())
	}
	if err := w.encoder.Flush(); err != nil {
		return fmt.Errorf("failed to flush encoder: %s", err.Error())
	}

	return nil
}

// closeCurrentFile writes the closing </urlset> tag and closes the file.
func (w *sitemapWriter) closeCurrentFile() error {
	if w.currentFile == nil {
		return nil
	}

	// Write closing </urlset> tag
	end := xml.EndElement{Name: xml.Name{Space: "", Local: "urlset"}}
	if err := w.encoder.EncodeToken(end); err != nil {
		return fmt.Errorf("failed to write urlset end tag: %s", err.Error())
	}
	if err := w.encoder.Flush(); err != nil {
		return fmt.Errorf("failed to flush encoder: %s", err.Error())
	}

	// Write a trailing newline for readability
	w.currentFile.WriteString("\n")

	if err := w.currentFile.Close(); err != nil {
		return fmt.Errorf("failed to close sitemap file: %s", err.Error())
	}
	w.currentFile = nil
	w.encoder = nil
	return nil
}

// generateIndex creates a sitemap index file referencing all generated sitemap files.
func (w *sitemapWriter) generateIndex() (string, error) {
	indexPath := filepath.Join(w.dir, "sitemap_index.xml")
	f, err := os.Create(indexPath)
	if err != nil {
		return "", fmt.Errorf("failed to create sitemap index: %s", err.Error())
	}
	defer f.Close()

	// XML declaration
	if _, err := f.WriteString(xml.Header); err != nil {
		return "", err
	}

	encoder := xml.NewEncoder(f)
	encoder.Indent("", "  ")

	// <sitemapindex> opening tag
	start := xml.StartElement{
		Name: xml.Name{Local: "sitemapindex"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: NSSitemap},
		},
	}
	if err := encoder.EncodeToken(start); err != nil {
		return "", err
	}

	// Write each <sitemap> entry
	now := time.Now().Format("2006-01-02")
	for _, filePath := range w.files {
		loc := filePath
		if w.baseURL != "" {
			loc = w.baseURL + "/" + filepath.Base(filePath)
		}
		entry := SitemapEntry{
			Loc:     loc,
			LastMod: now,
		}
		if err := encoder.Encode(entry); err != nil {
			return "", err
		}
	}

	// </sitemapindex> closing tag
	end := xml.EndElement{Name: xml.Name{Local: "sitemapindex"}}
	if err := encoder.EncodeToken(end); err != nil {
		return "", err
	}
	if err := encoder.Flush(); err != nil {
		return "", err
	}

	f.WriteString("\n")
	return indexPath, nil
}
