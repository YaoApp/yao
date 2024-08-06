package core

import (
	"net/url"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// DSL the struct for the DSL
type DSL struct {
	ID         string   `json:"-"`
	Name       string   `json:"name,omitempty"`
	Guard      string   `json:"guard,omitempty"`
	Storage    *Storage `json:"storage,omitempty"`
	Public     *Public  `json:"public,omitempty"`
	CacheStore string   `json:"cache_store,omitempty"` // The cache store
	Sid        string   `json:"-"`
	publicRoot string   `json:"-"`
}

// Setting is the struct for the setting
type Setting struct {
	ID     string                 `json:"id,omitempty"`
	Guard  string                 `json:"guard,omitempty"`
	Option map[string]interface{} `json:"option,omitempty"`
}

// Page is the struct for the page
type Page struct {
	Route      string              `json:"route"`
	Name       string              `json:"name,omitempty"`
	CacheStore string              `json:"-"`
	TemplateID string              `json:"-"`
	SuiID      string              `json:"-"`
	Config     *PageConfig         `json:"-"`
	Path       string              `json:"-"`
	Root       string              `json:"-"`
	Codes      SourceCodes         `json:"-"`
	Script     *Script             `json:"-"` // The backend script  name.backend.ts / name.backend.js
	Document   []byte              `json:"-"`
	GlobalData []byte              `json:"-"`
	Attrs      map[string]string   `json:"-"`
	Attributes []html.Attribute    `json:"-"`
	namespace  string              `json:"-"`
	transCtx   *TranslateContext   `json:"-"`
	parent     *Page               `json:"-"`
	props      map[string]PageProp `json:"-"`
}

// PageProp is the struct for the page prop
type PageProp struct {
	Key   string `json:"key"`
	Val   string `json:"val"`
	Exp   bool   `json:"exp"`
	Trans string `json:"trans"`
}

// BuildContext is the struct for the build context
type BuildContext struct {
	components    map[string]string
	jitComponents map[string]bool
	sequence      int
	doc           *goquery.Document
	scripts       []ScriptNode
	scriptUnique  map[string]bool
	styles        []StyleNode
	styleUnique   map[string]bool
	global        *GlobalBuildContext
	translations  []Translation
	warnings      []string
	visited       map[string]int // Keep a counter for each page
	stack         []string       // Stack to manage build states
}

// PageImport import instance
type PageImport struct {
	is        string
	selection *goquery.Selection
	slots     map[string]*goquery.Selection
}

// TranslateContext is the struct for the translate context
type TranslateContext struct {
	sequence     int
	translations []Translation
}

// ScriptNode is the struct for the script node
type ScriptNode struct {
	Source    string           `json:"source"`
	Attrs     []html.Attribute `json:"attrs"`
	Parent    string           `json:"parent"`
	Namespace string           `json:"namespace"`
	Component string           `json:"component"`
}

// StyleNode is the struct for the style node
type StyleNode struct {
	Source    string           `json:"source"`
	Attrs     []html.Attribute `json:"attrs"`
	Parent    string           `json:"parent"`
	Namespace string           `json:"namespace"`
	Component string           `json:"component"`
}

// GlobalBuildContext is the struct for the global build context
type GlobalBuildContext struct {
	jitComponents map[string]bool
	tmpl          ITemplate
}

// Translation is the struct for the translation
type Translation struct {
	Key     string `json:"key,omitempty"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"` // ENUM: 'text', 'html', 'attr', 'script'
}

// Locale is the struct for the locale
type Locale struct {
	Name           string            `json:"name,omitempty" yaml:"name,omitempty"`
	Formatter      string            `json:"formatter,omitempty" yaml:"formatter,omitempty"`
	Keys           map[string]string `json:"keys,omitempty" yaml:"keys,omitempty"`
	Messages       map[string]string `json:"messages,omitempty" yaml:"messages,omitempty"`
	ScriptMessages map[string]string `json:"script_messages,omitempty" yaml:"script_messages,omitempty"`
	Direction      string            `json:"direction,omitempty" yaml:"direction,omitempty"`
	Timezone       string            `json:"timezone,omitempty" yaml:"timezone,omitempty"`
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

// BlockLayoutItems is the struct for the block layout items
type BlockLayoutItems struct {
	Categories []LayoutItem                 `json:"categories"`
	Locals     map[string]map[string]string `json:"locals,omitempty"`
}

// LayoutItem is the struct for the layout it
type LayoutItem struct {
	ID       string       `json:"id"`
	Label    string       `json:"label,omitempty"`
	Width    int          `json:"width,omitempty"`
	Height   int          `json:"height,omitempty"`
	Keywords []string     `json:"keywords,omitempty"`
	Blocks   []LayoutItem `json:"blocks,omitempty"`
}

// Template is the struct for the template
type Template struct {
	Version      int              `json:"version"` // Yao Builder version
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Descrption   string           `json:"description"`
	Screenshots  []string         `json:"screenshots"`
	Themes       []SelectOption   `json:"themes"`
	Locales      []SelectOption   `json:"locales"`
	Document     []byte           `json:"-"`
	GlobalData   []byte           `json:"-"`
	Scripts      *TemplateScirpts `json:"scripts,omitempty"`
	Translator   string           `json:"translator,omitempty"`
	BuildScript  *Script          `json:"-"` // __build.backend.ts / __build.backend.js
	GlobalScript *Script          `json:"-"` // __global.backend.ts / __global.backend.js
}

// TemplateScirpts is the struct for the template scripts
type TemplateScirpts struct {
	BeforeBuild   []*TemplateScript `json:"before:build,omitempty"`   // Run before build
	AfterBuild    []*TemplateScript `json:"after:build,omitempty"`    // Run after build
	BuildComplete []*TemplateScript `json:"build:complete,omitempty"` // Run build complete
}

// TemplateScript is the struct for the template script
type TemplateScript struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// TemplateScirptResult is the struct for the template script result
type TemplateScirptResult struct {
	Message string          `json:"message,omitempty"`
	Error   error           `json:"error,omitempty"`
	Pid     int             `json:"pid,omitempty"`
	Script  *TemplateScript `json:"script,omitempty"`
}

// Theme is the struct for the theme
type Theme struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// SelectOption is the struct for the select option
type SelectOption struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	Default bool   `json:"default"`
}

// Asset is the struct for the asset
type Asset struct {
	file    string
	Type    string `json:"type"`
	Content []byte `json:"content"`
}

// Media is the struct for the media
type Media struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []byte `json:"content,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Size    int    `json:"size,omitempty"`
	Length  int    `json:"length,omitempty"`
	Thumb   string `json:"thumb,omitempty"`
	URL     string `json:"url,omitempty"`
}

// MediaSearchResult is the struct for the media search result
type MediaSearchResult struct {
	Data      []Media `json:"data"`
	Total     int     `json:"total"`
	Page      int     `json:"page"`
	PageCount int     `json:"pagecnt"`
	PageSize  int     `json:"pagesize"`
	Next      int     `json:"next"`
	Prev      int     `json:"prev"`
}

// BuildOption is the struct for the option option
type BuildOption struct {
	SSR             bool                   `json:"ssr"`
	CDN             bool                   `json:"cdn"`
	UpdateAll       bool                   `json:"update_all"`
	PublicRoot      string                 `json:"public_root,omitempty"`
	AssetRoot       string                 `json:"asset_root,omitempty"`
	IgnoreAssetRoot bool                   `json:"ignore_asset_root,omitempty"`
	IgnoreLibSUI    bool                   `json:"ignore_lib_sui,omitempty"`
	IgnoreDocument  bool                   `json:"ignore_document,omitempty"`
	JitMode         bool                   `json:"jit_mode,omitempty"`
	WithWrapper     bool                   `json:"with_wrapper,omitempty"`
	KeepPageTag     bool                   `json:"keep_page_tag,omitempty"`
	Namespace       string                 `json:"namespace,omitempty"`
	Data            map[string]interface{} `json:"data,omitempty"`
	ComponentName   string                 `json:"component_name,omitempty"`
	ScriptMinify    bool                   `json:"scriptminify,omitempty"`
	StyleMinify     bool                   `json:"styleminify,omitempty"`
	ExecScripts     bool                   `json:"exec_scripts,omitempty"`
	Locales         []string               `json:"locales,omitempty"`
}

// Request is the struct for the request
type Request struct {
	Method    string                 `json:"method"`
	AssetRoot string                 `json:"asset_root,omitempty"`
	Referer   string                 `json:"referer,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Query     url.Values             `json:"query,omitempty"`
	Params    map[string]string      `json:"params,omitempty"`
	Headers   url.Values             `json:"headers,omitempty"`
	Body      interface{}            `json:"body,omitempty"`
	URL       ReqeustURL             `json:"url,omitempty"`
	Sid       string                 `json:"sid,omitempty"`
	Theme     any                    `json:"theme,omitempty"`
	Locale    any                    `json:"locale,omitempty"`
	Script    *Script                `json:"-"`
}

// RequestSource is the struct for the request
type RequestSource struct {
	UID        string                  `json:"uid"`
	User       string                  `json:"user,omitempty"`
	Page       *SourceData             `json:"page,omitempty"`
	Style      *SourceData             `json:"style,omitempty"`
	Script     *SourceData             `json:"script,omitempty"`
	Data       *SourceData             `json:"data,omitempty"`
	Board      *BoardSourceData        `json:"board,omitempty"`
	Mock       *PageMock               `json:"mock,omitempty"`
	Setting    *PageSetting            `json:"setting,omitempty"`
	NeedToSave ReqeustSourceNeedToSave `json:"needToSave,omitempty"`
}

// ReqeustSourceNeedToSave is the struct for the request
type ReqeustSourceNeedToSave struct {
	Page     bool `json:"page,omitempty"`
	Style    bool `json:"style,omitempty"`
	Script   bool `json:"script,omitempty"`
	Data     bool `json:"data,omitempty"`
	Board    bool `json:"board,omitempty"`
	Mock     bool `json:"mock,omitempty"`
	Setting  bool `json:"setting,omitempty"`
	Validate bool `json:"validate,omitempty"`
}

// ResponseEditorRender is the struct for the response
type ResponseEditorRender struct {
	HTML     string                 `json:"html,omitempty"`
	CSS      string                 `json:"css,omitempty"`
	Scripts  []string               `json:"scripts,omitempty"`
	Styles   []string               `json:"styles,omitempty"`
	Setting  map[string]interface{} `json:"setting,omitempty"`
	Config   *PageConfig            `json:"config,omitempty"`
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

// PageMock is the struct for the request
type PageMock struct {
	Method  string                 `json:"method,omitempty"`
	Referer string                 `json:"referer,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Query   url.Values             `json:"query,omitempty"`
	Params  map[string]string      `json:"params,omitempty"`
	Headers url.Values             `json:"headers,omitempty"`
	Body    interface{}            `json:"body,omitempty"`
	URL     ReqeustURL             `json:"url,omitempty"`
	Sid     string                 `json:"sid,omitempty"`
}

// ReqeustURL is the struct for the request
type ReqeustURL struct {
	Host   string `json:"host,omitempty"`
	Domain string `json:"domain,omitempty"`
	Path   string `json:"path,omitempty"`
	Scheme string `json:"scheme,omitempty"`
	URL    string `json:"url,omitempty"`
}

// PageConfig is the struct for the page config
type PageConfig struct {
	PageSetting `json:",omitempty"`
	Mock        *PageMock           `json:"mock,omitempty"`
	Rendered    *PageConfigRendered `json:"rendered,omitempty"`
}

// PageSetting is the struct for the page setting
type PageSetting struct {
	Title       string   `json:"title,omitempty"`
	Guard       string   `json:"guard,omitempty"`
	CacheStore  string   `json:"cacheStore,omitempty"`
	Cache       int      `json:"cache,omitempty"`
	Root        string   `json:"root,omitempty"`
	DataCache   int      `json:"dataCache,omitempty"`
	Description string   `json:"description,omitempty"`
	SEO         *PageSEO `json:"seo,omitempty"`
	API         *PageAPI `json:"api,omitempty"`
}

// PageConfigRendered is the struct for the page config rendered
type PageConfigRendered struct {
	Title string `json:"title,omitempty"`
	Link  string `json:"link,omitempty"`
}

// PageAPI is the struct for the page api
type PageAPI struct {
	Prefix       string            `json:"prefix,omitempty"`
	DefaultGuard string            `json:"defaultGuard,omitempty"`
	Guards       map[string]string `json:"guards,omitempty"`
}

// PageSEO is the struct for the page seo
type PageSEO struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
}

// SourceCodes is the struct for the page codes
type SourceCodes struct {
	HTML Source `json:"-"`
	CSS  Source `json:"-"`
	JS   Source `json:"-"`
	TS   Source `json:"-"`
	LESS Source `json:"-"`
	DATA Source `json:"-"`
	CONF Source `json:"-"`
}

// Source is the struct for the source
type Source struct {
	File string `json:"-"`
	Code string `json:"-"`
}

// Public is the struct for the static
type Public struct {
	Host    string `json:"host,omitempty"`
	Root    string `json:"root,omitempty"`
	Index   string `json:"index,omitempty"`
	Matcher string `json:"matcher,omitempty"`
}

// Storage is the struct for the storage
type Storage struct {
	Driver string                 `json:"driver"`
	Option map[string]interface{} `json:"option,omitempty"`
}

// Matcher the struct for the matcher
type Matcher struct {
	Regex  *regexp.Regexp `json:"regex,omitempty"`
	Exact  string         `json:"exact,omitempty"`
	Parent string         `json:"-"`
	Ref    string         `json:"-"`
}

// DocumentDefault is the default document
var DocumentDefault = []byte(`
<!DOCTYPE html>
<html locale="{{ $locale ?? 'en-us' }}" class="{{ $theme }}" >
  <head>
    <meta charset="UTF-8" />
    <title>{{ $global.title ?? 'Untitled' }}</title>
    <meta
      name="viewport"
      content="width=device-width, initial-scale=1, shrink-to-fit=no"
    />
    <meta
      name="description"
      content="{{ $global.description ?? '' }}"
    />
    <meta
      name="keywords"
      content="{{ $global.keywords ?? '' }}"
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
