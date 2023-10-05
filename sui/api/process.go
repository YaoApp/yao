package api

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/sui/core"
)

func init() {
	process.RegisterGroup("sui", map[string]process.Handler{
		"template.get":   TemplateGet,
		"template.find":  TemplateFind,
		"template.asset": TemplateAsset,

		"locale.get": LocaleGet,
		"theme.get":  ThemeGet,

		"block.get":  BlockGet,
		"block.find": BlockFind,

		"component.get":  ComponentGet,
		"component.find": ComponentFind,

		"page.tree":     PageTree,
		"page.get":      PageGet,
		"page.save":     PageSave,
		"page.savetemp": PageSaveTemp,
		"page.create":   PageCreate,
		"page.remove":   PageRemove,
		"page.exist":    PageExist,
		"page.asset":    PageAsset,

		"editor.render":              EditorRender,
		"editor.source":              EditorSource,
		"editor.renderaftersavetemp": EditorRenderAfterSaveTemp,
		"editor.sourceaftersavetemp": EditorSourceAfterSaveTemp,
	})
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

	asset, err := tmpl.Asset(process.ArgsString(2))
	if err != nil {
		exception.New(err.Error(), 404).Throw()
	}

	return map[string]interface{}{
		"content": asset.Content,
		"type":    asset.Type,
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

	return template.Locales()
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
		page, err = tmpl.CreatePage(route)
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
		page, err = tmpl.CreatePage(route)
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
	route := route(process, 2)

	tmpl, err := sui.GetTemplate(templateID)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	page, err := tmpl.CreatePage(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if len(process.Args) <= 3 {
		return nil
	}

	source, err := getSource(process)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if source == nil {
		return nil
	}

	err = page.Save(source)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
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

	err = tmpl.RemovePage(route)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
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

	// Request data
	urlQuery := url.Values{}
	if process.NumOfArgs() > 3 {
		if v, ok := process.Args[3].(url.Values); ok {
			urlQuery = v
		}
	}

	req := &core.Request{Method: "GET", Query: urlQuery}
	res, err := page.EditorRender(req)
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

// get the sui
func get(process *process.Process) core.SUI {
	sui, has := core.SUIs[process.ArgsString(0)]
	if !has {
		exception.New("the sui %s does not exist", 404, process.ID).Throw()
	}
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
