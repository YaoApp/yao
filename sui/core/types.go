package core

// DSL the struct for the DSL
type DSL struct {
	ID      string   `json:"-"`
	Name    string   `json:"name,omitempty"`
	Storage *Storage `json:"storage,omitempty"`
	Public  *Public  `json:"public,omitempty"`
}

// Page is the struct for the page
type Page struct {
	Route    string      `json:"route"`
	Name     string      `json:"name,omitempty"`
	Path     string      `json:"-"`
	Codes    SourceCodes `json:"-"`
	Document []byte      `json:"-"`
}

// PageTreeNode is the struct for the page tree node
type PageTreeNode struct {
	Name     string          `json:"name,omitempty"`
	IsDir    bool            `json:"is_dir,omitempty"`
	Children []*PageTreeNode `json:"children,omitempty"`
	IPage    IPage           `json:"page,omitempty"`
	Expand   bool            `json:"expand,omitempty"`
	Active   bool            `json:"active,omitempty"`
}

// Component is the struct for the component
type Component struct {
	ID       string      `json:"id"`
	Name     string      `json:"name,omitempty"`
	Compiled string      `json:"-"`
	Codes    SourceCodes `json:"-"`
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
	Version     int            `json:"version"` // Yao Builder version
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Descrption  string         `json:"description"`
	Screenshots []string       `json:"screenshots"`
	Themes      []SelectOption `json:"themes"`
	Document    []byte         `json:"-"`
}

// Theme is the struct for the theme
type Theme struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// SelectOption is the struct for the select option
type SelectOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Asset is the struct for the asset
type Asset struct {
	file    string
	Type    string `json:"type"`
	Content []byte `json:"content"`
}

// Request is the struct for the request
type Request struct {
	Method  string                 `json:"method"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Query   map[string][]string    `json:"query,omitempty"`
	Params  map[string]string      `json:"params,omitempty"`
	Headers []string               `json:"headers,omitempty"`
	Body    []byte                 `json:"body,omitempty"`
	Theme   string                 `json:"theme,omitempty"`
	Locale  string                 `json:"locale,omitempty"`
}

// RequestSource is the struct for the request
type RequestSource struct {
	UID        string           `json:"uid"`
	User       string           `json:"user,omitempty"`
	Page       *SourceData      `json:"page,omitempty"`
	Style      *SourceData      `json:"style,omitempty"`
	Script     *SourceData      `json:"script,omitempty"`
	Data       *SourceData      `json:"data,omitempty"`
	Board      *BoardSourceData `json:"board,omitempty"`
	NeedToSave struct {
		Page     bool `json:"page,omitempty"`
		Style    bool `json:"style,omitempty"`
		Script   bool `json:"script,omitempty"`
		Data     bool `json:"data,omitempty"`
		Board    bool `json:"board,omitempty"`
		Validate bool `json:"validate,omitempty"`
	} `json:"needToSave,omitempty"`
}

// ResponseEditor is the struct for the response
type ResponseEditor struct {
	HTML     string                 `json:"html,omitempty"`
	CSS      string                 `json:"css,omitempty"`
	Scripts  []string               `json:"scripts,omitempty"`
	Styles   []string               `json:"styles,omitempty"`
	Setting  map[string]interface{} `json:"setting,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
}

// SourceData is the struct for the response
type SourceData struct {
	Source   string `json:"source,omitempty"`
	Language string `json:"language,omitempty"`
}

// BoardSourceData is the struct for the response
type BoardSourceData struct {
	HTML  string `json:"html,omitempty"`
	Style string `json:"style,omitempty"`
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

// DocumentDefault is the default document
var DocumentDefault = []byte(`
<!DOCTYPE html>
<html lang="{{ $REQ.locale || 'en' }}">
  <head>
    <meta charset="UTF-8" />
    <title>{{ $DATA.head.title || '' }}</title>
    <meta
      name="viewport"
      content="width=device-width, initial-scale=1, shrink-to-fit=no"
    />
    <meta
      name="description"
      content="{{ $DATA.head.description || '' }}"
    />
    <meta
      name="keywords"
      content="{{ $DATA.head.keywords || '' }}"
    />
    <meta name="author" content="Yao" />
    <meta name="website" content="https://yaoapps.com" />
    <meta name="email" content="friends@iqka.com" />
    <meta name="version" content="2.0.0" />
    <meta http-equiv="X-UA-Compatible" content="IE=edge" />
  </head>
  <body>
  	{{ __page }}
  </body>
</html>
`)
