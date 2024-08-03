package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
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

	html, err := r.renderHTML(c, name, c.HTML, core.Data(data))
	if err != nil {
		return fmt.Sprintf("<span class='sui-render-error'> %s </span>", err.Error())
	}

	return html
}

func (r *Request) renderHTML(c *core.Cache, name string, html string, data core.Data) (string, error) {

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
	html, err = sel.Html()
	if err != nil {
		return "", fmt.Errorf("Html error: %w", err)
	}

	return html, nil
}
