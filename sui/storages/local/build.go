package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
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
			err = multierror.Append(fmt.Errorf("The page %s is not loaded", route))
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

// Build is the struct for the public
func (page *Page) Build(globalCtx *core.GlobalBuildContext, option *core.BuildOption) ([]string, error) {

	ctx := core.NewBuildContext(globalCtx)
	if option.AssetRoot == "" {
		root, err := page.tmpl.local.DSL.PublicRoot(option.Data)
		if err != nil {
			log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), page.tmpl.local.DSL.Public.Root)
			root = page.tmpl.local.DSL.Public.Root
		}
		option.AssetRoot = filepath.Join(root, "assets")
	}

	html, warnings, err := page.Page.Compile(ctx, option)
	if err != nil {
		return warnings, err
	}

	// Save the html
	err = page.writeHTML([]byte(html), option.Data)
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

	_, messages, err := page.Page.CompileAsComponent(ctx, option)
	if err != nil {
		return warnings, err
	}

	if len(messages) > 0 {
		warnings = append(warnings, messages...)
	}

	// Tranlate the locale files
	err = page.writeLocaleSource(ctx)
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

func (page *Page) localeGlobal(name string) core.Locale {
	global := core.Locale{
		Keys:     map[string]string{},
		Messages: map[string]string{},
	}
	file := filepath.Join(page.tmpl.Root, "__locales", name, "__global.yml")
	exist, err := page.tmpl.local.fs.Exists(file)
	if err != nil {
		log.Error(`[SUI] Check the global locale file error: %s`, err.Error())
		return global
	}

	if !exist {
		return global
	}

	raw, err := page.tmpl.local.fs.ReadFile(file)
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

func (page *Page) locale(name string, pageOnly ...bool) core.Locale {
	file := filepath.Join(page.tmpl.Root, "__locales", name, fmt.Sprintf("%s.yml", page.Route))
	global := page.localeGlobal(name)

	// Check the locale file
	exist, err := page.tmpl.local.fs.Exists(file)
	if err != nil {
		return global
	}

	if !exist {
		return global
	}

	locale := core.Locale{
		Keys:     map[string]string{},
		Messages: map[string]string{},
		Date:     global.Date,
		Currency: global.Currency,
		Number:   global.Number,
	}
	raw, err := page.tmpl.local.fs.ReadFile(file)
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

func (page *Page) writeLocaleSource(ctx *core.BuildContext) error {

	locales := page.tmpl.Locales()
	translations := ctx.GetTranslations()
	for _, lc := range locales {
		if lc.Default {
			continue
		}

		locale := page.locale(lc.Value, true)
		for _, t := range translations {
			message := t.Message
			// Match the key
			if _, has := locale.Messages[message]; has {
				message = locale.Messages[message]
			}
			locale.Keys[t.Key] = message
			msg, has := locale.Messages[t.Message]
			if has && msg != t.Message {
				continue
			}
			locale.Messages[t.Message] = t.Message
		}

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

	translations := ctx.GetTranslations()
	if len(translations) == 0 {
		return nil
	}

	files := page.localeFiles(data)
	for name, file := range files {

		// Init Data
		keys := map[string]string{}
		messages := map[string]string{}
		for _, t := range translations {
			keys[t.Key] = t.Message
			messages[t.Message] = t.Message
		}

		locale := page.locale(name)
		for key := range keys {
			if _, has := locale.Keys[key]; has {
				keys[key] = locale.Keys[key]
			}

			if msgValue, has := locale.Messages[keys[key]]; has {
				keys[key] = msgValue
			}
		}

		for message := range messages {
			if _, has := locale.Messages[message]; has {
				messages[message] = locale.Messages[message]
			}
		}

		locale.Keys = keys
		locale.Messages = messages
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
	log.Trace("The page %s is removed", htmlFile)
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
	log.Trace("The page %s is removed", htmlFile)
	return nil
}
