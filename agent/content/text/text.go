package text

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/attachment"
)

// SupportedExtensions text file extensions
var SupportedExtensions = map[string]bool{
	// Markdown
	".md":       true,
	".markdown": true,
	// Plain text
	".txt": true,
	// Code files
	".go":     true,
	".ts":     true,
	".tsx":    true,
	".js":     true,
	".jsx":    true,
	".py":     true,
	".java":   true,
	".c":      true,
	".cpp":    true,
	".h":      true,
	".hpp":    true,
	".rs":     true,
	".rb":     true,
	".php":    true,
	".swift":  true,
	".kt":     true,
	".scala":  true,
	".sh":     true,
	".bash":   true,
	".zsh":    true,
	".fish":   true,
	".ps1":    true,
	".bat":    true,
	".cmd":    true,
	".sql":    true,
	".r":      true,
	".lua":    true,
	".perl":   true,
	".pl":     true,
	".groovy": true,
	".dart":   true,
	".elm":    true,
	".ex":     true,
	".exs":    true,
	".erl":    true,
	".hs":     true,
	".clj":    true,
	".lisp":   true,
	".vim":    true,
	// Config files
	".json":  true,
	".jsonc": true,
	".yaml":  true,
	".yml":   true,
	".toml":  true,
	".ini":   true,
	".conf":  true,
	".cfg":   true,
	".env":   true,
	".yao":   true,
	// Web files
	".html": true,
	".htm":  true,
	".css":  true,
	".scss": true,
	".sass": true,
	".less": true,
	".xml":  true,
	".svg":  true,
	// Documentation
	".rst":   true,
	".tex":   true,
	".latex": true,
	".org":   true,
	".adoc":  true,
	// Data files
	".csv": true,
	".tsv": true,
	// Log files
	".log": true,
}

// Text handles text file content
type Text struct {
	options *types.Options
}

// New creates a new text handler
func New(options *types.Options) *Text {
	return &Text{options: options}
}

// IsSupportedExtension checks if a file extension is supported
func IsSupportedExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return SupportedExtensions[ext]
}

// Parse parses text file content and returns text
func (h *Text) Parse(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.File == nil || content.File.URL == "" {
		return content, nil, fmt.Errorf("file content missing URL")
	}

	url := content.File.URL
	filename := content.File.Filename

	// Check cache first
	cachedText, found, err := h.readFromCache(ctx, url)
	if err == nil && found {
		return agentContext.ContentPart{
			Type: agentContext.ContentText,
			Text: cachedText,
		}, nil, nil
	}

	// Read text file
	data, err := h.readFile(ctx, url)
	if err != nil {
		return content, nil, fmt.Errorf("failed to read text file: %w", err)
	}

	// Convert to string
	text := string(data)

	// Add file type context if it's a code file
	ext := strings.ToLower(filepath.Ext(filename))
	if isCodeFile(ext) {
		// Wrap in markdown code block with language hint
		lang := getLanguageFromExt(ext)
		text = fmt.Sprintf("```%s\n%s\n```", lang, text)
	}

	// Cache the result
	if err := h.saveToCache(ctx, url, text); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to cache text: %v\n", err)
	}

	return agentContext.ContentPart{
		Type: agentContext.ContentText,
		Text: text,
	}, nil, nil
}

// ParseRaw parses any file as raw text content without code block wrapping
// This is used as a fallback for unsupported file types
func (h *Text) ParseRaw(ctx *agentContext.Context, content agentContext.ContentPart) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.File == nil || content.File.URL == "" {
		return content, nil, fmt.Errorf("file content missing URL")
	}

	url := content.File.URL
	filename := content.File.Filename

	// Check cache first
	cachedText, found, err := h.readFromCache(ctx, url)
	if err == nil && found {
		return agentContext.ContentPart{
			Type: agentContext.ContentText,
			Text: cachedText,
		}, nil, nil
	}

	// Read file
	data, err := h.readFile(ctx, url)
	if err != nil {
		return content, nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to string directly (no code block wrapping)
	text := string(data)

	// Add filename as context
	if filename != "" {
		text = fmt.Sprintf("File: %s\n\n%s", filename, text)
	}

	// Cache the result
	if err := h.saveToCache(ctx, url, text); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to cache text: %v\n", err)
	}

	return agentContext.ContentPart{
		Type: agentContext.ContentText,
		Text: text,
	}, nil, nil
}

// readFile reads text content from various sources
func (h *Text) readFile(ctx *agentContext.Context, url string) ([]byte, error) {
	if strings.HasPrefix(url, "__") {
		return h.readFromUploader(ctx, url)
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("HTTP URL fetch not implemented yet: %s", url)
	}

	// Try to read as local file path
	if _, err := os.Stat(url); err == nil {
		return os.ReadFile(url)
	}

	return nil, fmt.Errorf("unsupported text file source: %s", url)
}

// readFromUploader reads text content from file uploader
func (h *Text) readFromUploader(ctx *agentContext.Context, wrapper string) ([]byte, error) {
	uploaderName, fileID, ok := attachment.Parse(wrapper)
	if !ok {
		return nil, fmt.Errorf("invalid uploader wrapper format: %s", wrapper)
	}

	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return nil, fmt.Errorf("uploader '%s' not found", uploaderName)
	}

	data, err := manager.Read(ctx.Context, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// readFromCache reads cached text content
func (h *Text) readFromCache(ctx *agentContext.Context, url string) (string, bool, error) {
	uploaderName, fileID, isWrapper := attachment.Parse(url)
	if !isWrapper {
		return "", false, nil
	}

	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return "", false, nil
	}

	text, err := manager.GetText(ctx.Context, fileID, false)
	if err == nil && text != "" {
		return text, true, nil
	}

	return "", false, nil
}

// saveToCache saves processed text to cache
func (h *Text) saveToCache(ctx *agentContext.Context, url string, text string) error {
	uploaderName, fileID, isWrapper := attachment.Parse(url)
	if !isWrapper {
		return nil
	}

	manager, exists := attachment.Managers[uploaderName]
	if !exists {
		return nil
	}

	return manager.SaveText(ctx.Context, fileID, text)
}

// isCodeFile checks if the extension represents a code file
func isCodeFile(ext string) bool {
	codeExts := map[string]bool{
		".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".py": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
		".hpp": true, ".rs": true, ".rb": true, ".php": true, ".swift": true,
		".kt": true, ".scala": true, ".sh": true, ".bash": true, ".zsh": true,
		".sql": true, ".r": true, ".lua": true, ".perl": true, ".pl": true,
		".groovy": true, ".dart": true, ".elm": true, ".ex": true, ".exs": true,
		".erl": true, ".hs": true, ".clj": true, ".lisp": true, ".vim": true,
	}
	return codeExts[ext]
}

// getLanguageFromExt returns the language name for markdown code block
func getLanguageFromExt(ext string) string {
	langMap := map[string]string{
		".go":     "go",
		".ts":     "typescript",
		".tsx":    "tsx",
		".js":     "javascript",
		".jsx":    "jsx",
		".py":     "python",
		".java":   "java",
		".c":      "c",
		".cpp":    "cpp",
		".h":      "c",
		".hpp":    "cpp",
		".rs":     "rust",
		".rb":     "ruby",
		".php":    "php",
		".swift":  "swift",
		".kt":     "kotlin",
		".scala":  "scala",
		".sh":     "bash",
		".bash":   "bash",
		".zsh":    "zsh",
		".fish":   "fish",
		".ps1":    "powershell",
		".bat":    "batch",
		".cmd":    "batch",
		".sql":    "sql",
		".r":      "r",
		".lua":    "lua",
		".perl":   "perl",
		".pl":     "perl",
		".groovy": "groovy",
		".dart":   "dart",
		".elm":    "elm",
		".ex":     "elixir",
		".exs":    "elixir",
		".erl":    "erlang",
		".hs":     "haskell",
		".clj":    "clojure",
		".lisp":   "lisp",
		".vim":    "vim",
		".json":   "json",
		".jsonc":  "jsonc",
		".yaml":   "yaml",
		".yml":    "yaml",
		".toml":   "toml",
		".xml":    "xml",
		".html":   "html",
		".htm":    "html",
		".css":    "css",
		".scss":   "scss",
		".sass":   "sass",
		".less":   "less",
		".svg":    "svg",
		".yao":    "json",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return ""
}
