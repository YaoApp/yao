package core

// SUI is the interface for the SUI
type SUI interface {
	GetTemplates() ([]ITemplate, error)
	GetTemplate(name string) (ITemplate, error)
	UploadTemplate(src string, dst string) (ITemplate, error)
}

// Page is the struct for the page
type Page struct {
	Route string    `json:"route"`
	Root  string    `json:"root"`
	Files PageFiles `json:"files"`
}

// PageFiles is the struct for the page files
type PageFiles struct {
	HTML string `json:"html"`
	CSS  string `json:"css"`
	JS   string `json:"js"`
	TS   string `json:"ts"`
	LESS string `json:"less"`
	DATA string `json:"data"`
}

// Component is the struct for the component
type Component struct {
	templage *Template
	name     string
}

// Block is the struct for the block
type Block struct {
	template *Template
	name     string
}

// Template is the struct for the template
type Template struct {
	Version     int      `json:"version"` // Yao Builder version
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Descrption  string   `json:"description"`
	Screenshots []string `json:"screenshots"`
}

// DSL the struct for the DSL
type DSL struct {
	ID      string   `json:"-"`
	Name    string   `json:"name,omitempty"`
	Storage *Storage `json:"storage,omitempty"`
	Public  *Public  `json:"public,omitempty"`
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
