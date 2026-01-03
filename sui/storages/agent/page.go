package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
	"gopkg.in/yaml.v3"
)

// Page wraps core.Page with agent-specific functionality
type Page struct {
	*core.Page
	tmpl        *Template
	assistantID string
	pagesRoot   string
}

// Load load the page content
func (page *Page) Load() error {
	p := page.Page
	fs := page.tmpl.agent.fs

	// Set document from template
	p.Document = page.tmpl.Document

	// Read HTML
	htmlFile := filepath.Join(p.Path, p.Codes.HTML.File)
	if fs.IsFile(htmlFile) {
		content, err := fs.ReadFile(htmlFile)
		if err != nil {
			return err
		}
		p.Codes.HTML.Code = string(content)
	}

	// Read CSS
	cssFile := filepath.Join(p.Path, p.Codes.CSS.File)
	if fs.IsFile(cssFile) {
		content, err := fs.ReadFile(cssFile)
		if err != nil {
			return err
		}
		p.Codes.CSS.Code = string(content)
	}

	// Read JS
	jsFile := filepath.Join(p.Path, p.Codes.JS.File)
	if fs.IsFile(jsFile) {
		content, err := fs.ReadFile(jsFile)
		if err != nil {
			return err
		}
		p.Codes.JS.Code = string(content)
	}

	// Read TS
	tsFile := filepath.Join(p.Path, p.Codes.TS.File)
	if fs.IsFile(tsFile) {
		content, err := fs.ReadFile(tsFile)
		if err != nil {
			return err
		}
		p.Codes.TS.Code = string(content)
	}

	// Read DATA (JSON)
	dataFile := filepath.Join(p.Path, p.Codes.DATA.File)
	if fs.IsFile(dataFile) {
		content, err := fs.ReadFile(dataFile)
		if err != nil {
			return err
		}
		p.Codes.DATA.Code = string(content)
	}

	// Read Config
	confFile := filepath.Join(p.Path, p.Codes.CONF.File)
	if fs.IsFile(confFile) {
		content, err := fs.ReadFile(confFile)
		if err != nil {
			return err
		}
		p.Codes.CONF.Code = string(content)

		// Parse config
		var config core.PageConfig
		if err := jsoniter.Unmarshal(content, &config); err == nil {
			p.Config = &config
		}
	}

	// Load backend script
	err := page.loadScript()
	if err != nil {
		return err
	}

	return nil
}

// loadScript load the backend script
func (page *Page) loadScript() error {
	p := page.Page
	fs := page.tmpl.agent.fs

	// Try .backend.ts first, then .backend.js
	tsFile := filepath.Join(p.Path, fmt.Sprintf("%s.backend.ts", p.Name))
	jsFile := filepath.Join(p.Path, fmt.Sprintf("%s.backend.js", p.Name))

	var scriptFile string
	if fs.IsFile(tsFile) {
		scriptFile = tsFile
	} else if fs.IsFile(jsFile) {
		scriptFile = jsFile
	}

	if scriptFile == "" {
		return nil
	}

	content, err := fs.ReadFile(scriptFile)
	if err != nil {
		return err
	}

	script, err := v8.MakeScript(content, scriptFile, 5*time.Second)
	if err != nil {
		return err
	}

	p.Script = &core.Script{Script: script}
	return nil
}

// Get get the page info
func (page *Page) Get() *core.Page {
	return page.Page
}

// GetConfig get the page config
func (page *Page) GetConfig() *core.PageConfig {
	p := page.Page
	if p.Config != nil {
		return p.Config
	}

	// Try to load config if not loaded
	fs := page.tmpl.agent.fs
	confFile := filepath.Join(p.Path, p.Codes.CONF.File)
	if fs.IsFile(confFile) {
		content, err := fs.ReadFile(confFile)
		if err != nil {
			return nil
		}

		var config core.PageConfig
		if err := jsoniter.Unmarshal(content, &config); err == nil {
			p.Config = &config
			return p.Config
		}
	}

	return nil
}

// SaveTemp save the page temporarily (not supported for agent pages)
func (page *Page) SaveTemp(request *core.RequestSource) error {
	return fmt.Errorf("SaveTemp is not supported for agent pages")
}

// Save save the page (not supported for agent pages)
func (page *Page) Save(request *core.RequestSource) error {
	return fmt.Errorf("Save is not supported for agent pages")
}

// SaveAs save the page as (not supported for agent pages)
func (page *Page) SaveAs(route string, setting *core.PageSetting) (core.IPage, error) {
	return nil, fmt.Errorf("SaveAs is not supported for agent pages")
}

// Remove remove the page (not supported for agent pages)
func (page *Page) Remove() error {
	return fmt.Errorf("Remove is not supported for agent pages")
}

// SUI get the SUI interface
func (page *Page) SUI() (core.SUI, error) {
	return page.tmpl.agent, nil
}

// Sid get the session id
func (page *Page) Sid() (string, error) {
	return page.tmpl.agent.DSL.Sid, nil
}

// Template get the template
func (page *Page) Template() core.ITemplate {
	return page.tmpl
}

// AssetScript get the script
func (page *Page) AssetScript() (*core.Asset, error) {
	fs := page.tmpl.agent.fs

	// Try .ts first, then .js
	tsFile := filepath.Join(page.Path, page.Codes.TS.File)
	if fs.IsFile(tsFile) {
		tsCode, err := fs.ReadFile(tsFile)
		if err != nil {
			return nil, err
		}

		jsCode, _, err := page.CompileTS(tsCode, false)
		if err != nil {
			return nil, err
		}

		return &core.Asset{
			Type:    "text/javascript; charset=utf-8",
			Content: []byte(jsCode),
		}, nil
	}

	jsFile := filepath.Join(page.Path, page.Codes.JS.File)
	if fs.IsFile(jsFile) {
		jsCode, err := fs.ReadFile(jsFile)
		if err != nil {
			return nil, err
		}

		jsCode, _, err = page.CompileJS(jsCode, false)
		if err != nil {
			return nil, err
		}

		return &core.Asset{
			Type:    "text/javascript; charset=utf-8",
			Content: jsCode,
		}, nil
	}

	return nil, fmt.Errorf("%s script not found", page.Route)
}

// AssetStyle get the style
func (page *Page) AssetStyle() (*core.Asset, error) {
	fs := page.tmpl.agent.fs

	cssFile := filepath.Join(page.Path, page.Codes.CSS.File)
	if fs.IsFile(cssFile) {
		cssCode, err := fs.ReadFile(cssFile)
		if err != nil {
			return nil, err
		}

		cssCode, err = page.CompileCSS(cssCode, false)
		if err != nil {
			return nil, err
		}

		return &core.Asset{
			Type:    "text/css; charset=utf-8",
			Content: cssCode,
		}, nil
	}
	return nil, fmt.Errorf("%s style not found", page.Route)
}

// Build build the page
func (page *Page) Build(globalCtx *core.GlobalBuildContext, option *core.BuildOption) ([]string, error) {
	ctx := core.NewBuildContext(globalCtx)
	root := option.PublicRoot
	if root == "" {
		root = page.tmpl.agent.DSL.Public.Root
	}

	if option.AssetRoot == "" {
		option.AssetRoot = filepath.Join(root, "assets")
	}
	page.Root = root

	// Load page if not loaded
	if page.Codes.HTML.Code == "" {
		if err := page.Load(); err != nil {
			return nil, err
		}
	}

	html, config, warnings, err := page.Page.Compile(ctx, option)
	if err != nil {
		return warnings, fmt.Errorf("Compile the page %s error: %s", page.Route, err.Error())
	}

	// Save the html
	err = page.writeHTML([]byte(html), option.Data)
	if err != nil {
		return warnings, fmt.Errorf("Write the page %s error: %s", page.Route, err.Error())
	}

	// Save the backend script file
	err = page.writeBackendScript(option.Data)
	if err != nil {
		return warnings, fmt.Errorf("Write the backend script file error: %s", err.Error())
	}

	// Save the config file
	err = page.writeConfig([]byte(config), option.Data)
	if err != nil {
		return warnings, err
	}

	// Write locale files from page's __locales directory
	err = page.writeLocaleFiles(option.Data)
	if err != nil {
		log.Warn("[Agent] Write locale files error: %s", err.Error())
		// Don't fail the build for locale errors
	}

	return warnings, nil
}

// publicFile get the public file path
func (page *Page) publicFile(data map[string]interface{}) string {
	root, err := page.tmpl.agent.DSL.PublicRoot(data)
	if err != nil {
		log.Error("publicFile: Get the public root error: %s. use %s", err.Error(), page.tmpl.agent.DSL.Public.Root)
		root = page.tmpl.agent.DSL.Public.Root
	}
	return filepath.Join("/", "public", root, page.Route)
}

// writeHTML write the html to file
func (page *Page) writeHTML(html []byte, data map[string]interface{}) error {
	htmlFile := fmt.Sprintf("%s.sui", page.publicFile(data))
	htmlFileAbs := filepath.Join(application.App.Root(), htmlFile)
	dir := filepath.Dir(htmlFileAbs)
	if exist, _ := os.Stat(dir); exist == nil {
		os.MkdirAll(dir, os.ModePerm)
	}
	err := os.WriteFile(htmlFileAbs, html, 0644)
	if err != nil {
		return err
	}

	core.RemoveCache(htmlFile)
	return nil
}

// writeConfig write the config to file
func (page *Page) writeConfig(config []byte, data map[string]interface{}) error {
	configFile := fmt.Sprintf("%s.cfg", page.publicFile(data))
	configFileAbs := filepath.Join(application.App.Root(), configFile)
	dir := filepath.Dir(configFileAbs)
	if exist, _ := os.Stat(dir); exist == nil {
		os.MkdirAll(dir, os.ModePerm)
	}
	err := os.WriteFile(configFileAbs, config, 0644)
	if err != nil {
		return err
	}
	return nil
}

// backendScriptSource get the backend script source
func (page *Page) backendScriptSource() (string, []byte, error) {
	fs := page.tmpl.agent.fs
	backendFile := filepath.Join(page.Path, fmt.Sprintf("%s.backend.ts", page.Name))
	if !fs.IsFile(backendFile) {
		backendFile = filepath.Join(page.Path, fmt.Sprintf("%s.backend.js", page.Name))
	}

	if !fs.IsFile(backendFile) {
		return "", nil, nil
	}

	source, err := fs.ReadFile(backendFile)
	if err != nil {
		return "", nil, err
	}

	source = []byte(fmt.Sprintf("%s\n%s", source, core.BackendScript(page.Route)))
	return backendFile, source, nil
}

// writeBackendScript write the backend script to file
func (page *Page) writeBackendScript(data map[string]interface{}) error {
	file, source, err := page.backendScriptSource()
	if err != nil {
		return err
	}

	if source == nil {
		return nil
	}

	ext := filepath.Ext(file)
	scriptFile := fmt.Sprintf("%s.backend%s", page.publicFile(data), ext)
	scriptFileAbs := filepath.Join(application.App.Root(), scriptFile)
	dir := filepath.Dir(scriptFileAbs)
	if exist, _ := os.Stat(dir); exist == nil {
		os.MkdirAll(dir, os.ModePerm)
	}

	err = os.WriteFile(scriptFileAbs, []byte(source), 0644)
	if err != nil {
		return err
	}
	core.RemoveCache(scriptFile)
	return nil
}

// BuildAsComponent build the page as component
func (page *Page) BuildAsComponent(globalCtx *core.GlobalBuildContext, option *core.BuildOption) ([]string, error) {
	warnings := []string{}

	if option.AssetRoot == "" {
		root, err := page.tmpl.agent.DSL.PublicRoot(option.Data)
		if err != nil {
			return warnings, err
		}
		option.AssetRoot = root
	}

	// Load page if not loaded
	if page.Codes.HTML.Code == "" {
		if err := page.Load(); err != nil {
			return nil, err
		}
	}

	// BuildAsComponent needs a parent selection, which is handled by the caller
	return warnings, nil
}

// Trans translate the page
func (page *Page) Trans(globalCtx *core.GlobalBuildContext, option *core.BuildOption) ([]string, error) {
	warnings := []string{}
	ctx := core.NewBuildContext(globalCtx)

	_, _, messages, err := page.Page.Compile(ctx, option)
	if err != nil {
		return warnings, err
	}

	// Save translations - messages is []string, not map
	log.Debug("[Agent] Page %s translation messages: %v", page.Route, messages)

	return warnings, nil
}

// AssetRoot get the asset root for this page
func (page *Page) AssetRoot() string {
	// If this is an assistant page, check for assistant-specific assets
	if page.assistantID != "" {
		assistantAssetsDir := filepath.Join(page.pagesRoot, "__assets")
		if page.tmpl.agent.fs.IsDir(assistantAssetsDir) {
			return fmt.Sprintf("/%s/assets", page.assistantID)
		}
	}

	// Default to global agent assets
	return "/assets"
}

// AssistantID get the assistant ID (empty for global agent pages)
func (page *Page) AssistantID() string {
	return page.assistantID
}

// writeLocaleFiles writes locale files from page's __locales directory to public
func (page *Page) writeLocaleFiles(data map[string]interface{}) error {
	fs := page.tmpl.agent.fs

	// Check if page has __locales directory
	localesDir := filepath.Join(page.Path, "__locales")
	if !fs.IsDir(localesDir) {
		return nil
	}

	// Get the public root
	root, err := page.tmpl.agent.DSL.PublicRoot(data)
	if err != nil {
		log.Error("writeLocaleFiles: Get the public root error: %s. use %s", err.Error(), page.tmpl.agent.DSL.Public.Root)
		root = page.tmpl.agent.DSL.Public.Root
	}

	// Read all locale files in __locales directory
	files, err := fs.ReadDir(localesDir, false)
	if err != nil {
		return err
	}

	for _, file := range files {
		// Skip directories
		if fs.IsDir(file) {
			continue
		}

		// Only process .yml files
		if filepath.Ext(file) != ".yml" {
			continue
		}

		// Get locale name (e.g., "zh-cn" from "zh-cn.yml")
		localeName := filepath.Base(file)
		localeName = localeName[:len(localeName)-4] // Remove .yml extension

		// Read the locale file
		content, err := fs.ReadFile(file)
		if err != nil {
			log.Error("[Agent] Read locale file error: %s", err.Error())
			continue
		}

		// Parse the locale file
		var localeData map[string]interface{}
		err = yaml.Unmarshal(content, &localeData)
		if err != nil {
			log.Error("[Agent] Parse locale file error: %s", err.Error())
			continue
		}

		// Convert to the format expected by core.Locale
		locale := core.Locale{
			Name:           localeName,
			Keys:           map[string]string{},
			Messages:       map[string]string{},
			ScriptMessages: map[string]string{},
		}

		// Extract messages
		if messages, ok := localeData["messages"].(map[string]interface{}); ok {
			for k, v := range messages {
				if strVal, ok := v.(string); ok {
					locale.Messages[k] = strVal
				}
			}
		}

		// Extract script_messages
		if scriptMessages, ok := localeData["script_messages"].(map[string]interface{}); ok {
			for k, v := range scriptMessages {
				if strVal, ok := v.(string); ok {
					locale.ScriptMessages[k] = strVal
				}
			}
		}

		// Extract timezone and direction
		if tz, ok := localeData["timezone"].(string); ok {
			locale.Timezone = tz
		}
		if dir, ok := localeData["direction"].(string); ok {
			locale.Direction = dir
		}

		// Write to public/.locales/<locale>/<route>.yml
		// page.Route may contain path like /expense/test, so we need to create nested directories
		targetFile := filepath.Join(application.App.Root(), "public", root, ".locales", localeName, fmt.Sprintf("%s.yml", page.Route))
		targetDir := filepath.Dir(targetFile)
		if exist, _ := os.Stat(targetDir); exist == nil {
			os.MkdirAll(targetDir, os.ModePerm)
		}

		localeContent, err := yaml.Marshal(locale)
		if err != nil {
			log.Error("[Agent] Marshal locale error: %s", err.Error())
			continue
		}

		err = os.WriteFile(targetFile, localeContent, 0644)
		if err != nil {
			log.Error("[Agent] Write locale file error: %s", err.Error())
			continue
		}

		log.Info("[Agent] Wrote locale file: %s", targetFile)
	}

	return nil
}
