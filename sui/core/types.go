package core

// SUI is the interface for the SUI
type SUI interface {
	GetTemplates() ([]ITemplate, error)
	GetTemplate(name string) (ITemplate, error)
	UploadTemplate(src string, dst string) (ITemplate, error)
}

// DSL the struct for the DSL
type DSL struct {
	ID      string   `json:"-"`
	Name    string   `json:"name,omitempty"`
	Storage *Storage `json:"storage,omitempty"`
	Public  *Public  `json:"public,omitempty"`
}

// Page is the struct for the page
type Page struct {
	Route string      `json:"route"`
	Name  string      `json:"name,omitempty"`
	Root  string      `json:"-"`
	Codes SourceCodes `json:"-"`
}

// Component is the struct for the component
type Component struct {
	templage *Template
	name     string
}

// Block is the struct for the block
type Block struct {
	ID       string      `json:"id"`
	Name     string      `json:"name,omitempty"`
	Compiled string      `json:"-"`
	Codes    SourceCodes `json:"-"`
}

// Template is the struct for the template
type Template struct {
	Version     int      `json:"version"` // Yao Builder version
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Descrption  string   `json:"description"`
	Screenshots []string `json:"screenshots"`
}

// SourceCodes is the struct for the page codes
type SourceCodes struct {
	HTML Source `json:"-"`
	CSS  Source `json:"-"`
	JS   Source `json:"-"`
	TS   Source `json:"-"`
	LESS Source `json:"-"`
	DATA Source `json:"-"`
}

// Source is the struct for the source
type Source struct {
	File string `json:"-"`
	Code string `json:"-"`
}

// Public is the struct for the static
type Public struct {
	Host  string `json:"host,omitempty"`
	Root  string `json:"root,omitempty"`
	Index string `json:"index,omitempty"`
}

// Storage is the struct for the storage
type Storage struct {
	Driver string                 `json:"driver"`
	Option map[string]interface{} `json:"option,omitempty"`
}
