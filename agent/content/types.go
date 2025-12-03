package content

import "github.com/yaoapp/yao/agent/context"

// FileType represents the type of file content
type FileType string

const (
	// Image types
	FileTypeImage FileType = "image"

	// Audio types
	FileTypeAudio FileType = "audio"

	// Document types
	FileTypeText  FileType = "text"
	FileTypePDF   FileType = "pdf"
	FileTypeWord  FileType = "word"
	FileTypeExcel FileType = "excel"
	FileTypePPT   FileType = "ppt"
	FileTypeCSV   FileType = "csv"

	// Data types
	FileTypeJSON FileType = "json"
	FileTypeXML  FileType = "xml"

	// Binary
	FileTypeBinary FileType = "binary"

	// Other
	FileTypeUnknown FileType = "unknown"
)

// Source represents where the content comes from
type Source string

const (
	SourceHTTP     Source = "http"     // HTTP(S) URL
	SourceUploader Source = "uploader" // Uploader wrapper: __uploader://fileid
	SourceBase64   Source = "base64"   // Base64 encoded data
	SourceLocal    Source = "local"    // Local file path
)

// Result represents the result of content handling
type Result struct {
	Text        string                 // Extracted text content
	ContentPart *context.ContentPart   // Processed ContentPart (for model input)
	Metadata    map[string]interface{} // Additional metadata
	Error       error                  // Error if handling failed
}

// Info holds information about a content part to be handled
type Info struct {
	Source      Source   // Where the content comes from
	FileType    FileType // Type of the file
	ContentType string   // MIME content type
	URL         string   // Original URL or file ID
	Data        []byte   // File data (if already fetched)

	// For uploader wrapper
	UploaderName string // Uploader name from wrapper
	FileID       string // File ID from wrapper
}

// DetectFileType detects file type from content type, filename, and file extension
func DetectFileType(contentType, filename string) FileType {
	// Check by content type first
	switch {
	case contentType == "application/pdf":
		return FileTypePDF
	case contentType == "application/json":
		return FileTypeJSON
	case contentType == "application/xml" || contentType == "text/xml":
		return FileTypeXML
	case contentType == "text/csv":
		return FileTypeCSV
	case contentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		contentType == "application/msword":
		return FileTypeWord
	case contentType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		contentType == "application/vnd.ms-excel":
		return FileTypeExcel
	case contentType == "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		contentType == "application/vnd.ms-powerpoint":
		return FileTypePPT
	}

	// Check image types
	if isImageContentType(contentType) {
		return FileTypeImage
	}

	// Check audio types
	if isAudioContentType(contentType) {
		return FileTypeAudio
	}

	// Check text types
	if isTextContentType(contentType) {
		return FileTypeText
	}

	// If content type doesn't help, check file extension
	if filename != "" {
		if ext := getFileExtension(filename); ext != "" {
			return detectTypeByExtension(ext)
		}
	}

	return FileTypeUnknown
}

// isImageContentType checks if content type is an image
func isImageContentType(contentType string) bool {
	return contentType != "" &&
		(contentType == "image/png" ||
			contentType == "image/jpeg" ||
			contentType == "image/jpg" ||
			contentType == "image/gif" ||
			contentType == "image/webp" ||
			contentType == "image/svg+xml" ||
			contentType == "image/bmp")
}

// isAudioContentType checks if content type is audio
func isAudioContentType(contentType string) bool {
	return contentType != "" &&
		(contentType == "audio/mpeg" ||
			contentType == "audio/mp3" ||
			contentType == "audio/wav" ||
			contentType == "audio/ogg" ||
			contentType == "audio/flac" ||
			contentType == "audio/aac")
}

// isTextContentType checks if content type is text-based
func isTextContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	// Common text MIME types
	textTypes := []string{
		"text/plain",
		"text/html",
		"text/css",
		"text/javascript",
		"text/markdown",
		"text/x-markdown",
		"application/javascript",
		"application/typescript",
		"application/x-yaml",
		"application/yaml",
		"application/toml",
		"application/x-sh",
		"application/x-python",
		"application/x-ruby",
		"application/x-perl",
		"application/x-php",
		"application/x-go",
	}

	for _, t := range textTypes {
		if contentType == t {
			return true
		}
	}

	return false
}

// getFileExtension extracts file extension from filename (without dot)
func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i+1:]
		}
		if filename[i] == '/' || filename[i] == '\\' {
			break
		}
	}
	return ""
}

// detectTypeByExtension detects file type by file extension
func detectTypeByExtension(ext string) FileType {
	// Normalize to lowercase
	ext = toLower(ext)

	// Image extensions
	imageExts := []string{"png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "ico"}
	for _, e := range imageExts {
		if ext == e {
			return FileTypeImage
		}
	}

	// Audio extensions
	audioExts := []string{"mp3", "wav", "ogg", "flac", "aac", "m4a"}
	for _, e := range audioExts {
		if ext == e {
			return FileTypeAudio
		}
	}

	// Document extensions
	switch ext {
	case "pdf":
		return FileTypePDF
	case "doc", "docx":
		return FileTypeWord
	case "xls", "xlsx":
		return FileTypeExcel
	case "ppt", "pptx":
		return FileTypePPT
	case "csv":
		return FileTypeCSV
	case "json":
		return FileTypeJSON
	case "xml":
		return FileTypeXML
	}

	// Code and text file extensions (very comprehensive list)
	textExts := []string{
		"txt", "text", "md", "markdown", "rst",
		// Programming languages
		"go", "py", "js", "ts", "jsx", "tsx", "java", "c", "cpp", "h", "hpp",
		"cs", "rb", "php", "pl", "swift", "kt", "rs", "scala", "clj",
		// Web
		"html", "htm", "css", "scss", "sass", "less",
		// Config
		"yaml", "yml", "toml", "ini", "conf", "config",
		// Shell
		"sh", "bash", "zsh", "fish",
		// Data
		"sql", "graphql", "proto",
		// Others
		"log", "gitignore", "env", "dockerfile",
	}
	for _, e := range textExts {
		if ext == e {
			return FileTypeText
		}
	}

	return FileTypeUnknown
}

// toLower converts ASCII string to lowercase (simple version)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
