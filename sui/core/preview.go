package core

import (
	"fmt"
)

// PreviewRender render HTML for the preview
func (page *Page) PreviewRender(referer string) (string, error) {

	// get the page config
	page.GetConfig()

	// Render the page
	request := NewRequestMock(page.Config.Mock)
	if referer != "" {
		request.Referer = referer
	}

	warnings := []string{}
	ctx := NewBuildContext(nil)
	doc, warnings, err := page.Build(ctx, &BuildOption{
		SSR:         true,
		AssetRoot:   fmt.Sprintf("/api/__yao/sui/v1/%s/asset/%s/@assets", page.SuiID, page.TemplateID),
		KeepPageTag: false,
	})

	// Get the data
	var data Data = nil
	if page.Codes.DATA.Code != "" || page.GlobalData != nil {
		data, err = page.Exec(request)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
	}

	// Add Frame Height
	if request.Referer != "" {
		doc.Selection.Find("body").AppendHtml(`
			<script>
				function setIframeHeight(height) {
				window.parent.postMessage(
					{
					messageType: "setIframeHeight",
					iframeHeight: height,
					},
					"` + request.Referer + `"
				);
				}

				window.onload = function () {
				const contentHeight = document.documentElement.scrollHeight;
				console.log("window.onload: setIframeHeight", contentHeight);
				try {
					setIframeHeight(contentHeight + "px");
				} catch (err) {
					console.log(` + "`" + `setIframeHeight error: ${err}` + "`" + `);
				}
				};
			</script>
  		`)
	}

	// Add Warning
	if len(warnings) > 0 {
		warningHTML := "<div class=\"sui-warning\">"
		for _, warning := range warnings {
			warningHTML += fmt.Sprintf("<div>%s</div>", warning)
		}
		warningHTML += "</div>"
		doc.Selection.Find("body").AppendHtml(warningHTML)
	}

	html, err := doc.Html()
	if err != nil {
		return "", err
	}

	// Parser and render
	parser := NewTemplateParser(data, &ParserOption{Preview: true})
	html, err = parser.Render(html)
	if err != nil {
		return "", err
	}

	// Warnings should be added after rendering
	if len(parser.errors) > 0 {
		for _, err := range parser.errors {
			warnings = append(warnings, err.Error())
		}
	}
	return html, nil
}
