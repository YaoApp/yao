package core

import (
	"fmt"
	"strings"
)

// Compile compile the block
func (block *Block) Compile() (string, error) {

	if block.Codes.JS.Code == "" {
		return "", fmt.Errorf("Block %s has no JS code", block.ID)
	}

	js := strings.Replace(block.Codes.JS.Code, "export default", fmt.Sprintf("window.block__%s =", block.ID), 1)
	if block.Codes.HTML.Code != "" && !strings.Contains(block.Codes.JS.Code, "content:") {
		html := strings.ReplaceAll(block.Codes.HTML.Code, "`", "\\`")
		js = strings.Replace(js, "{", fmt.Sprintf("{\n  content: `%s`,", html), 1)
	}

	block.Compiled = js
	return js, nil
}
