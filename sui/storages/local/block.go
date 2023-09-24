package local

import (
	"path/filepath"
)

// Load get the block from the storage
func (block *Block) Load() error {

	root := filepath.Join(block.tmpl.Root, "__blocks")

	// Type script is the default language
	tsFile := filepath.Join(root, block.Codes.TS.File)
	if exist, _ := block.tmpl.local.fs.Exists(tsFile); exist {
		tsCode, err := block.tmpl.local.fs.ReadFile(tsFile)
		if err != nil {
			return err
		}
		block.Codes.TS.Code = string(tsCode)

	} else {
		jsFile := filepath.Join(root, block.Codes.JS.File)
		jsCode, err := block.tmpl.local.fs.ReadFile(jsFile)
		if err != nil {
			return err
		}
		block.Codes.JS.Code = string(jsCode)
	}

	htmlFile := filepath.Join(root, block.Codes.HTML.File)
	if exist, _ := block.tmpl.local.fs.Exists(htmlFile); exist {
		htmlCode, err := block.tmpl.local.fs.ReadFile(htmlFile)
		if err != nil {
			return err
		}
		block.Codes.HTML.Code = string(htmlCode)
	}

	return nil
}
