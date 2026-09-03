package ocr

import "testing"

func TestParseGoogleResponse_FullText(t *testing.T) {
	raw := `{
		"responses": [{
			"fullTextAnnotation": {
				"text": "Hello World",
				"pages": [{}]
			},
			"textAnnotations": [
				{"description": "Hello World"},
				{"description": "Hello", "boundingPoly": {"vertices": [{"x":10,"y":20},{"x":100,"y":20},{"x":100,"y":50},{"x":10,"y":50}]}},
				{"description": "World", "boundingPoly": {"vertices": [{"x":10,"y":60},{"x":100,"y":60},{"x":100,"y":90},{"x":10,"y":90}]}}
			]
		}]
	}`
	resp, err := parseGoogleResponse([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "Hello World" {
		t.Errorf("text = %q, want %q", resp.Text, "Hello World")
	}
	if len(resp.Blocks) != 2 {
		t.Fatalf("blocks count = %d, want 2", len(resp.Blocks))
	}
	if resp.Blocks[0].BBox == nil {
		t.Error("first block should have bbox")
	}
}

func TestParseGoogleResponse_Error(t *testing.T) {
	raw := `{
		"responses": [{
			"error": {"code": 3, "message": "Bad Request"}
		}]
	}`
	_, err := parseGoogleResponse([]byte(raw))
	if err == nil {
		t.Fatal("expected error for Google error response")
	}
}

func TestParseGoogleResponse_Empty(t *testing.T) {
	raw := `{"responses": []}`
	resp, err := parseGoogleResponse([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "" {
		t.Errorf("text = %q, want empty", resp.Text)
	}
}

func TestGoogleRejectsPDF(t *testing.T) {
	h := &GoogleHandler{APIKey: "test-key"}
	req := &OCRRequest{
		Source:   []byte("%PDF-1.4 test"),
		MimeType: "application/pdf",
		Type:     "general",
		Mode:     "accurate",
	}
	_, err := h.Recognize(t.Context(), req)
	if err == nil {
		t.Fatal("Google handler should reject PDF input")
	}
	if !contains(err.Error(), "does not support PDF") {
		t.Errorf("error = %q, want mention of PDF not supported", err.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestGoogleSupportedTypes(t *testing.T) {
	h := &GoogleHandler{}
	types := h.SupportedTypes()
	if !types["general"] || !types["document"] || !types["handwriting"] {
		t.Error("Google should support general, document, and handwriting")
	}
	if types["invoice"] {
		t.Error("Google should not support invoice type natively")
	}
}
