package core

import (
	"fmt"
)

// PreviewRender render HTML for the preview
func (page *Page) PreviewRender(request *Request) (string, error) {

	warnings := []string{}
	doc, warnings, err := page.Build(&BuildOption{
		SSR:       true,
		AssetRoot: request.AssetRoot,
	})

	data, _, err := page.Data(request)
	if err != nil {
		warnings = append(warnings, err.Error())
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

	parser := NewTemplateParser(data, nil)
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
