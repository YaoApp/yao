package core

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/runtime/transform"
)

// Compile compile the component
func (component *Component) Compile() (string, error) {

	// Typescript is the default language
	// Typescript
	if component.Codes.TS.Code != "" {
		varName := strings.Replace(component.ID, "-", "_", -1)
		ts := strings.Replace(component.Codes.TS.Code, "export default", fmt.Sprintf("window.component__%s =", varName), 1)
		if component.Codes.HTML.Code != "" && !strings.Contains(component.Codes.TS.Code, "content:") {
			html := strings.ReplaceAll(component.Codes.HTML.Code, "`", "\\`")
			ts = strings.Replace(ts, "defaults: {", "defaults: {\n  components: `"+html+"`,", 1)

		}

		js, err := transform.TypeScript(ts, api.TransformOptions{
			Target:            api.ESNext,
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
		})

		if err != nil {
			return "", err
		}
		component.Compiled = js
		return js, nil
	}

	// Javascript
	if component.Codes.JS.Code == "" {
		return "", fmt.Errorf("Block %s has no JS code", component.ID)
	}

	varName := strings.Replace(component.ID, "-", "_", -1)
	js := strings.Replace(component.Codes.JS.Code, "export default", fmt.Sprintf("window.component__%s =", varName), 1)
	if component.Codes.HTML.Code != "" && !strings.Contains(component.Codes.JS.Code, "content:") {
		html := strings.ReplaceAll(component.Codes.HTML.Code, "`", "\\`")
		js = strings.Replace(js, "defaults: {", "defaults: {\n  components: `"+html+"`,", 1)
	}

	minified, err := transform.MinifyJS(js)
	if err != nil {
		return "", err
	}

	component.Compiled = minified
	return minified, nil
}

// Source get the compiled code
func (component *Component) Source() string {
	return component.Compiled
}
