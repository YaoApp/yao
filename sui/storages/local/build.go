package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Build the template
func (tmpl *Template) Build(option *core.BuildOption) ([]string, error) {
	var err error
	warnings := []string{}
	defer func() {
		if option.ExecScripts {
			tmpl.ExecBuildCompleteScripts()
		}
	}()

	// Execute the build before hook
	if option.ExecScripts {
		res := tmpl.ExecBeforeBuildScripts()
		scriptsErrorMessages := []string{}
		for _, r := range res {
			if r.Error != nil {
				scriptsErrorMessages = append(scriptsErrorMessages, fmt.Sprintf("%s: %s", r.Script.Content, r.Error.Error()))
			}
		}
		if len(scriptsErrorMessages) > 0 {
			return warnings, fmt.Errorf("Build scripts error: %s", strings.Join(scriptsErrorMessages, ";\n"))
		}

		err = tmpl.Reload()
		if err != nil {
			return warnings, err
		}
	}

	root, err := tmpl.local.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), tmpl.local.DSL.Public.Root)
		root = tmpl.local.DSL.Public.Root
	}

	if option.AssetRoot == "" {
		option.AssetRoot = filepath.Join(root, "assets")
	}

	// Write the global script
	err = tmpl.writeGlobalScript(option.Data)
	if err != nil {
		return warnings, err
	}

	// Sync the assets
	if err = tmpl.SyncAssets(option); err != nil {
		return warnings, err
	}

	// Build all pages
	ctx := core.NewGlobalBuildContext()
	pages, err := tmpl.Pages()
	if err != nil {
		return warnings, err
	}

	// loaed pages
	tmpl.loaded = map[string]core.IPage{}
	publicRoot, err := tmpl.local.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("Get the public root error: %s. use %s", err.Error(), publicRoot)
	}
	option.PublicRoot = publicRoot
	for _, page := range pages {
		err := page.Load()
		if err != nil {
			return warnings, err
		}
		messages, err := page.Build(ctx, option)
		if err != nil {
			return warnings, err
		}

		if len(messages) > 0 {
			warnings = append(warnings, messages...)
		}

		tmpl.loaded[page.Get().Route] = page
	}

	// Build jit components for the global <route> -> <name>.sui.lib
	jitComponents, err := tmpl.GlobRoutes(ctx.GetJitComponents(), true)
	if err != nil {
		return warnings, err
	}

	for _, route := range jitComponents {
		page, has := tmpl.loaded[route]
		if !has {
			// err = multierror.Append(fmt.Errorf("The page %s is not loaded", route))
			log.Warn("The page %s is not loaded", route)
			continue
		}

		messages, err := page.BuildAsComponent(ctx, option)
		if err != nil {
			err = multierror.Append(err)
		}
		if len(messages) > 0 {
			warnings = append(warnings, messages...)
		}
	}

	// Add sui lib to the global
	err = tmpl.UpdateJSSDK(option)
	if err != nil {
		return warnings, err
	}

	// Execute the build after hook
	if option.ExecScripts {
		res := tmpl.ExecAfterBuildScripts()
		scriptsErrorMessages := []string{}
		for _, r := range res {
			if r.Error != nil {
				scriptsErrorMessages = append(scriptsErrorMessages, fmt.Sprintf("%s: %s", r.Script.Content, r.Error.Error()))
			}
		}
		if len(scriptsErrorMessages) > 0 {
			return warnings, fmt.Errorf("Build scripts error: %s", strings.Join(scriptsErrorMessages, ";\n"))
		}
	}

	return warnings, err
}

// Trans the template
func (tmpl *Template) Trans(option *core.BuildOption) ([]string, error) {
	var err error
	warnings := []string{}

	ctx := core.NewGlobalBuildContext()
	pages, err := tmpl.Pages()
	if err != nil {
		return warnings, err
	}

	// loaed pages
	for _, page := range pages {
		err := page.Load()
		if err != nil {
			return warnings, err
		}

		messages, err := page.Trans(ctx, option)
		if err != nil {
			return warnings, err
		}

		if len(messages) > 0 {
			warnings = append(warnings, messages...)
		}
	}

	return warnings, nil
}

// SyncAssetFile sync the assets
func (tmpl *Template) SyncAssetFile(file string, option *core.BuildOption) error {

	// get source abs path
	sourceRoot := filepath.Join(tmpl.local.fs.Root(), tmpl.Root, "__assets")
	if exist, _ := os.Stat(sourceRoot); exist == nil {
		return nil
	}

	//get target abs path
	root, err := tmpl.local.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), tmpl.local.DSL.Public.Root)
		root = tmpl.local.DSL.Public.Root
	}

	targetRoot := filepath.Join(application.App.Root(), "public", root, "assets")
	sourceFile := filepath.Join(sourceRoot, file)
	targetFile := filepath.Join(targetRoot, file)

	// create the target directory
	if exist, _ := os.Stat(targetFile); exist == nil {
		os.MkdirAll(filepath.Dir(targetFile), os.ModePerm)
	}

	return copy(sourceFile, targetFile)
}

// UpdateJSSDK update the js sdk
func (tmpl *Template) UpdateJSSDK(option *core.BuildOption) error {

	jsCode, sourceMap, err := core.LibSUI()
	if err != nil {
		return err
	}

	// get source abs path
	root, err := tmpl.local.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), tmpl.local.DSL.Public.Root)
		root = tmpl.local.DSL.Public.Root
	}

	targetRoot := filepath.Join(application.App.Root(), "public", root, "assets")

	file := filepath.Join(targetRoot, "libsui.min.js")
	mapFile := filepath.Join(targetRoot, "libsui.min.js.map")

	// create the target directory
	if exist, _ := os.Stat(targetRoot); exist == nil {
		os.MkdirAll(targetRoot, os.ModePerm)
	}

	// write the js sdk
	// add source map url
	jsCode = append(jsCode, []byte("\n//# sourceMappingURL=libsui.min.js.map")...)
	err = os.WriteFile(file, jsCode, 0644)
	if err != nil {
		return err
	}

	// write the source map
	err = os.WriteFile(mapFile, sourceMap, 0644)
	return nil
}

func (tmpl *Template) writeGlobalScript(data map[string]interface{}) error {
	file, source, err := tmpl.backendScriptSource("__global.backend")
	if err != nil {
		return err
	}

	if source == nil {
		return nil
	}

	name := filepath.Base(file)
	ext := filepath.Ext(name)
	root, err := tmpl.local.DSL.PublicRoot(data)
	if err != nil {
		log.Error("WriteGlobalScript: Get the public root error: %s. use %s", err.Error(), tmpl.local.DSL.Public.Root)
		root = tmpl.local.DSL.Public.Root
	}
	target := filepath.Join(application.App.Root(), "public", root, fmt.Sprintf("__global%s", ext))
	dir := filepath.Dir(target)
	if exist, _ := os.Stat(dir); exist == nil {
		os.MkdirAll(dir, os.ModePerm)
	}
	return os.WriteFile(target, source, 0644)
}

// SyncAssets sync the assets
func (tmpl *Template) SyncAssets(option *core.BuildOption) error {

	// get source abs path
	sourceRoot := filepath.Join(tmpl.local.fs.Root(), tmpl.Root, "__assets")
	if exist, _ := os.Stat(sourceRoot); exist == nil {
		return nil
	}

	//get target abs path
	root, err := tmpl.local.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), tmpl.local.DSL.Public.Root)
		root = tmpl.local.DSL.Public.Root
	}
	targetRoot := filepath.Join(application.App.Root(), "public", root, "assets")

	if exist, _ := os.Stat(targetRoot); exist == nil {
		os.MkdirAll(targetRoot, os.ModePerm)
	}
	os.RemoveAll(targetRoot)

	return copyDirectory(sourceRoot, targetRoot)
}

func (tmpl *Template) getLocaleGlobal(name string) core.Locale {
	global := core.Locale{
		Keys:     map[string]string{},
		Messages: map[string]string{},
	}
	file := filepath.Join(tmpl.Root, "__locales", name, "__global.yml")
	exist, err := tmpl.local.fs.Exists(file)
	if err != nil {
		log.Error(`[SUI] Check the global locale file error: %s`, err.Error())
		return global
	}

	if !exist {
		return global
	}

	raw, err := tmpl.local.fs.ReadFile(file)
	if err != nil {
		log.Error(`[SUI] Read the global locale file error: %s`, err.Error())
		return global
	}

	err = yaml.Unmarshal(raw, &global)
	if err != nil {
		log.Error(`[SUI] Parse the global locale file error: %s`, err.Error())
		return global
	}

	return global
}

func (tmpl *Template) getLocale(name string, route string, pageOnly ...bool) core.Locale {
	file := filepath.Join(tmpl.Root, "__locales", name, fmt.Sprintf("%s.yml", route))
	global := tmpl.getLocaleGlobal(name)

	// Check the locale file
	exist, err := tmpl.local.fs.Exists(file)
	if err != nil {
		return global
	}

	if !exist {
		return global
	}

	locale := core.Locale{
		Name:           name,
		Keys:           map[string]string{},
		Messages:       map[string]string{},
		ScriptMessages: map[string]string{},
		Direction:      global.Direction,
		Timezone:       global.Timezone,
		Formatter:      global.Formatter,
	}

	raw, err := tmpl.local.fs.ReadFile(file)
	if err != nil {
		log.Error(`[SUI] Read the locale file error: %s`, err.Error())
		return global
	}

	err = yaml.Unmarshal(raw, &locale)
	if err != nil {
		log.Error(`[SUI] Parse the locale file error: %s`, err.Error())
		return global
	}

	if len(pageOnly) == 0 || !pageOnly[0] {

		// Merge the global
		for key, message := range global.Keys {
			if _, ok := locale.Keys[key]; !ok {
				locale.Keys[key] = message
			}
		}

		for key, message := range global.Messages {
			if _, ok := locale.Messages[key]; !ok {
				locale.Messages[key] = message
			}
		}
	}

	return locale
}

// Build is the struct for the public
func (page *Page) Build(globalCtx *core.GlobalBuildContext, option *core.BuildOption) ([]string, error) {

	ctx := core.NewBuildContext(globalCtx)
	var err error = nil
	root := option.PublicRoot
	if root == "" {
		root, err = page.tmpl.local.DSL.PublicRoot(option.Data)
		if err != nil {
			log.Error("Get the public root error: %s. use %s", err.Error(), page.tmpl.local.DSL.Public.Root)
		}
	}

	if option.AssetRoot == "" {
		option.AssetRoot = filepath.Join(root, "assets")
	}
	page.Root = root

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

	// Save the locale files
	err = page.writeLocaleFiles(ctx, option.Data)
	if err != nil {
		return warnings, err
	}

	// Save the config file
	err = page.writeConfig([]byte(config), option.Data)
	if err != nil {
		return warnings, err
	}

	// Jit Components
	if globalCtx == nil {
		jitComponents, err := page.tmpl.GlobRoutes(ctx.GetJitComponents(), true)
		if err != nil {
			return warnings, fmt.Errorf("Glob the jit components error: %s", err.Error())
		}

		for _, route := range jitComponents {
			p := page.tmpl.loaded[route]
			if p == nil {
				p, err = page.tmpl.Page(route)
				if err != nil {
					err = multierror.Append(err)
					continue
				}
			}

			messages, err := p.BuildAsComponent(globalCtx, option)
			if err != nil {
				err = multierror.Append(err)
			}
			if len(messages) > 0 {
				warnings = append(warnings, messages...)
			}
		}

		if err != nil {
			return warnings, fmt.Errorf("Build the page %s error: %s", page.Route, err.Error())
		}
	}

	return warnings, nil

}

// BuildAsComponent build the page as component
func (page *Page) BuildAsComponent(globalCtx *core.GlobalBuildContext, option *core.BuildOption) ([]string, error) {

	warnings := []string{}
	ctx := core.NewBuildContext(globalCtx)
	if option.AssetRoot == "" {
		root, err := page.tmpl.local.DSL.PublicRoot(option.Data)
		if err != nil {
			log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), page.tmpl.local.DSL.Public.Root)
			root = page.tmpl.local.DSL.Public.Root
		}
		option.AssetRoot = filepath.Join(root, "assets")
	}

	html, messages, err := page.Page.CompileAsComponent(ctx, option)
	if err != nil {
		return warnings, err
	}

	if len(messages) > 0 {
		warnings = append(warnings, messages...)
	}

	// Save the html
	err = page.writeJitHTML([]byte(html), option.Data)
	if err != nil {
		return warnings, err
	}

	// Save the locale files
	err = page.writeLocaleFiles(ctx, option.Data)
	if err != nil {
		return warnings, err
	}

	// Jit Components
	if globalCtx == nil {
		jitComponents, err := page.tmpl.GlobRoutes(ctx.GetJitComponents(), true)
		if err != nil {
			return warnings, err
		}

		for _, route := range jitComponents {
			var err error
			p := page.tmpl.loaded[route]
			if p == nil {
				p, err = page.tmpl.Page(route)
				if err != nil {
					err = multierror.Append(err)
					continue
				}
			}

			messages, err := p.BuildAsComponent(globalCtx, option)
			if err != nil {
				err = multierror.Append(err)
			}

			if len(messages) > 0 {
				warnings = append(warnings, messages...)
			}
		}
	}

	return warnings, err
}

// Trans the page
func (page *Page) Trans(globalCtx *core.GlobalBuildContext, option *core.BuildOption) ([]string, error) {
	warnings := []string{}
	ctx := core.NewBuildContext(globalCtx)

	_, _, messages, err := page.Page.Compile(ctx, option)
	if err != nil {
		return warnings, err
	}

	if len(messages) > 0 {
		warnings = append(warnings, messages...)
	}

	// Tranlate the locale files
	err = page.writeLocaleSource(ctx, option)
	return warnings, err
}

func (page *Page) publicFile(data map[string]interface{}) string {
	root, err := page.tmpl.local.DSL.PublicRoot(data)
	if err != nil {
		log.Error("publicFile: Get the public root error: %s. use %s", err.Error(), page.tmpl.local.DSL.Public.Root)
		root = page.tmpl.local.DSL.Public.Root
	}
	return filepath.Join("/", "public", root, page.Route)
}

func (page *Page) localeFiles(data map[string]interface{}) map[string]string {
	root, err := page.tmpl.local.DSL.PublicRoot(data)
	if err != nil {
		log.Error("publicFile: Get the public root error: %s. use %s", err.Error(), page.tmpl.local.DSL.Public.Root)
		root = page.tmpl.local.DSL.Public.Root
	}

	roots := map[string]string{}
	locales := page.tmpl.Locales()
	for _, locale := range locales {
		if locale.Default {
			continue
		}
		target := filepath.Join("/", "public", root, ".locales", locale.Value, fmt.Sprintf("%s.yml", page.Route))
		roots[locale.Value] = target
	}
	return roots
}

func (page *Page) writeLocaleSource(ctx *core.BuildContext, option *core.BuildOption) error {

	locales := page.tmpl.Locales()
	translations := ctx.GetTranslations()

	if option.Locales != nil && len(option.Locales) > 0 {
		locales = []core.SelectOption{}
		for _, lc := range option.Locales {
			label := language.Make(lc).String()
			locales = append(locales, core.SelectOption{Value: lc, Label: label})
		}
	}

	prefix := core.TranslationKeyPrefix(page.Route)
	for _, lc := range locales {
		if lc.Default {
			continue
		}

		locale := page.tmpl.getLocale(lc.Value, page.Route, true)
		locale.MergeTranslations(translations, prefix)

		// Call the hook
		var keys any = locale.Keys
		var messages any = locale.Messages
		if page.tmpl.Translator != "" {
			p, err := process.Of(page.tmpl.Translator, lc.Value, locale, page.Route, page.TemplateID)
			if err != nil {
				return err
			}

			res, err := p.Exec()
			if err != nil {
				return err
			}

			pres, ok := res.(map[string]interface{})
			if !ok {
				return fmt.Errorf("The translator %s should return a locale", page.tmpl.Translator)
			}

			keys = pres["keys"]
			messages = pres["messages"]
		}

		if keys == nil && messages == nil {
			return nil
		}

		// Save to file
		file := filepath.Join(page.tmpl.Root, "__locales", lc.Value, fmt.Sprintf("%s.yml", page.Route))
		content, err := yaml.Marshal(map[string]interface{}{
			"keys":     keys,
			"messages": messages,
		})
		if err != nil {
			return err
		}

		_, err = page.tmpl.local.fs.WriteFile(file, content, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (page *Page) writeLocaleFiles(ctx *core.BuildContext, data map[string]interface{}) error {

	if ctx == nil {
		return nil
	}

	components := ctx.GetComponents()
	translations := ctx.GetTranslations()
	if len(translations) == 0 && len(components) == 0 {
		return nil
	}
	prefix := core.TranslationKeyPrefix(page.Route)
	files := page.localeFiles(data)
	for name, file := range files {
		locale := page.tmpl.getLocale(name, page.Route)
		locale.MergeTranslations(translations, prefix)

		// Merge the components locale
		for _, route := range components {
			compLocale := page.tmpl.getLocale(name, route, true)
			compLocale.ParseKeys()
			locale.Merge(compLocale)
		}

		// Remove messages
		locale.Messages = map[string]string{}
		raw, err := yaml.Marshal(locale)
		if err != nil {
			log.Error(`[SUI] Marshal the locale file error: %s`, err.Error())
			return err
		}

		fileAbs := filepath.Join(application.App.Root(), file)
		dir := filepath.Dir(fileAbs)
		if exist, _ := os.Stat(dir); exist == nil {
			os.MkdirAll(dir, os.ModePerm)
		}

		err = os.WriteFile(fileAbs, raw, 0644)
		if err != nil {
			log.Error(`[SUI] Write the locale file error: %s`, err.Error())
			return err
		}
	}

	return nil
}

func (page *Page) backendScriptSource() (string, []byte, error) {
	backendFile := filepath.Join(page.Path, fmt.Sprintf("%s.backend.ts", page.Name))
	if exist, _ := page.tmpl.local.fs.Exists(backendFile); !exist {
		backendFile = filepath.Join(page.Path, fmt.Sprintf("%s.backend.js", page.Name))
	}

	if exist, _ := page.tmpl.local.fs.Exists(backendFile); !exist {
		return "", nil, nil
	}

	source, err := page.tmpl.local.fs.ReadFile(backendFile)
	if err != nil {
		return "", nil, err
	}

	source = []byte(fmt.Sprintf("%s\n%s", source, core.BackendScript(page.Route)))
	return backendFile, source, nil
}

func (page *Page) loadBackendScript() error {
	file, source, err := page.backendScriptSource()
	if err != nil {
		return err
	}

	if source == nil {
		return nil
	}
	approot := page.tmpl.local.AppRoot()
	file = filepath.Join(approot, file)
	script, err := v8.MakeScript(source, file, 5*time.Second)
	if err != nil {
		return err
	}
	page.Script = &core.Script{Script: script}
	return nil
}

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

// writeHTMLTo write the html to file
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

// writeHTMLTo write the html to file
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

// writeHTMLTo write the html to file
func (page *Page) writeJitHTML(html []byte, data map[string]interface{}) error {
	htmlFile := fmt.Sprintf("%s.jit", page.publicFile(data))
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
