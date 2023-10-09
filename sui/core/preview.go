package core

import (
	"fmt"
)

// PreviewRender render HTML for the preview
func (page *Page) PreviewRender(request *Request) (string, error) {

	warnings := []string{}
	html, err := page.BuildHTML(request.AssetRoot)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	data, _, err := page.Data(request)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	html, err = page.Render(html, data, warnings)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Add Style & Script & Warning
	doc, err := NewDocument([]byte(html))
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Add Style
	style, err := page.BuildStyle(request.AssetRoot)
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	doc.Selection.Find("head").AppendHtml(style)

	// Add Script
	script, err := page.BuildScript(request.AssetRoot)
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	doc.Selection.Find("body").AppendHtml(script)

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

	return doc.Html()
}
