package core

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/runtime/transform"
)

// Compile compile the block
func (block *Block) Compile() (string, error) {

	// Typescript is the default language
	// Typescript
	if block.Codes.TS.Code != "" {
		varName := strings.Replace(block.ID, "-", "_", -1)
		ts := strings.Replace(block.Codes.TS.Code, "export default", fmt.Sprintf("window.block__%s =", varName), 1)
		if block.Codes.HTML.Code != "" && !strings.Contains(block.Codes.TS.Code, "content:") {
			html := strings.ReplaceAll(block.Codes.HTML.Code, "`", "\\`")
			ts = strings.Replace(ts, "{", fmt.Sprintf("{\n  content: `%s`,", html), 1)
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
		block.Compiled = js
		return js, nil
	}

	// Javascript
	if block.Codes.JS.Code == "" {
		return "", fmt.Errorf("Block %s has no JS code", block.ID)
	}

	varName := strings.Replace(block.ID, "-", "_", -1)
	js := strings.Replace(block.Codes.JS.Code, "export default", fmt.Sprintf("window.block__%s =", varName), 1)
	if block.Codes.HTML.Code != "" && !strings.Contains(block.Codes.JS.Code, "content:") {
		html := strings.ReplaceAll(block.Codes.HTML.Code, "`", "\\`")
		js = strings.Replace(js, "{", fmt.Sprintf("{\n  content: `%s`,", html), 1)
	}

	minified, err := transform.MinifyJS(js)
	if err != nil {
		return "", err
	}

	block.Compiled = minified
	return minified, nil
}

// Source get the compiled code
func (block *Block) Source() string {
	return block.Compiled
}

// Get get the block
func (block *Block) Get() *Block {
	return block
}
