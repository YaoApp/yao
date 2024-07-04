package local

// func TestTemplateComponents(t *testing.T) {
// 	tests := prepare(t)
// 	defer clean()

// 	tmpl, err := tests.Demo.GetTemplate("tech-blue")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	components, err := tmpl.Components()
// 	if err != nil {
// 		t.Fatalf("Components error: %v", err)
// 	}

// 	if len(components) < 2 {
// 		t.Fatalf("Components error: %v", len(components))
// 	}
// 	assert.Equal(t, "Box", components[0].(*Component).ID)
// 	assert.Equal(t, "/Box/Box.html", components[0].(*Component).Codes.HTML.File)
// 	assert.Equal(t, "/Box/Box.js", components[0].(*Component).Codes.JS.File)
// 	assert.Equal(t, "/Box/Box.ts", components[0].(*Component).Codes.TS.File)

// 	assert.Equal(t, "Card", components[1].(*Component).ID)
// 	assert.Equal(t, "/Card/Card.html", components[1].(*Component).Codes.HTML.File)
// 	assert.Equal(t, "/Card/Card.js", components[1].(*Component).Codes.JS.File)
// 	assert.Equal(t, "/Card/Card.ts", components[1].(*Component).Codes.TS.File)

// 	assert.Equal(t, "Nav", components[2].(*Component).ID)
// 	assert.Equal(t, "/Nav/Nav.html", components[2].(*Component).Codes.HTML.File)
// 	assert.Equal(t, "/Nav/Nav.js", components[2].(*Component).Codes.JS.File)
// 	assert.Equal(t, "/Nav/Nav.ts", components[2].(*Component).Codes.TS.File)

// }

// func TestTemplateComponentJS(t *testing.T) {
// 	tests := prepare(t)
// 	defer clean()

// 	tmpl, err := tests.Demo.GetTemplate("tech-blue")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	component, err := tmpl.Component("Card")
// 	if err != nil {
// 		t.Fatalf("Components error: %v", err)
// 	}

// 	assert.Equal(t, "Card", component.(*Component).ID)
// 	assert.NotEmpty(t, component.(*Component).Codes.HTML.Code)
// 	assert.NotEmpty(t, component.(*Component).Codes.JS.Code)
// 	assert.Contains(t, component.(*Component).Compiled, "window.component__Card")
// 	assert.Contains(t, component.(*Component).Compiled, `<h1>Card</h1>`)
// }

// func TestTemplateComponentTS(t *testing.T) {
// 	tests := prepare(t)
// 	defer clean()

// 	tmpl, err := tests.Demo.GetTemplate("tech-blue")
// 	if err != nil {
// 		t.Fatalf("GetTemplate error: %v", err)
// 	}

// 	component, err := tmpl.Component("Box")
// 	if err != nil {
// 		t.Fatalf("Components error: %v", err)
// 	}

// 	assert.Equal(t, "Box", component.(*Component).ID)
// 	assert.NotEmpty(t, component.(*Component).Codes.HTML.Code)
// 	assert.NotEmpty(t, component.(*Component).Codes.TS.Code)
// 	assert.Contains(t, component.(*Component).Compiled, "window.component__Box")
// 	assert.Contains(t, component.(*Component).Compiled, `<div>Box</div>`)
// }
