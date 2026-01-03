package agent

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
	"golang.org/x/text/language"
)

// Template is the struct for the agent sui template
type Template struct {
	Root    string `json:"-"`
	agent   *Agent
	locales []core.SelectOption
	loaded  map[string]core.IPage
	*core.Template
}

// Pages get the pages from both /agent/pages and /assistants/*/pages
func (tmpl *Template) Pages() ([]core.IPage, error) {
	pages := []core.IPage{}

	// 1. Get pages from /agent/pages (global agent pages like login, error, etc.)
	agentPagesDir := filepath.Join(tmpl.agent.root, "pages")
	if tmpl.agent.fs.IsDir(agentPagesDir) {
		agentPages, err := tmpl.getPagesFromDir(agentPagesDir, "")
		if err != nil {
			log.Error("[Agent] Failed to load agent pages: %v", err)
		} else {
			pages = append(pages, agentPages...)
		}
	}

	// 2. Get pages from each assistant's pages directory
	assistants, err := tmpl.agent.getAssistants()
	if err != nil {
		return nil, err
	}

	for _, assistantID := range assistants {
		pagesDir := tmpl.agent.getAssistantPagesRoot(assistantID)
		assistantPages, err := tmpl.getPagesFromDir(pagesDir, assistantID)
		if err != nil {
			log.Error("[Agent] Failed to load pages for assistant %s: %v", assistantID, err)
			continue
		}
		pages = append(pages, assistantPages...)
	}

	return pages, nil
}

// getPagesFromDir get pages from a directory with optional route prefix
func (tmpl *Template) getPagesFromDir(dir string, routePrefix string) ([]core.IPage, error) {
	exts := []string{"*.sui", "*.html", "*.htm", "*.page"}
	pages := []core.IPage{}

	tmpl.agent.fs.Walk(dir, func(root, file string, isdir bool) error {
		name := filepath.Base(file)
		if isdir {
			if strings.HasPrefix(name, "__") || name == ".tmp" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(name, "__") {
			return nil
		}

		page, err := tmpl.getPageFrom(file, dir, routePrefix)
		if err != nil {
			log.Error("[Agent] Get page error: %v", err)
			return nil
		}

		pages = append(pages, page)
		return nil
	}, exts...)

	return pages, nil
}

// getPageFrom create a page from file
func (tmpl *Template) getPageFrom(file, pagesRoot, assistantID string) (core.IPage, error) {
	route := tmpl.getPageRoute(file, pagesRoot, assistantID)
	return tmpl.getPage(route, file, pagesRoot, assistantID)
}

// getPageRoute get the route for a page
func (tmpl *Template) getPageRoute(file, pagesRoot, assistantID string) string {
	// Get relative path from pages root
	relPath := filepath.Dir(file[len(pagesRoot):])

	// Add assistant prefix if this is an assistant page
	if assistantID != "" {
		return filepath.Join("/", assistantID, relPath)
	}

	return relPath
}

// getPage create a page object
func (tmpl *Template) getPage(route, file, pagesRoot, assistantID string) (core.IPage, error) {
	path := filepath.Dir(file)
	name := tmpl.getPageBase(route)

	return &Page{
		Page: &core.Page{
			Route:      route,
			Path:       path,
			Name:       name,
			TemplateID: tmpl.ID,
			SuiID:      tmpl.agent.DSL.ID,
			Codes: core.SourceCodes{
				HTML: core.Source{File: fmt.Sprintf("%s%s", name, filepath.Ext(file))},
				CSS:  core.Source{File: fmt.Sprintf("%s.css", name)},
				JS:   core.Source{File: fmt.Sprintf("%s.js", name)},
				DATA: core.Source{File: fmt.Sprintf("%s.json", name)},
				TS:   core.Source{File: fmt.Sprintf("%s.ts", name)},
				LESS: core.Source{File: fmt.Sprintf("%s.less", name)},
				CONF: core.Source{File: fmt.Sprintf("%s.config", name)},
			},
		},
		tmpl:        tmpl,
		assistantID: assistantID,
		pagesRoot:   pagesRoot,
	}, nil
}

func (tmpl *Template) getPageBase(route string) string {
	return filepath.Base(route)
}

// Page get a specific page by route
func (tmpl *Template) Page(route string) (core.IPage, error) {
	// Parse the route to determine if it's an assistant page or agent page
	parts := strings.Split(strings.Trim(route, "/"), "/")

	if len(parts) == 0 {
		return nil, fmt.Errorf("Invalid route: %s", route)
	}

	// Check if first part is an assistant ID
	assistants, err := tmpl.agent.getAssistants()
	if err != nil {
		return nil, err
	}

	assistantID := ""
	pageRoute := route
	pagesRoot := filepath.Join(tmpl.agent.root, "pages")

	for _, ast := range assistants {
		if parts[0] == ast {
			assistantID = ast
			pageRoute = "/" + strings.Join(parts[1:], "/")
			pagesRoot = tmpl.agent.getAssistantPagesRoot(assistantID)
			break
		}
	}

	// Find the page file
	pagePath := tmpl.getPagePath(pageRoute, pagesRoot)
	exts := []string{".sui", ".html", ".htm", ".page"}

	for _, ext := range exts {
		file := fmt.Sprintf("%s%s", pagePath, ext)
		if tmpl.agent.fs.IsFile(file) {
			return tmpl.getPage(route, file, pagesRoot, assistantID)
		}
	}

	return nil, fmt.Errorf("Page not found: %s", route)
}

func (tmpl *Template) getPagePath(route, pagesRoot string) string {
	name := tmpl.getPageBase(route)
	return filepath.Join(pagesRoot, route, name)
}

// PageExist check if page exists
func (tmpl *Template) PageExist(route string) bool {
	_, err := tmpl.Page(route)
	return err == nil
}

// RemovePage remove a page (not supported)
func (tmpl *Template) RemovePage(route string) error {
	return fmt.Errorf("RemovePage is not supported for agent pages")
}

// GetPageFromAsset get page from asset
func (tmpl *Template) GetPageFromAsset(file string) (core.IPage, error) {
	route := filepath.Dir(file)
	return tmpl.Page(route)
}

// CreateEmptyPage create an empty page (not supported)
func (tmpl *Template) CreateEmptyPage(route string, setting *core.PageSetting) (core.IPage, error) {
	return nil, fmt.Errorf("CreateEmptyPage is not supported for agent pages")
}

// CreatePage create a page from source (not supported for editing)
func (tmpl *Template) CreatePage(source string) core.IPage {
	// This is used for rendering, we need to find the page by route
	page, err := tmpl.Page(source)
	if err != nil {
		log.Error("[Agent] CreatePage error: %v", err)
		return nil
	}
	return page
}

// GetRoot get the root path (returns agent root for assets, etc.)
func (tmpl *Template) GetRoot() string {
	return tmpl.agent.root
}

// GetWatchDirs returns all directories that should be watched for changes
// This implements the core.IWatchDirs interface
func (tmpl *Template) GetWatchDirs() []string {
	dirs := []string{}

	// 1. Add the main agent template directory
	dirs = append(dirs, tmpl.agent.root)

	// 2. Add each assistant's pages directory
	assistants, err := tmpl.agent.getAssistants()
	if err != nil {
		return dirs
	}

	for _, assistantID := range assistants {
		pagesDir := tmpl.agent.getAssistantPagesRoot(assistantID)
		dirs = append(dirs, pagesDir)
	}

	return dirs
}

// GetWatchRoot returns "app" to indicate paths are relative to application source root
func (tmpl *Template) GetWatchRoot() string {
	return "app"
}

// Asset get the asset (check agent assets first, then assistant assets)
func (tmpl *Template) Asset(file string, width, height uint) (*core.Asset, error) {
	// First check in agent assets
	agentFile := filepath.Join(tmpl.agent.root, "__assets", file)
	if tmpl.agent.fs.IsFile(agentFile) {
		return tmpl.readAsset(agentFile, width, height)
	}

	// If not found and this is an assistant-specific request, check assistant assets
	// Format: /<assistant-id>/assets/...
	parts := strings.SplitN(strings.TrimPrefix(file, "/"), "/", 2)
	if len(parts) >= 2 {
		assistantID := parts[0]
		assetPath := parts[1]

		// Check if this is a valid assistant
		assistants, _ := tmpl.agent.getAssistants()
		for _, ast := range assistants {
			if ast == assistantID {
				assistantAssetFile := filepath.Join(tmpl.agent.assistantsRoot, assistantID, "pages", "__assets", assetPath)
				if tmpl.agent.fs.IsFile(assistantAssetFile) {
					return tmpl.readAsset(assistantAssetFile, width, height)
				}
				break
			}
		}
	}

	return nil, fmt.Errorf("Asset %s not found", file)
}

// readAsset read asset from file
func (tmpl *Template) readAsset(file string, width, height uint) (*core.Asset, error) {
	content, err := tmpl.agent.fs.ReadFile(file)
	if err != nil {
		return nil, err
	}

	typ, err := tmpl.agent.fs.MimeType(file)
	if err != nil {
		typ = "application/octet-stream"
	}

	return &core.Asset{Type: typ, Content: content}, nil
}

// Locales get the global locales
func (tmpl *Template) Locales() []core.SelectOption {
	if tmpl.locales != nil {
		return tmpl.locales
	}

	supportLocales := []core.SelectOption{}
	localeMap := map[string]bool{}

	// Check __locales directory
	path := filepath.Join(tmpl.Root, "__locales")
	if !tmpl.agent.fs.IsDir(path) {
		return supportLocales
	}

	dirs, err := tmpl.agent.fs.ReadDir(path, false)
	if err != nil {
		return supportLocales
	}

	for _, dir := range dirs {
		locale := filepath.Base(dir)
		if localeMap[locale] {
			continue
		}
		label := language.Make(locale).String()
		localeMap[locale] = true
		supportLocales = append(supportLocales, core.SelectOption{
			Value: locale,
			Label: label,
		})
	}

	tmpl.locales = supportLocales
	return tmpl.locales
}

// Themes get the global themes
func (tmpl *Template) Themes() []core.SelectOption {
	return tmpl.Template.Themes
}

// Assets get the assets
func (tmpl *Template) Assets() []string {
	return nil
}

// Glob the files
func (tmpl *Template) Glob(pattern string) ([]string, error) {
	path := filepath.Join(tmpl.Root, pattern)
	paths, err := tmpl.agent.fs.Glob(path)
	if err != nil {
		return nil, err
	}

	routes := []string{}
	for _, p := range paths {
		routes = append(routes, strings.TrimPrefix(p, tmpl.Root))
	}
	return routes, nil
}

// GlobRoutes the files
func (tmpl *Template) GlobRoutes(patterns []string, unique ...bool) ([]string, error) {
	routes := []string{}
	for _, pattern := range patterns {
		paths, err := tmpl.Glob(pattern)
		if err != nil {
			return nil, err
		}

		for _, path := range paths {
			if !tmpl.agent.fs.IsDir(filepath.Join(tmpl.Root, path)) {
				continue
			}
			routes = append(routes, path)
		}
	}

	if len(unique) > 0 && unique[0] {
		mapRoutes := map[string]bool{}
		for _, route := range routes {
			mapRoutes[route] = true
		}

		routes = []string{}
		for route := range mapRoutes {
			routes = append(routes, route)
		}
	}

	return routes, nil
}

// Reload the template
func (tmpl *Template) Reload() error {
	return nil
}

// PageTree gets the page tree
func (tmpl *Template) PageTree(route string) ([]*core.PageTreeNode, error) {
	return nil, nil
}

// MediaSearch search the asset
func (tmpl *Template) MediaSearch(query url.Values, page int, pageSize int) (core.MediaSearchResult, error) {
	return core.MediaSearchResult{Data: []core.Media{}, Page: page, PageSize: pageSize}, nil
}

// AssetUpload upload the asset (not supported)
func (tmpl *Template) AssetUpload(reader io.Reader, name string) (string, error) {
	return "", fmt.Errorf("AssetUpload is not supported for agent template")
}

// Block get the block (not supported)
func (tmpl *Template) Block(name string) (core.IBlock, error) {
	return nil, fmt.Errorf("Block is not supported for agent template")
}

// Blocks get the blocks (not supported)
func (tmpl *Template) Blocks() ([]core.IBlock, error) {
	return nil, nil
}

// BlockLayoutItems get block layout items
func (tmpl *Template) BlockLayoutItems() (*core.BlockLayoutItems, error) {
	return nil, nil
}

// BlockMedia get block media
func (tmpl *Template) BlockMedia(id string) (*core.Asset, error) {
	return nil, fmt.Errorf("BlockMedia is not supported for agent template")
}

// Component get the component (not supported)
func (tmpl *Template) Component(name string) (core.IComponent, error) {
	return nil, fmt.Errorf("Component is not supported for agent template")
}

// Components get the components (not supported)
func (tmpl *Template) Components() ([]core.IComponent, error) {
	return nil, nil
}

// SupportLocales get the support locales
func (tmpl *Template) SupportLocales() []string {
	locales := tmpl.Locales()
	result := make([]string, len(locales))
	for i, locale := range locales {
		result[i] = locale.Value
	}
	return result
}

// ExecBeforeBuildScripts execute the before build scripts
func (tmpl *Template) ExecBeforeBuildScripts() []core.TemplateScirptResult {
	return nil
}

// ExecAfterBuildScripts execute the after build scripts
func (tmpl *Template) ExecAfterBuildScripts() []core.TemplateScirptResult {
	return nil
}

// ExecBuildCompleteScripts execute the build complete scripts
func (tmpl *Template) ExecBuildCompleteScripts() []core.TemplateScirptResult {
	return nil
}

// Build build the template
func (tmpl *Template) Build(option *core.BuildOption) ([]string, error) {
	warnings := []string{}

	// Execute before build scripts
	tmpl.ExecBeforeBuildScripts()

	root, err := tmpl.agent.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("Build: Get the public root error: %s. use %s", err.Error(), tmpl.agent.DSL.Public.Root)
		root = tmpl.agent.DSL.Public.Root
	}

	if option.AssetRoot == "" {
		option.AssetRoot = filepath.Join(root, "assets")
	}
	option.PublicRoot = root

	// Sync the assets
	if err = tmpl.SyncAssets(option); err != nil {
		return warnings, err
	}

	// Get all pages
	pages, err := tmpl.Pages()
	if err != nil {
		return warnings, err
	}

	// Build global context
	globalCtx := core.NewGlobalBuildContext(tmpl)

	// Build each page
	tmpl.loaded = map[string]core.IPage{}
	for _, page := range pages {
		if err := page.Load(); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to load page %s: %v", page.Get().Route, err))
			continue
		}

		pageWarnings, err := page.Build(globalCtx, option)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to build page %s: %v", page.Get().Route, err))
			continue
		}
		warnings = append(warnings, pageWarnings...)
		tmpl.loaded[page.Get().Route] = page
	}

	// Add sui lib to the global
	err = tmpl.UpdateJSSDK(option)
	if err != nil {
		return warnings, err
	}

	// Execute after build scripts
	tmpl.ExecAfterBuildScripts()

	return warnings, nil
}

// SyncAssets sync assets from template __assets to public
func (tmpl *Template) SyncAssets(option *core.BuildOption) error {
	// Get source abs path
	sourceRoot := filepath.Join(tmpl.agent.fs.Root(), tmpl.Root, "__assets")
	if exist, _ := os.Stat(sourceRoot); exist == nil {
		return nil
	}

	// Get target abs path
	root, err := tmpl.agent.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), tmpl.agent.DSL.Public.Root)
		root = tmpl.agent.DSL.Public.Root
	}

	targetRoot := filepath.Join(application.App.Root(), "public", root, "assets")
	if exist, _ := os.Stat(targetRoot); exist == nil {
		os.MkdirAll(targetRoot, os.ModePerm)
	}

	// Copy the assets
	return tmpl.copyDir(sourceRoot, targetRoot)
}

// copyDir copy directory recursively
func (tmpl *Template) copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, os.ModePerm)
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, 0644)
	})
}

// SyncAssetFile sync asset file
func (tmpl *Template) SyncAssetFile(file string, option *core.BuildOption) error {
	sourceRoot := filepath.Join(tmpl.agent.fs.Root(), tmpl.Root, "__assets")
	if exist, _ := os.Stat(sourceRoot); exist == nil {
		return nil
	}

	root, err := tmpl.agent.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssetFile: Get the public root error: %s. use %s", err.Error(), tmpl.agent.DSL.Public.Root)
		root = tmpl.agent.DSL.Public.Root
	}

	targetRoot := filepath.Join(application.App.Root(), "public", root, "assets")
	sourceFile := filepath.Join(sourceRoot, file)
	targetFile := filepath.Join(targetRoot, file)

	// Create the target directory
	dir := filepath.Dir(targetFile)
	if exist, _ := os.Stat(dir); exist == nil {
		os.MkdirAll(dir, os.ModePerm)
	}

	// Copy file
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return err
	}

	return os.WriteFile(targetFile, data, 0644)
}

// UpdateJSSDK update the JS SDK (libsui.min.js)
func (tmpl *Template) UpdateJSSDK(option *core.BuildOption) error {
	root, err := tmpl.agent.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("UpdateJSSDK: Get the public root error: %s. use %s", err.Error(), tmpl.agent.DSL.Public.Root)
		root = tmpl.agent.DSL.Public.Root
	}

	targetRoot := filepath.Join(application.App.Root(), "public", root, "assets")
	if exist, _ := os.Stat(targetRoot); exist == nil {
		os.MkdirAll(targetRoot, os.ModePerm)
	}

	// Get libsui source
	libsui, libsuiMap, err := core.LibSUI()
	if err != nil {
		return err
	}

	// Write libsui.min.js
	file := filepath.Join(targetRoot, "libsui.min.js")
	err = os.WriteFile(file, libsui, 0644)
	if err != nil {
		return err
	}

	// Write libsui.min.js.map
	mapFile := filepath.Join(targetRoot, "libsui.min.js.map")
	err = os.WriteFile(mapFile, libsuiMap, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Trans translate the template
func (tmpl *Template) Trans(option *core.BuildOption) ([]string, error) {
	warnings := []string{}

	// Get all pages
	pages, err := tmpl.Pages()
	if err != nil {
		return warnings, err
	}

	// Build global context
	globalCtx := core.NewGlobalBuildContext(tmpl)

	// Translate each page
	for _, page := range pages {
		if err := page.Load(); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to load page %s: %v", page.Get().Route, err))
			continue
		}

		pageWarnings, err := page.Trans(globalCtx, option)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to translate page %s: %v", page.Get().Route, err))
			continue
		}
		warnings = append(warnings, pageWarnings...)
	}

	return warnings, nil
}

// loadBuildScript load the build script
func (tmpl *Template) loadBuildScript() error {
	file, source, err := tmpl.backendScriptSource("__build.backend")
	if err != nil {
		return err
	}

	if file == "" {
		return nil
	}

	script, err := v8.MakeScript(source, file, 5*time.Second)
	if err != nil {
		return err
	}
	tmpl.BuildScript = &core.Script{Script: script}
	return nil
}

func (tmpl *Template) backendScriptSource(name string) (string, []byte, error) {
	path := filepath.Join(tmpl.Root, fmt.Sprintf("%s.ts", name))
	if !tmpl.agent.fs.IsFile(path) {
		path = filepath.Join(tmpl.Root, fmt.Sprintf("%s.js", name))
	}

	if !tmpl.agent.fs.IsFile(path) {
		return "", nil, nil
	}

	content, err := tmpl.agent.fs.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return path, content, nil
}
