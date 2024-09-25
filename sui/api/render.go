package api

import (
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/sui/core"
)

// Render the frontend page
func Render(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	ctx, ok := process.Args[0].(*gin.Context)
	if !ok {
		return "The context is required"
	}

	ctx.Header("Content-Type", "text/html; charset=utf-8")
	route := process.ArgsString(1)
	payload := process.ArgsMap(2)

	if route == "" {
		return "The route is required"
	}

	if payload["name"] == nil {
		return "The render name is required"
	}

	ctx.Request.URL.Path = route
	r, _, err := NewRequestContext(ctx)
	if err != nil {
		return fmt.Sprintf("<span class='sui-render-error'> %s </span>", err.Error())
	}

	var c *core.Cache = nil
	if !r.Request.DisableCache() {
		c = core.GetCache(r.File)
	}

	if c == nil {
		c, _, err = r.MakeCache()
		if err != nil {
			return fmt.Sprintf("<span class='sui-render-error'> %s </span>", err.Error())
		}
	}

	if c == nil {
		return fmt.Sprintf("<span class='sui-render-error'> Cache not found </span>")
	}

	// Guard the page
	code, err := r.Guard(c)
	if err != nil {
		return fmt.Sprintf("<span class='sui-render-error'> %v %s </span>", code, err.Error())
	}

	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		return fmt.Sprintf("<span class='sui-render-error'> Data not found </span>")
	}

	name, ok := payload["name"].(string)
	if !ok {
		return fmt.Sprintf("<span class='sui-render-error'> Name not found </span>")
	}

	// Get the render option
	option := map[string]interface{}{}
	if v, ok := payload["option"].(map[string]interface{}); ok {
		option = v
	}

	// Get the component name (optional)
	comp := ""
	if v, ok := option["component"].(string); ok {
		comp = v
	}

	html, err := r.renderHTML(c, name, comp, c.HTML, core.Data(data))
	if err != nil {
		return fmt.Sprintf("<span class='sui-render-error'> %s </span>", err.Error())
	}

	return html
}

func (r *Request) renderHTML(c *core.Cache, name string, comp string, html string, data core.Data) (string, error) {

	doc, err := core.NewDocument([]byte(html))
	if err != nil {
		return "", fmt.Errorf("Document error: %w", err)
	}

	sel := doc.Find(fmt.Sprintf("[s\\:render='%s']", name))
	if sel.Length() == 0 {
		return "", fmt.Errorf("Render %s not found", name)
	}

	// Set the page request data
	option := core.ParserOption{
		Theme:        r.Request.Theme,
		Locale:       r.Request.Locale,
		Debug:        r.Request.DebugMode(),
		DisableCache: r.Request.DisableCache(),
		Route:        r.Request.URL.Path,
		Root:         c.Root,
		Script:       c.Script,
		Imports:      c.Imports,
		Request:      r.Request,
	}

	// Parse the template
	parser := core.NewTemplateParser(data, &option)
	err = parser.RenderSelection(sel)
	if err != nil {
		return "", fmt.Errorf("Parser error: %w", err)
	}

	sel.Find("[sui-hide]").Remove()
	parser.Tidy(sel)

	// **** warning ****
	// Fix s:event-cn="__page" to s:event-cn="component" for component
	// The following code is the temporary solution for the component event
	// will be removed in the sui v2 release
	if comp != "" {
		sel.Find("[s\\:event-cn='__page']").SetAttr("s:event-cn", comp)
	}

	html, err = sel.Html()
	if err != nil {
		return "", fmt.Errorf("Html error: %w", err)
	}

	return html, nil
}

// TemplateRender render the template asset
func TemplateRender(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	sui := get(process)
	tmpl, err := sui.GetTemplate(process.ArgsString(1))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	opt := process.ArgsMap(4, map[string]interface{}{})
	buildOptionData, ok := opt["data"].(map[string]interface{})
	if !ok {
		buildOptionData = map[string]interface{}{}
	}

	root, err := sui.PublicRoot(buildOptionData)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	assetRoot := filepath.Join(root, "assets")

	source := process.ArgsString(2)
	page := tmpl.CreatePage(source)
	route := page.Get().Route
	globalCtx := core.NewGlobalBuildContext()
	suicode, _, err := page.Get().CompileAsComponent(core.NewBuildContext(globalCtx), &core.BuildOption{
		PublicRoot:     root,
		AssetRoot:      assetRoot,
		IgnoreDocument: true,
		Data:           buildOptionData,
	})
	if err != nil {
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	doc, err := core.NewDocument([]byte(suicode))
	if err != nil {
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	var imports map[string]string
	importsSel := doc.Find("script[name=imports]")
	if importsSel != nil && importsSel.Length() > 0 {
		importsRaw := importsSel.Text()
		importsSel.Remove()
		err := jsoniter.UnmarshalFromString(importsRaw, &imports)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
			return nil
		}
	}

	r := core.Request{Theme: opt["theme"], Locale: opt["locale"], Sid: process.Sid}
	if process.NumOfArgs() > 5 {

		raw, err := jsoniter.Marshal(process.Args[5])
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		err = jsoniter.Unmarshal(raw, &r)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}

		if r.Theme == "" {
			r.Theme = opt["theme"]
		}

		if r.Locale == "" {
			r.Locale = opt["locale"]
		}
	}

	data := process.ArgsMap(3)
	option := core.ParserOption{
		Theme:        opt["theme"],
		Locale:       opt["locale"],
		Debug:        false,
		DisableCache: true,
		Route:        route,
		Root:         root,
		Script:       nil,
		Imports:      imports,
		Request:      &r,
	}

	parser := core.NewTemplateParser(core.Data(data), &option)
	sel := doc.Find("body")
	err = parser.RenderSelection(sel)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	sel.Find("[sui-hide]").Remove()
	parser.Tidy(sel)
	html, err := sel.Html()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return html
}
