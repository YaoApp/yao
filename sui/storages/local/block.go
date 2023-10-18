package local

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
)

// Blocks get the blocks
func (tmpl *Template) Blocks() ([]core.IBlock, error) {
	path := filepath.Join(tmpl.Root, "__blocks")

	blocks := []core.IBlock{}
	if exist, _ := tmpl.local.fs.Exists(path); !exist {
		return blocks, nil
	}

	dirs, err := tmpl.local.fs.ReadDir(path, false)
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if !tmpl.local.fs.IsDir(dir) {
			continue
		}

		block, err := tmpl.getBlockFrom(dir)
		if err != nil {
			log.Error("Get block error: %v", err)
			continue
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

// Block get the block
func (tmpl *Template) Block(id string) (core.IBlock, error) {
	path := filepath.Join(tmpl.Root, "__blocks", id)
	if exist, _ := tmpl.local.fs.Exists(path); !exist {
		return nil, fmt.Errorf("Block %s not found", id)
	}

	block, err := tmpl.getBlockFrom(path)
	if err != nil {
		return nil, err
	}

	err = block.Load()
	if err != nil {
		return nil, err
	}

	_, err = block.Compile()
	if err != nil {
		return nil, err
	}

	return block, nil
}

// BlockMedia get the block media
func (tmpl *Template) BlockMedia(id string) (*core.Asset, error) {
	path := filepath.Join(tmpl.Root, "__blocks", id, "media.png")
	if exist, _ := tmpl.local.fs.Exists(path); exist {

		content, err := tmpl.local.fs.ReadFile(path)
		if err != nil {
			return nil, err
		}

		return &core.Asset{
			Type:    "image/png",
			Content: content,
		}, nil
	}

	path = filepath.Join(tmpl.Root, "__blocks", id, "media.svg")
	if exist, _ := tmpl.local.fs.Exists(path); !exist {
		return nil, fmt.Errorf("Block %s media not found (media.png / media.png )", id)
	}

	content, err := tmpl.local.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &core.Asset{
		Type:    "image/svg+xml",
		Content: content,
	}, nil
}

// BlockLayoutItems export the blocks
func (tmpl *Template) BlockLayoutItems() (*core.BlockLayoutItems, error) {

	path := filepath.Join(tmpl.Root, "__blocks", "export.json")
	if exist, _ := tmpl.local.fs.Exists(path); !exist {

		blocks, err := tmpl.Blocks()
		if err != nil {
			return nil, err
		}

		// Default layout items
		layoutItems := &core.BlockLayoutItems{
			Categories: []core.LayoutItem{{
				ID:     "Basic",
				Label:  "Basic",
				Blocks: []core.LayoutItem{},
			}},
			Locals: map[string]map[string]string{
				"zh-CN": {"Basic": "基础"},
				"zh-TW": {"Basic": "基礎"},
			},
		}

		for _, block := range blocks {
			layoutItems.Categories[0].Blocks = append(
				layoutItems.Categories[0].Blocks, core.LayoutItem{
					ID:    block.Get().ID,
					Label: block.Get().Name,
				})
		}
		return layoutItems, nil
	}

	data, err := tmpl.local.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	layoutItems := core.BlockLayoutItems{}
	err = application.Parse(path, data, &layoutItems)
	if err != nil {
		return nil, err
	}

	return &layoutItems, nil
}

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

func (tmpl *Template) getBlockFrom(path string) (core.IBlock, error) {
	id := tmpl.getBlockID(path)
	return tmpl.getBlock(id)
}

func (tmpl *Template) getBlock(id string) (core.IBlock, error) {

	path := filepath.Join(tmpl.Root, "__blocks", id)
	if !tmpl.local.fs.IsDir(path) {
		return nil, fmt.Errorf("Block %s not found", id)
	}

	jsFile := filepath.Join("/", id, fmt.Sprintf("%s.js", id))
	tsFile := filepath.Join("/", id, fmt.Sprintf("%s.ts", id))
	htmlFile := filepath.Join("/", id, fmt.Sprintf("%s.html", id))
	block := &Block{
		tmpl: tmpl,
		Block: &core.Block{
			ID: id,
			Codes: core.SourceCodes{
				HTML: core.Source{File: htmlFile},
				JS:   core.Source{File: jsFile},
				TS:   core.Source{File: tsFile},
			},
		},
	}

	return block, nil
}

func (tmpl *Template) getBlockID(path string) string {
	return filepath.Base(path)
}
