package ocr

import (
	"strings"
	"testing"
)

func TestFormatResponse_Text(t *testing.T) {
	resp := &OCRResponse{Text: "Hello World", Pages: 1}
	result := formatResponse(resp, "text", "general")
	if s, ok := result.(string); !ok || s != "Hello World" {
		t.Errorf("text format = %v, want %q", result, "Hello World")
	}
}

func TestFormatResponse_JSON(t *testing.T) {
	resp := &OCRResponse{
		Text: "Hello",
		Blocks: []OCRBlock{
			{Text: "Hello", Confidence: 0.99},
		},
		Pages: 1,
	}
	result := formatResponse(resp, "json", "general")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("json format should return map, got %T", result)
	}
	if m["text"] != "Hello" {
		t.Errorf("text = %v", m["text"])
	}
	if m["pages"] != 1 {
		t.Errorf("pages = %v", m["pages"])
	}
	blocks, ok := m["blocks"].([]OCRBlock)
	if !ok || len(blocks) != 1 {
		t.Errorf("blocks = %v", m["blocks"])
	}
}

func TestFormatResponse_JSON_WithFields(t *testing.T) {
	resp := &OCRResponse{
		Text:   "Invoice content",
		Fields: map[string]interface{}{"InvoiceNum": "123"},
		Pages:  1,
	}
	result := formatResponse(resp, "json", "invoice")
	m := result.(map[string]interface{})
	if m["fields"] == nil {
		t.Error("json format should include fields for invoice")
	}
}

func TestFormatResponse_Markdown(t *testing.T) {
	resp := &OCRResponse{Text: "# Title\n\nParagraph", Pages: 1}
	result := formatResponse(resp, "markdown", "general")
	if s, ok := result.(string); !ok || s != "# Title\n\nParagraph" {
		t.Errorf("markdown format = %v", result)
	}
}

func TestFormatResponse_Markdown_WithFields(t *testing.T) {
	resp := &OCRResponse{
		Text:   "InvoiceNum: 12345",
		Fields: map[string]interface{}{"InvoiceNum": "12345", "Amount": "100.00"},
		Pages:  1,
	}
	result := formatResponse(resp, "markdown", "invoice")
	s, ok := result.(string)
	if !ok {
		t.Fatalf("markdown format should return string, got %T", result)
	}
	if !strings.Contains(s, "**InvoiceNum**") || !strings.Contains(s, "**Amount**") {
		t.Errorf("markdown with Fields should format as KV list, got: %s", s)
	}
}

func TestFormatMarkdown_Fields(t *testing.T) {
	resp := &OCRResponse{
		Fields: map[string]interface{}{"Name": "张三", "ID": "123456"},
	}
	md := formatMarkdown(resp, "id_card")
	if md == "" {
		t.Error("formatMarkdown should return non-empty for fields")
	}
}

func TestFormatMarkdown_Blocks(t *testing.T) {
	resp := &OCRResponse{
		Blocks: []OCRBlock{{Text: "Line 1"}, {Text: "Line 2"}},
	}
	md := formatMarkdown(resp, "general")
	if md == "" {
		t.Error("formatMarkdown should return non-empty for blocks")
	}
}

func TestExtractExtra(t *testing.T) {
	args := map[string]interface{}{
		"extra": map[string]interface{}{"detect_direction": true},
		"other": "ignored",
	}
	extra := extractExtra(args)
	if extra["detect_direction"] != true {
		t.Error("extra should contain detect_direction")
	}
	if len(extra) != 1 {
		t.Errorf("extra length = %d, want 1", len(extra))
	}
}

func TestExtractExtra_Empty(t *testing.T) {
	extra := extractExtra(map[string]interface{}{})
	if len(extra) != 0 {
		t.Errorf("empty args should give empty extra, got %v", extra)
	}
}

func TestParseVLMJSON_Fields(t *testing.T) {
	resp := &OCRResponse{Text: `{"fields": {"InvoiceNum": "12345"}, "text": "Invoice"}`, Pages: 1}
	parseVLMJSON(resp.Text, resp)
	if resp.Fields == nil || resp.Fields["InvoiceNum"] != "12345" {
		t.Errorf("Fields = %v, want InvoiceNum=12345", resp.Fields)
	}
	if resp.Text != "Invoice" {
		t.Errorf("Text = %q, want %q", resp.Text, "Invoice")
	}
}

func TestParseVLMJSON_Blocks(t *testing.T) {
	resp := &OCRResponse{Text: `{"text": "Hello", "blocks": [{"text": "Hello", "confidence": 0.99, "page": 1}]}`}
	parseVLMJSON(resp.Text, resp)
	if len(resp.Blocks) != 1 {
		t.Fatalf("Blocks count = %d, want 1", len(resp.Blocks))
	}
	if resp.Blocks[0].Text != "Hello" || resp.Blocks[0].Confidence != 0.99 {
		t.Errorf("Block = %+v", resp.Blocks[0])
	}
}

func TestParseVLMJSON_CodeFence(t *testing.T) {
	resp := &OCRResponse{Text: "```json\n{\"fields\": {\"Name\": \"Test\"}}\n```"}
	parseVLMJSON(resp.Text, resp)
	if resp.Fields == nil || resp.Fields["Name"] != "Test" {
		t.Errorf("Fields = %v, want Name=Test after code fence strip", resp.Fields)
	}
}

func TestParseVLMJSON_InvalidJSON(t *testing.T) {
	resp := &OCRResponse{Text: "This is just plain text, not JSON"}
	parseVLMJSON(resp.Text, resp)
	if resp.Fields != nil || len(resp.Blocks) != 0 {
		t.Error("invalid JSON should leave Fields/Blocks empty")
	}
}

func TestStripCodeFence(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{`{"a": 1}`, `{"a": 1}`},
		{"```json\n{\"a\": 1}\n```", `{"a": 1}`},
		{"```\n{\"a\": 1}\n```", `{"a": 1}`},
		{"  ```json\n{\"a\": 1}\n```  ", `{"a": 1}`},
	}
	for _, tt := range tests {
		got := stripCodeFence(tt.input)
		if got != tt.want {
			t.Errorf("stripCodeFence(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractJSONStr(t *testing.T) {
	data := []byte(`{"status": "succeeded", "result": "ok"}`)
	if got := extractJSONStr(data, "status"); got != "succeeded" {
		t.Errorf("got %q, want %q", got, "succeeded")
	}
	if got := extractJSONStr(data, "missing"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := extractJSONStr([]byte("invalid"), "key"); got != "" {
		t.Errorf("got %q, want empty for invalid json", got)
	}
}
