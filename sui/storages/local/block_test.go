package local

// func TestTemplateBlocks(t *testing.T) {
// 	tests := prepare(t)
// 	defer clean()

// 	tmpl, err := tests.Demo.GetTemplate("tech-blue")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	blocks, err := tmpl.Blocks()
// 	if err != nil {
// 		t.Fatalf("Blocks error: %v", err)
// 	}

// 	if len(blocks) < 3 {
// 		t.Fatalf("Blocks error: %v", len(blocks))
// 	}

// 	assert.Equal(t, "ColumnsTwo", blocks[0].(*Block).ID)
// 	assert.Equal(t, "/ColumnsTwo/ColumnsTwo.html", blocks[0].(*Block).Codes.HTML.File)
// 	assert.Equal(t, "/ColumnsTwo/ColumnsTwo.js", blocks[0].(*Block).Codes.JS.File)
// 	assert.Equal(t, "/ColumnsTwo/ColumnsTwo.ts", blocks[0].(*Block).Codes.TS.File)

// 	assert.Equal(t, "Hero", blocks[1].(*Block).ID)
// 	assert.Equal(t, "/Hero/Hero.html", blocks[1].(*Block).Codes.HTML.File)
// 	assert.Equal(t, "/Hero/Hero.js", blocks[1].(*Block).Codes.JS.File)
// 	assert.Equal(t, "/Hero/Hero.ts", blocks[1].(*Block).Codes.TS.File)

// 	assert.Equal(t, "Image", blocks[2].(*Block).ID)
// 	assert.Equal(t, "/Image/Image.html", blocks[2].(*Block).Codes.HTML.File)
// 	assert.Equal(t, "/Image/Image.js", blocks[2].(*Block).Codes.JS.File)
// 	assert.Equal(t, "/Image/Image.ts", blocks[2].(*Block).Codes.TS.File)
// }

// func TestTemplateBlockJS(t *testing.T) {
// 	tests := prepare(t)
// 	defer clean()

// 	tmpl, err := tests.Demo.GetTemplate("tech-blue")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	block, err := tmpl.Block("ColumnsTwo")
// 	if err != nil {
// 		t.Fatalf("Blocks error: %v", err)
// 	}

// 	assert.Equal(t, "ColumnsTwo", block.(*Block).ID)
// 	assert.NotEmpty(t, block.(*Block).Codes.HTML.Code)
// 	assert.NotEmpty(t, block.(*Block).Codes.JS.Code)
// 	assert.Contains(t, block.(*Block).Compiled, "window.block__ColumnsTwo")
// 	assert.Contains(t, block.(*Block).Compiled, `<div class="columns-two-left"`)
// }

// func TestTemplateBlockTS(t *testing.T) {
// 	tests := prepare(t)
// 	defer clean()

// 	tmpl, err := tests.Demo.GetTemplate("tech-blue")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	block, err := tmpl.Block("Hero")
// 	if err != nil {
// 		t.Fatalf("Blocks error: %v", err)
// 	}

// 	assert.Equal(t, "Hero", block.(*Block).ID)
// 	assert.Empty(t, block.(*Block).Codes.HTML.Code)
// 	assert.NotEmpty(t, block.(*Block).Codes.TS.Code)
// 	assert.Contains(t, block.(*Block).Compiled, "window.block__Hero")
// 	assert.Contains(t, block.(*Block).Compiled, `<div data-gjs-type='Nav'></div>`)
// }

// func TestBlockLayoutItems(t *testing.T) {
// 	tests := prepare(t)
// 	defer clean()

// 	tmpl, err := tests.Demo.GetTemplate("tech-blue")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	items, err := tmpl.BlockLayoutItems()
// 	if err != nil {
// 		t.Fatalf("BlockLayoutItems error: %v", err)
// 	}

// 	assert.Equal(t, 3, len(items.Categories))

// 	tmpl, err = tests.Demo.GetTemplate("website-ai")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	items, err = tmpl.BlockLayoutItems()
// 	if err != nil {
// 		t.Fatalf("BlockLayoutItems error: %v", err)
// 	}

// 	assert.Equal(t, 1, len(items.Categories))
// }

// func TestBlockMedia(t *testing.T) {
// 	tests := prepare(t)
// 	defer clean()

// 	tmpl, err := tests.Demo.GetTemplate("tech-blue")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	media, err := tmpl.BlockMedia("ColumnsTwo")
// 	if err != nil {
// 		t.Fatalf("BlockMedia error: %v", err)
// 	}

// 	assert.Equal(t, "image/png", media.Type)
// }
