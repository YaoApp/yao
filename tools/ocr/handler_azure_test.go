package ocr

import "testing"

func TestAzureModelID(t *testing.T) {
	tests := []struct {
		ocrType, want string
	}{
		{"general", "prebuilt-read"},
		{"invoice", "prebuilt-invoice"},
		{"receipt", "prebuilt-receipt"},
		{"id_card", "prebuilt-idDocument"},
		{"passport", "prebuilt-idDocument"},
		{"table", "prebuilt-layout"},
		{"document", "prebuilt-layout"},
	}
	for _, tt := range tests {
		got := azureModelID(tt.ocrType)
		if got != tt.want {
			t.Errorf("azureModelID(%q) = %q, want %q", tt.ocrType, got, tt.want)
		}
	}
}

func TestAzureContentType(t *testing.T) {
	if got := azureContentType("image/jpeg"); got != "image/jpeg" {
		t.Errorf("got %q, want image/jpeg", got)
	}
	if got := azureContentType("application/pdf"); got != "application/pdf" {
		t.Errorf("got %q, want application/pdf", got)
	}
	if got := azureContentType("text/plain"); got != "application/octet-stream" {
		t.Errorf("got %q, want application/octet-stream", got)
	}
}

func TestParseAzureResponse_Basic(t *testing.T) {
	raw := `{
		"analyzeResult": {
			"content": "Hello Azure OCR",
			"pages": [{"width": 100, "height": 200, "lines": [
				{"content": "Hello", "polygon": [10,20,90,20,90,50,10,50]}
			]}]
		}
	}`
	resp, err := parseAzureResponse([]byte(raw), "general")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "Hello Azure OCR" {
		t.Errorf("text = %q, want %q", resp.Text, "Hello Azure OCR")
	}
	if resp.Pages != 1 {
		t.Errorf("pages = %d, want 1", resp.Pages)
	}
	if len(resp.Blocks) != 1 {
		t.Fatalf("blocks count = %d, want 1", len(resp.Blocks))
	}
	// bbox should be normalized
	bbox := resp.Blocks[0].BBox
	if bbox == nil || len(bbox) != 4 {
		t.Fatal("expected bbox with 4 values")
	}
	if bbox[0] != 0.1 || bbox[1] != 0.1 { // 10/100, 20/200
		t.Errorf("bbox[0:2] = %v, want [0.1, 0.1]", bbox[:2])
	}
}

func TestParseAzureResponse_Invoice(t *testing.T) {
	raw := `{
		"analyzeResult": {
			"content": "Invoice text",
			"pages": [{}],
			"documents": [{
				"fields": {
					"InvoiceId": {"content": "INV-001"},
					"TotalAmount": {"valueCurrency": {"amount": 100, "currencyCode": "USD"}}
				}
			}]
		}
	}`
	resp, err := parseAzureResponse([]byte(raw), "invoice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Fields) != 2 {
		t.Fatalf("fields count = %d, want 2", len(resp.Fields))
	}
	if resp.Fields["InvoiceId"] != "INV-001" {
		t.Errorf("InvoiceId = %v", resp.Fields["InvoiceId"])
	}
}

func TestAzurePolygonToBBox(t *testing.T) {
	polygon := []interface{}{10.0, 20.0, 90.0, 20.0, 90.0, 50.0, 10.0, 50.0}
	bbox := azurePolygonToBBox(polygon, 100, 200)
	if len(bbox) != 4 {
		t.Fatal("expected 4 values")
	}
	if bbox[0] != 0.1 || bbox[1] != 0.1 || bbox[2] != 0.9 || bbox[3] != 0.25 {
		t.Errorf("bbox = %v", bbox)
	}
}

func TestAzurePolygonToBBox_TooFew(t *testing.T) {
	polygon := []interface{}{10.0, 20.0}
	bbox := azurePolygonToBBox(polygon, 100, 200)
	if bbox != nil {
		t.Error("expected nil for too few points")
	}
}

func TestAzureSupportedTypes(t *testing.T) {
	h := &AzureHandler{}
	types := h.SupportedTypes()
	if !types["general"] || !types["invoice"] || !types["table"] {
		t.Error("Azure should support general, invoice, and table")
	}
	if !types["passport"] {
		t.Error("Azure should support passport via prebuilt-idDocument")
	}
	if types["handwriting"] {
		t.Error("Azure should not have handwriting as dedicated type")
	}
}
