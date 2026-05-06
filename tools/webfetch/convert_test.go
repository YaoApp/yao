package webfetch

import (
	"strings"
	"testing"
)

func TestExtractTitle(t *testing.T) {
	html := `<html><head><title>Hello World</title></head><body></body></html>`
	title := ExtractTitle(html)
	if title != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", title)
	}
}

func TestExtractTitle_Empty(t *testing.T) {
	html := `<html><head></head><body></body></html>`
	title := ExtractTitle(html)
	if title != "" {
		t.Errorf("expected empty, got '%s'", title)
	}
}

func TestExtractMetaDescription(t *testing.T) {
	html := `<html><head><meta name="description" content="Test description"></head></html>`
	desc := ExtractMetaDescription(html)
	if desc != "Test description" {
		t.Errorf("expected 'Test description', got '%s'", desc)
	}
}

func TestExtractContent(t *testing.T) {
	html := `<html><body>
		<script>var x = 1;</script>
		<article>` + strings.Repeat("This is article content. ", 20) + `</article>
	</body></html>`
	content := ExtractContent(html)
	if !strings.Contains(content, "article content") {
		t.Errorf("expected content to contain 'article content', got '%s'", content)
	}
}

func TestHtmlToMarkdown(t *testing.T) {
	html := `<html><body>
		<h1>Title</h1>
		<p>Paragraph text</p>
		<a href="https://example.com">Link</a>
		<ul><li>Item 1</li><li>Item 2</li></ul>
	</body></html>`

	md := HtmlToMarkdown(html)

	if !strings.Contains(md, "# Title") {
		t.Error("expected markdown heading")
	}
	if !strings.Contains(md, "Paragraph text") {
		t.Error("expected paragraph text")
	}
	if !strings.Contains(md, "[Link](https://example.com)") {
		t.Error("expected markdown link")
	}
	if !strings.Contains(md, "- Item 1") {
		t.Error("expected list item")
	}
}

func TestHtmlToMarkdown_CodeBlock(t *testing.T) {
	html := `<body><pre><code>func main() {}</code></pre></body>`
	md := HtmlToMarkdown(html)
	if !strings.Contains(md, "```") {
		t.Error("expected fenced code block")
	}
	if !strings.Contains(md, "func main()") {
		t.Error("expected code content")
	}
}

func TestCapStr(t *testing.T) {
	s := "hello world"
	if capStr(s, 5) != "hello" {
		t.Errorf("expected 'hello', got '%s'", capStr(s, 5))
	}
	if capStr(s, 100) != s {
		t.Errorf("expected full string, got '%s'", capStr(s, 100))
	}
}
