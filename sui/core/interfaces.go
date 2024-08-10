package core

import (
	"io"
	"net/url"
	"regexp"
)

// SUIs the loaded SUI instances
var SUIs = map[string]SUI{}

// RouteMatchers the route matchers for the SUI instance
var RouteMatchers = map[*regexp.Regexp][][]*Matcher{}

// RouteExactMatchers the route exact matchers for the SUI instance
var RouteExactMatchers = map[string][][]*Matcher{}

// RouteRegexp the regexp for the route
var RouteRegexp = regexp.MustCompile(`([a-z0-9A-Z_\-]+)`)

// SUI is the interface for the SUI
type SUI interface {
	Setting() (*Setting, error)
	GetTemplates() ([]ITemplate, error)
	GetTemplate(name string) (ITemplate, error)
	UploadTemplate(src string, dst string) (ITemplate, error)
	WithSid(sid string)
	GetSid() string
	PublicRootMatcher() *Matcher
	GetPublic() *Public
	PublicRootWithSid(sid string) (string, error)
	PublicRoot(data map[string]any) (string, error)
}

// ITemplate is the interface for the ITemplate
type ITemplate interface {
	Pages() ([]IPage, error)
	PageTree(route string) ([]*PageTreeNode, error)
	Page(route string) (IPage, error)
	PageExist(route string) bool
	CreatePage(html string) IPage
	CreateEmptyPage(route string, setting *PageSetting) (IPage, error)
	RemovePage(route string) error
	GetPageFromAsset(asset string) (IPage, error)

	Blocks() ([]IBlock, error)
	BlockLayoutItems() (*BlockLayoutItems, error)
	BlockMedia(id string) (*Asset, error)
	Block(name string) (IBlock, error)

	Components() ([]IComponent, error)
	Component(name string) (IComponent, error)

	Assets() []string
	Locales() []SelectOption
	Themes() []SelectOption

	Asset(file string, width, height uint) (*Asset, error)
	AssetUpload(reader io.Reader, name string) (string, error)

	MediaSearch(query url.Values, page int, pageSize int) (MediaSearchResult, error)

	Build(option *BuildOption) ([]string, error)
	SyncAssets(option *BuildOption) error
	SyncAssetFile(file string, option *BuildOption) error
	GetRoot() string

	ExecBeforeBuildScripts() []TemplateScirptResult
	ExecAfterBuildScripts() []TemplateScirptResult

	Trans(option *BuildOption) ([]string, error)
}

// IPage is the interface for the page
type IPage interface {
	Load() error

	SUI() (SUI, error)
	Sid() (string, error)

	Get() *Page
	GetConfig() *PageConfig
	SaveAs(route string, setting *PageSetting) (IPage, error)
	Save(request *RequestSource) error
	SaveTemp(request *RequestSource) error
	Remove() error

	EditorRender() (*ResponseEditorRender, error)
	EditorPageSource() SourceData
	EditorScriptSource() SourceData
	EditorStyleSource() SourceData
	EditorDataSource() SourceData

	PreviewRender(referer string) (string, error)

	AssetScript() (*Asset, error)
	AssetStyle() (*Asset, error)

	Build(globalCtx *GlobalBuildContext, option *BuildOption) ([]string, error)
	BuildAsComponent(globalCtx *GlobalBuildContext, option *BuildOption) ([]string, error)

	Trans(globalCtx *GlobalBuildContext, option *BuildOption) ([]string, error)
}

// IBlock is the interface for the block
type IBlock interface {
	Compile() (string, error)
	Load() error
	Source() string
	Get() *Block
}

// IComponent is the interface for the component
type IComponent interface {
	Compile() (string, error)
	Load() error
	Source() string
}
