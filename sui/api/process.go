package api

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/sui/core"
)

func init() {
	process.RegisterGroup("sui", map[string]process.Handler{
		"setting": Setting,

		"render": Render, // do not use this in script or flow, this is an internal method. Use the template.render instead
		"run":    Run,    // do not use this in script or flow, this is an internal method. Use the template.run instead

		"template.get":         TemplateGet,
		"template.find":        TemplateFind,
		"template.asset":       TemplateAsset,
		"template.assetupload": TemplateAssetUpload,
		"template.render":      TemplateRender,
		// "template.run":         TemplateRun,

		"locale.get": LocaleGet,
		"theme.get":  ThemeGet,

		"block.get":    BlockGet,
		"block.find":   BlockFind,
		"block.Media":  BlockMedia,
		"block.export": BlockExport,

		"component.get":  ComponentGet,
		"component.find": ComponentFind,

		"page.tree":      PageTree,
		"page.get":       PageGet,
		"page.save":      PageSave,
		"page.savetemp":  PageSaveTemp,
		"page.create":    PageCreate,
		"page.duplicate": PageDuplicate,
		"page.rename":    PageRename,
		"page.remove":    PageRemove,
		"page.exist":     PageExist,
		"page.asset":     PageAsset,

		"editor.render":              EditorRender,
		"editor.source":              EditorSource,
		"editor.renderaftersavetemp": EditorRenderAfterSaveTemp,
		"editor.sourceaftersavetemp": EditorSourceAfterSaveTemp,

		"media.search": MediaSearch,

		"preview.render": PreviewRender,

		"build.all":  BuildAll,
		"build.page": BuildPage,

		"trans.all":  TransAll,
		"trans.page": TransPage,

		"sync.assetfile": SyncAssetFile, // Will be deprecated or change in the future

		// Will be deprecated or change in the future
		"types.QueryParam": TypesQueryParam,
	})
}

// TypesQueryParam handle the get Template request
func TypesQueryParam(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	switch v := process.Args[0].(type) {

	case url.Values:
		return types.URLToQueryParam(v)

	case map[string][]string:
		return types.URLToQueryParam(v)

	case map[string]interface{}:
		values := url.Values{}
		for key, value := range v {
			switch val := value.(type) {
			case []string:
				for _, v := range val {
					values.Add(key, v)
				}

			case []interface{}:
				for _, v := range val {
					values.Add(key, fmt.Sprintf("%v", v))
				}

			default:
				values.Set(key, fmt.Sprintf("%v", value))
			}
		}
		return types.URLToQueryParam(values)
	}

	v, _ := types.AnyToQueryParam(process.Args[0])
	return v
}

// Setting handle the get Template request
func Setting(process *process.Process) interface{} {
	sui := get(process)
	setting, err := sui.Setting()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return setting
}

// TemplateGet handle the get Template request
func TemplateGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	sui := get(process)
	templates, err := sui.GetTemplates()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return templates
}

// TemplateFind handle the find Template request
func TemplateFind(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	sui := get(process)
	tmpl, err := sui.GetTemplate(process.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return tmpl
}

// TemplateAsset handle the find Template request
func TemplateAsset(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	sui := get(process)
	tmpl, err := sui.GetTemplate(process.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	w := process.ArgsInt(3, 0)
	h := process.ArgsInt(4, 0)
	asset, err := tmpl.Asset(process.ArgsString(2), uint(w), uint(h))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	return map[string]interface{}{
		"content": asset.Content,
		"type":    asset.Type,
	}
}

// TemplateAssetUpload handle the find Template request
func TemplateAssetUpload(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	sui := get(process)
	tmpl, err := sui.GetTemplate(process.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	switch v := process.Args[2].(type) {
	case *gin.Context:
		file, err := v.FormFile("file")
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		reader, err := file.Open()
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}
		defer reader.Close()

		path, err := tmpl.AssetUpload(reader, file.Filename)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		url := v.PostForm("url")
		fileurl := fmt.Sprintf("%s/%s", url, path)
		// time.Sleep(10 * time.Second)
		return map[string]interface{}{
			"data":   []interface{}{fileurl},
			"header": file.Header,
		}

	case string:
		data, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		name := process.ArgsString(3, "file.png")
		path, err := tmpl.AssetUpload(strings.NewReader(string(data)), name)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		return path

	default:
		exception.New("the file is required", 400).Throw()
		return nil
	}
}

// MediaSearch handle the find Template request
func MediaSearch(process *process.Process) interface{} {

	process.ValidateArgNums(2)
	sui := get(process)
	driver := process.ArgsString(1)

	query := url.Values{}
	if process.NumOfArgs() > 2 {
		switch v := process.Args[2].(type) {
		case map[string]string:
			for key, value := range v {
				query.Set(key, value)
			}
			break

		case map[string]interface{}:
			for key, value := range v {
				query.Set(key, fmt.Sprintf("%v", value))
			}
			break

		case map[string][]string:
			query = v
			break

		case url.Values:
			query = v
			break
		}
	}

	var err error
	page := 1
	if v := query.Get("page"); v != "" {
		page, err = strconv.Atoi(v)
		if err != nil {
			exception.New(err.Error(), 400).Throw()
		}
		query.Del("page")
	}

	pageSize := 20
	if v := query.Get("pagesize"); v != "" {
		pageSize, err = strconv.Atoi(v)
		if err != nil {
			exception.New(err.Error(), 400).Throw()
		}
		query.Del("pagesize")
	}

	switch driver {
	case "local":
		templateID := query.Get("template")
		if templateID == "" {
			exception.New("the template is required", 400).Throw()
		}

		tmpl, err := sui.GetTemplate(templateID)
		if err != nil {
			exception.New(err.Error(), 400).Throw()
		}

		res, err := tmpl.MediaSearch(query, page, pageSize)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		return res

	default:
		exception.New("the driver %s does not exist", 404, driver).Throw()
		return nil

	}
}

// LocaleGet handle the find Template request
func LocaleGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	sui := get(process)
	template, err := sui.GetTemplate(process.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	locals := template.Locales()
	if locals == nil {
		return []core.SelectOption{}
	}
	return locals
}

// ThemeGet handle the find Template request
func ThemeGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	sui := get(process)
	template, err := sui.GetTemplate(process.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return template.Themes()
}

// BlockGet handle the find Template request
func BlockGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	sui := get(process)
	templateID := process.ArgsString(1)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	blocks, err := tmpl.Blocks()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return blocks
}

// BlockExport handle the find Template request
func BlockExport(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	sui := get(process)
	templateID := process.ArgsString(1)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	items, err := tmpl.BlockLayoutItems()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return items
}

// BlockMedia handle the find Template request
func BlockMedia(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	sui := get(process)
	templateID := process.ArgsString(1)
	blockID := strings.TrimRight(process.ArgsString(2), ".js")

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	asset, err := tmpl.BlockMedia(blockID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return map[string]interface{}{
		"content": asset.Content,
		"type":    asset.Type,
	}
}

// BlockFind handle the find Template request
func BlockFind(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	sui := get(process)
	templateID := process.ArgsString(1)
	blockID := strings.TrimRight(process.ArgsString(2), ".js")

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	block, err := tmpl.Block(blockID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return block.Source()
}

// ComponentGet handle the find Template request
func ComponentGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	sui := get(process)
	templateID := process.ArgsString(1)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	components, err := tmpl.Components()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return components
}

// ComponentFind handle the find Template request
func ComponentFind(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	sui := get(process)
	templateID := process.ArgsString(1)
	componentID := strings.TrimRight(process.ArgsString(2), ".js")

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	component, err := tmpl.Component(componentID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return component.Source()
}

// PageTree handle the find Template request
func PageTree(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	sui := get(process)
	templateID := process.ArgsString(1)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	route := route(process, 2)
	tree, err := tmpl.PageTree(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return tree
}

// PageGet handle the find Template request
func PageGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	sui := get(process)
	templateID := process.ArgsString(1)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	tree, err := tmpl.Pages()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return tree
}

// PageSave handle the find Template request
func PageSave(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	var page core.IPage
	if tmpl.PageExist(route) {
		page, err = tmpl.Page(route)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}
	} else {
		page, err = tmpl.CreateEmptyPage(route, nil)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}
	}

	source, err := getSource(process)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if source == nil {
		exception.New("the source is required", 400).Throw()
	}

	err = page.Save(source)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return nil
}

// PageSaveTemp handle the find Template request
func PageSaveTemp(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	var page core.IPage
	if tmpl.PageExist(route) {
		page, err = tmpl.Page(route)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}
	} else {
		page, err = tmpl.CreateEmptyPage(route, nil)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}
	}

	source, err := getSource(process)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if source == nil {
		exception.New("the source is required", 400).Throw()
	}

	if source.UID == "" {
		exception.New("the source.uid is required", 400).Throw()
	}

	err = page.SaveTemp(source)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return nil
}

// PageCreate handle the find Template request
func PageCreate(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	sui := get(process)
	templateID := process.ArgsString(1)
	route := process.ArgsString(2)
	payload := process.ArgsMap(4, map[string]interface{}{})

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// Get the route from payload
	if v, ok := payload["route"].(string); ok {
		route = v
	}

	title := route
	if v, ok := payload["title"].(string); ok {
		title = v
	}
	setting := &core.PageSetting{Title: title}
	page, err := tmpl.CreateEmptyPage(route, setting)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if len(process.Args) <= 3 {
		return nil
	}

	source, err := getSource(process)
	if err != nil {
		return nil
	}

	if source == nil {
		return nil
	}

	err = page.Save(source)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	Reload()
	return nil
}

// PageRename handle the find Template request
func PageRename(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	sui := get(process)
	templateID := process.ArgsString(1)
	copyfrom := process.ArgsString(2)
	payload := process.ArgsMap(3, map[string]interface{}{})

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	page, err := tmpl.Page(copyfrom)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// Get the route from payload
	route, ok := payload["route"].(string)
	if !ok {
		exception.New("the route is required", 400).Throw()
	}

	// Rename
	_, err = page.SaveAs(route, nil)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// delete the old page
	err = tmpl.RemovePage(copyfrom)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	Reload()
	return nil
}

// PageDuplicate handle the find Template request
func PageDuplicate(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	sui := get(process)
	templateID := process.ArgsString(1)
	copyfrom := process.ArgsString(2)
	payload := process.ArgsMap(3, map[string]interface{}{})

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	page, err := tmpl.Page(copyfrom)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// Get the route from payload
	route, ok := payload["route"].(string)
	if !ok {
		exception.New("the route is required", 400).Throw()
	}

	title := route
	if v, ok := payload["title"].(string); ok {
		title = v
	}

	// Page Save as
	setting := &core.PageSetting{Title: title}
	_, err = page.SaveAs(route, setting)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	Reload()
	return nil
}

// PageRemove handle the find Template request
func PageRemove(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if !tmpl.PageExist(route) {
		exception.New("page does not exists!", 400).Throw()
	}

	err = tmpl.RemovePage(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	Reload()
	return nil
}

// PageExist handle the find Template request
func PageExist(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return tmpl.PageExist(route)
}

// PageAsset handle the find Template request
func PageAsset(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	sui := get(process)
	templateID := process.ArgsString(1)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	file := process.ArgsString(2)
	page, err := tmpl.GetPageFromAsset(file)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	var asset *core.Asset

	switch filepath.Ext(file) {
	case ".css":
		asset, err = page.AssetStyle()
		if err != nil {
			exception.New(err.Error(), 400).Throw()
		}
		break

	case ".js", ".ts":
		asset, err = page.AssetScript()
		if err != nil {
			exception.New(err.Error(), 400).Throw()
		}
		break

	default:
		exception.New("does not support the %s file", 400, filepath.Ext(file)).Throw()
	}

	return map[string]interface{}{
		"content": asset.Content,
		"type":    asset.Type,
	}
}

// EditorRender handle the render page request
func EditorRender(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	page, err := tmpl.Page(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	res, err := page.EditorRender()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

// EditorRenderAfterSaveTemp handle the render page request
func EditorRenderAfterSaveTemp(process *process.Process) interface{} {
	process.ValidateArgNums(5)
	PageSaveTemp(process)
	args := append([]interface{}{}, process.Args[:3]...)
	args = append(args, process.Args[4:]...)
	process.Args = args
	return EditorRender(process)
}

// EditorSource handle the render page request
func EditorSource(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)
	kind := process.ArgsString(3)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	page, err := tmpl.Page(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	switch kind {

	case "page":
		return page.EditorPageSource()

	case "style":
		return page.EditorStyleSource()

	case "script":
		return page.EditorScriptSource()

	case "data":
		return page.EditorDataSource()

	default:
		exception.New("the %s source does not exist", 404, kind).Throw()
		return nil
	}
}

// EditorSourceAfterSaveTemp handle the render page request
func EditorSourceAfterSaveTemp(process *process.Process) interface{} {
	process.ValidateArgNums(5)
	PageSaveTemp(process)

	args := append([]interface{}{}, process.Args[:3]...)
	args = append(args, process.Args[4:]...)
	process.Args = args
	return EditorSource(process)
}

// PreviewRender handle the render page request
func PreviewRender(process *process.Process) interface{} {

	process.ValidateArgNums(3)
	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)
	referer := process.ArgsString(3, "")

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	page, err := tmpl.Page(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// Request data
	html, err := page.PreviewRender(referer)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return html
}

// SyncAssetFile  handle the render page request
func SyncAssetFile(process *process.Process) interface{} {

	process.ValidateArgNums(4)
	sui := get(process)
	templateID := process.ArgsString(1)
	filename := process.ArgsString(2)

	option := process.ArgsMap(3, map[string]interface{}{})
	ssr := true
	if v, ok := option["ssr"].(bool); ok {
		ssr = v
	}

	assetRoot := ""
	if v, ok := option["asset_root"].(string); ok {
		assetRoot = v
	}

	data := map[string]interface{}{}
	if v, ok := option["data"].(map[string]interface{}); ok {
		data = v
	}

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	err = tmpl.SyncAssetFile(filename, &core.BuildOption{SSR: ssr, AssetRoot: assetRoot, Data: data})
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return nil
}

// BuildAll handle the render page request
func BuildAll(process *process.Process) interface{} {

	process.ValidateArgNums(3)
	sui := get(process)
	templateID := process.ArgsString(1)

	option := process.ArgsMap(2, map[string]interface{}{})
	ssr := true
	if v, ok := option["ssr"].(bool); ok {
		ssr = v
	}

	assetRoot := ""
	if v, ok := option["asset_root"].(string); ok {
		assetRoot = v
	}

	data := map[string]interface{}{}
	if v, ok := option["data"].(map[string]interface{}); ok {
		data = v
	}

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	warnings, err := tmpl.Build(&core.BuildOption{SSR: ssr, AssetRoot: assetRoot, Data: data})
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if warnings != nil && len(warnings) > 0 {
		return warnings
	}
	return nil
}

// BuildPage handle the render page request
func BuildPage(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)
	option := process.ArgsMap(3, map[string]interface{}{})
	ssr := true
	if v, ok := option["ssr"].(bool); ok {
		ssr = v
	}

	assetRoot := ""
	if v, ok := option["asset_root"].(string); ok {
		assetRoot = v
	}

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	page, err := tmpl.Page(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	err = page.Load()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	data := process.ArgsMap(5, map[string]interface{}{})
	warnings, err := page.Build(nil, &core.BuildOption{SSR: ssr, AssetRoot: assetRoot, Data: data})
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	if warnings != nil && len(warnings) > 0 {
		return warnings
	}

	return nil
}

// TransAll handle the render page request
func TransAll(process *process.Process) interface{} {

	process.ValidateArgNums(3)
	sui := get(process)
	templateID := process.ArgsString(1)

	option := process.ArgsMap(2, map[string]interface{}{})
	ssr := true
	if v, ok := option["ssr"].(bool); ok {
		ssr = v
	}

	assetRoot := ""
	if v, ok := option["asset_root"].(string); ok {
		assetRoot = v
	}

	data := map[string]interface{}{}
	if v, ok := option["data"].(map[string]interface{}); ok {
		data = v
	}

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	warnings, err := tmpl.Trans(&core.BuildOption{SSR: ssr, AssetRoot: assetRoot, Data: data})
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if warnings != nil && len(warnings) > 0 {
		return warnings
	}
	return nil
}

// TransPage handle the render page request
func TransPage(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	sui := get(process)
	templateID := process.ArgsString(1)
	route := route(process, 2)
	option := process.ArgsMap(3, map[string]interface{}{})
	ssr := true
	if v, ok := option["ssr"].(bool); ok {
		ssr = v
	}

	assetRoot := ""
	if v, ok := option["asset_root"].(string); ok {
		assetRoot = v
	}

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	page, err := tmpl.Page(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	err = page.Load()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	data := process.ArgsMap(5, map[string]interface{}{})
	warnings, err := page.Trans(nil, &core.BuildOption{SSR: ssr, AssetRoot: assetRoot, Data: data})
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	if warnings != nil && len(warnings) > 0 {
		return warnings
	}

	return nil
}

// get the sui
func get(process *process.Process) core.SUI {
	sui, has := core.SUIs[process.ArgsString(0)]
	if !has {
		exception.New("the sui %s does not exist", 404, process.ID).Throw()
	}
	sui.WithSid(process.Sid)
	return sui
}

func route(process *process.Process, i int) string {
	route := process.ArgsString(i)
	if route == "" {
		route = "/index"
	}

	if route[0] != '/' {
		route = "/" + route
	}
	return route
}

func getSource(process *process.Process) (*core.RequestSource, error) {

	if process.NumOfArgs() < 4 {
		return nil, nil
	}

	switch v := process.Args[3].(type) {
	case *core.RequestSource:
		return v, nil

	case *gin.Context:
		source := core.RequestSource{UID: v.GetHeader("Yao-Builder-Uid")}
		err := v.ShouldBind(&source)
		if err != nil {
			return nil, fmt.Errorf("Bind: %s", err.Error())
		}
		return &source, nil

	case string:
		if process.NumOfArgs() > 4 {
			uid := process.ArgsString(3)
			payload, err := jsoniter.Marshal(process.Args[4])
			if err != nil {
				return nil, err
			}
			source := core.RequestSource{UID: uid}
			err = jsoniter.Unmarshal(payload, &source)
			if err != nil {
				return nil, err
			}
			return &source, nil
		}

		source := core.RequestSource{}
		err := jsoniter.UnmarshalFromString(process.ArgsString(3), &source)
		if err != nil {
			return nil, err
		}
		return &source, nil

	default:

		payload, err := jsoniter.Marshal(process.Args[3])
		if err != nil {
			return nil, err
		}

		source := core.RequestSource{}
		err = jsoniter.Unmarshal(payload, &source)
		if err != nil {
			return nil, err
		}
		return &source, nil
	}
}
