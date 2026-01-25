package agent

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/test"
)

func TestAgentExists(t *testing.T) {
	prepare(t)
	defer clean()

	exists := Exists()
	assert.True(t, exists, "Agent template should exist")
}

func TestHasAssistantPages(t *testing.T) {
	prepare(t)
	defer clean()

	hasPages := HasAssistantPages()
	assert.True(t, hasPages, "Should have assistant pages")
}

func TestGetAssistants(t *testing.T) {
	prepare(t)
	defer clean()

	agent := createAgent(t)
	assistants, err := agent.getAssistants()
	assert.Nil(t, err)
	assert.NotEmpty(t, assistants)

	// Sort for consistent comparison
	sort.Strings(assistants)

	// Should include both direct and nested assistants
	// Direct: tests.sui-pages (has pages directly)
	// Nested: tests.nested.demo (nested assistant with pages)
	found := map[string]bool{}
	for _, ast := range assistants {
		found[ast] = true
	}

	assert.True(t, found["tests.sui-pages"], "Should find tests.sui-pages assistant")
	assert.True(t, found["tests.nested.demo"], "Should find tests.nested.demo assistant")
}

func TestGetAssistantPagesRoot(t *testing.T) {
	prepare(t)
	defer clean()

	agent := createAgent(t)

	// Test direct assistant
	root := agent.getAssistantPagesRoot("tests.sui-pages")
	assert.Equal(t, "/assistants/tests/sui-pages/pages", root)

	// Test nested assistant
	root = agent.getAssistantPagesRoot("tests.nested.demo")
	assert.Equal(t, "/assistants/tests/nested/demo/pages", root)
}

func TestGetTemplate(t *testing.T) {
	prepare(t)
	defer clean()

	agent := createAgent(t)
	tmpl, err := agent.GetTemplate("agent")
	assert.Nil(t, err)
	assert.NotNil(t, tmpl)
	assert.Equal(t, "agent", tmpl.(*Template).ID)
}

func TestTemplatePages(t *testing.T) {
	prepare(t)
	defer clean()

	agent := createAgent(t)
	tmpl, err := agent.GetTemplate("agent")
	assert.Nil(t, err)

	pages, err := tmpl.Pages()
	assert.Nil(t, err)
	assert.NotEmpty(t, pages)

	// Check that we have pages from nested assistants
	routes := map[string]bool{}
	for _, page := range pages {
		routes[page.Get().Route] = true
	}

	// Should have pages from:
	// 1. Agent global pages (/index)
	// 2. Direct assistant (tests.sui-pages) -> /tests.sui-pages/dashboard
	// 3. Nested assistant (tests.nested.demo) -> /tests.nested.demo/article
	assert.True(t, routes["/index"], "Should have agent global page /index")
	assert.True(t, routes["/tests.sui-pages/dashboard"], "Should have direct assistant page /tests.sui-pages/dashboard")
	assert.True(t, routes["/tests.nested.demo/article"], "Should have nested assistant page /tests.nested.demo/article")
}

func TestTemplatePage(t *testing.T) {
	prepare(t)
	defer clean()

	agent := createAgent(t)
	tmpl, err := agent.GetTemplate("agent")
	assert.Nil(t, err)

	// Test getting agent global page
	page, err := tmpl.Page("/index")
	assert.Nil(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, "/index", page.Get().Route)

	// Test getting direct assistant page
	page, err = tmpl.Page("/tests.sui-pages/dashboard")
	assert.Nil(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, "/tests.sui-pages/dashboard", page.Get().Route)
	assert.Equal(t, "tests.sui-pages", page.(*Page).assistantID)

	// Test getting nested assistant page
	page, err = tmpl.Page("/tests.nested.demo/article")
	assert.Nil(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, "/tests.nested.demo/article", page.Get().Route)
	assert.Equal(t, "tests.nested.demo", page.(*Page).assistantID)

	// Test page not found
	_, err = tmpl.Page("/non-existent/page")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPageLoad(t *testing.T) {
	prepare(t)
	defer clean()

	agent := createAgent(t)
	tmpl, err := agent.GetTemplate("agent")
	assert.Nil(t, err)

	// Test loading nested assistant page
	page, err := tmpl.Page("/tests.nested.demo/article")
	assert.Nil(t, err)

	err = page.Load()
	assert.Nil(t, err)

	// Check that content was loaded
	p := page.Get()
	assert.NotEmpty(t, p.Codes.HTML.Code, "HTML code should be loaded")
	assert.NotEmpty(t, p.Codes.CSS.Code, "CSS code should be loaded")
}

func TestPageBuild(t *testing.T) {
	prepare(t)
	defer clean()

	agent := createAgent(t)

	// Register the agent SUI so page build can find it
	core.SUIs["agent"] = agent

	tmpl, err := agent.GetTemplate("agent")
	assert.Nil(t, err)

	// Test building nested assistant page
	page, err := tmpl.Page("/tests.nested.demo/article")
	assert.Nil(t, err)

	err = page.Load()
	assert.Nil(t, err)

	ctx := core.NewGlobalBuildContext(tmpl)
	warnings, err := page.Build(ctx, &core.BuildOption{
		PublicRoot: "/agents",
		AssetRoot:  "/agents/assets",
	})
	assert.Nil(t, err)
	assert.Empty(t, warnings)
}

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
}

func clean() {
	test.Clean()
}

func createAgent(t *testing.T) *Agent {
	dsl := &core.DSL{
		ID:   "agent",
		Name: "Agent",
		Public: &core.Public{
			Root:  "/agents",
			Host:  "/",
			Index: "/index",
		},
	}

	agent, err := New(dsl)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	return agent
}
