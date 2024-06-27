package local

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
	"gopkg.in/yaml.v3"
)

// Build the template
func (tmpl *Template) Build(option *core.BuildOption) error {
	var err error

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
		return err
	}

	// Build all pages
	pages, err := tmpl.Pages()
	if err != nil {
		return err
	}

	for _, page := range pages {
		perr := page.Load()
		if err != nil {
			err = multierror.Append(perr)
			continue
		}

		perr = page.Build(option)
		if perr != nil {
			err = multierror.Append(perr)
		}
	}

	return err
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
func (page *Page) Build(option *core.BuildOption) error {

	if option.AssetRoot == "" {
		root, err := page.tmpl.local.DSL.PublicRoot(option.Data)
		if err != nil {
			log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), page.tmpl.local.DSL.Public.Root)
			root = page.tmpl.local.DSL.Public.Root
		}
		option.AssetRoot = filepath.Join(root, "assets")
	}

	log.Trace("Build the page %s AssetRoot: %s", page.Route, option.AssetRoot)
	html, err := page.Page.Compile(option)
	if err != nil {
		return err
	}

	// Save the html
	err = page.writeHTML([]byte(html), option.Data)
	if err != nil {
		return err
	}

	// Save the locale files
	return page.writeLocaleFiles(option.Data)
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

func (page *Page) locale(name string) core.Locale {
	file := filepath.Join(page.tmpl.Root, "__locales", name, fmt.Sprintf("%s.yml", page.Route))
	global := page.localeGlobal(name)

	// Check the locale file
	exist, err := page.tmpl.local.fs.Exists(file)
	if err != nil {
		log.Error(`[SUI] Check the locale file error: %s`, err.Error())
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

	return locale
}

func (page *Page) writeLocaleFiles(data map[string]interface{}) error {

	//  No translations
	if len(page.Page.Translations) == 0 {
		return nil
	}

	keys := map[string]string{}
	messages := map[string]string{}
	for _, t := range page.Page.Translations {
		keys[t.Key] = t.Message
		messages[t.Message] = t.Message
	}

	files := page.localeFiles(data)
	for name, file := range files {
		locale := page.locale(name)
		for key := range keys {
			if _, has := locale.Keys[key]; has {
				keys[key] = locale.Keys[key]
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
