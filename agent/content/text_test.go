package content

import (
	"testing"
)

func TestTextHandler_CanHandle(t *testing.T) {
	handler := &TextHandler{}

	tests := []struct {
		name        string
		contentType string
		fileType    FileType
		want        bool
	}{
		{"Plain text", "text/plain", FileTypeText, true},
		{"Markdown", "text/markdown", FileTypeText, true},
		{"HTML", "text/html", FileTypeText, true},
		{"JSON", "application/json", FileTypeJSON, true},
		{"JavaScript", "application/javascript", FileTypeText, true},
		{"TypeScript", "application/typescript", FileTypeText, true},
		{"YAML", "application/yaml", FileTypeText, true},
		{"CSV", "text/csv", FileTypeCSV, true},
		{"XML", "application/xml", FileTypeText, true},
		{"PDF (should not handle)", "application/pdf", FileTypePDF, false},
		{"Image (should not handle)", "image/png", FileTypeImage, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.CanHandle(tt.contentType, tt.fileType)
			if got != tt.want {
				t.Errorf("CanHandle(%q, %q) = %v, want %v", tt.contentType, tt.fileType, got, tt.want)
			}
		})
	}
}

func TestTextHandler_Handle(t *testing.T) {
	handler := &TextHandler{}

	tests := []struct {
		name        string
		info        *Info
		wantErr     bool
		checkResult func(*testing.T, *Result)
	}{
		{
			name: "Plain text",
			info: &Info{
				FileType:    FileTypeText,
				ContentType: "text/plain",
				Data:        []byte("Hello, World!"),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *Result) {
				if r.Text != "Hello, World!" {
					t.Errorf("Expected 'Hello, World!', got %q", r.Text)
				}
			},
		},
		{
			name: "JSON with pretty print",
			info: &Info{
				FileType:    FileTypeJSON,
				ContentType: "application/json",
				Data:        []byte(`{"name":"test","value":123}`),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *Result) {
				// Should be pretty printed
				if len(r.Text) <= len(`{"name":"test","value":123}`) {
					t.Errorf("JSON should be pretty printed, got: %q", r.Text)
				}
			},
		},
		{
			name: "Code file (Go)",
			info: &Info{
				FileType:    FileTypeText,
				ContentType: "text/plain",
				Data:        []byte("package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}"),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *Result) {
				if r.Text == "" {
					t.Error("Expected non-empty text for Go code")
				}
			},
		},
		{
			name: "Empty data",
			info: &Info{
				FileType:    FileTypeText,
				ContentType: "text/plain",
				Data:        []byte{},
			},
			wantErr: true,
		},
	}

	// Create test context
	testCtx := newTestContext(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.Handle(testCtx, tt.info, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestDetectFileType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		filename    string
		want        FileType
	}{
		{"Go file", "text/plain", "main.go", FileTypeText},
		{"Python file", "text/plain", "script.py", FileTypeText},
		{"JavaScript file", "application/javascript", "app.js", FileTypeText},
		{"TypeScript file", "text/plain", "index.ts", FileTypeText},
		{"Markdown file", "text/markdown", "README.md", FileTypeText},
		{"JSON file", "application/json", "config.json", FileTypeJSON},
		{"YAML file", "text/plain", "config.yml", FileTypeText},
		{"PDF file", "application/pdf", "document.pdf", FileTypePDF},
		{"Image file", "image/png", "photo.png", FileTypeImage},
		{"CSV file", "text/csv", "data.csv", FileTypeCSV},
		{"XML file", "application/xml", "config.xml", FileTypeXML},
		{"Shell script", "text/plain", "script.sh", FileTypeText},
		{"Dockerfile", "text/plain", "Dockerfile", FileTypeText},
		{"gitignore", "text/plain", ".gitignore", FileTypeText},
		{"HTML", "text/html", "index.html", FileTypeText},
		{"CSS", "text/css", "styles.css", FileTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFileType(tt.contentType, tt.filename)
			if got != tt.want {
				t.Errorf("DetectFileType(%q, %q) = %v, want %v", tt.contentType, tt.filename, got, tt.want)
			}
		})
	}
}
