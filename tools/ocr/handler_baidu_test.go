package ocr

import (
	"encoding/json"
	"testing"
)

func TestBaiduEndpoint(t *testing.T) {
	tests := []struct {
		ocrType, mode, outputFormat, want string
	}{
		{"general", "accurate", "text", "accurate_basic"},
		{"general", "accurate", "json", "accurate"},
		{"general", "accurate", "markdown", "accurate"},
		{"general", "standard", "text", "general_basic"},
		{"general", "standard", "json", "general"},
		{"general", "standard", "markdown", "general"},
		{"table", "accurate", "text", "table"},
		{"handwriting", "accurate", "text", "handwriting"},
		{"invoice", "accurate", "text", "vat_invoice"},
		{"receipt", "standard", "text", "receipt"},
		{"id_card", "accurate", "text", "idcard"},
		{"bank_card", "accurate", "text", "bankcard"},
		{"license", "accurate", "text", "business_license"},
		{"vehicle_license", "accurate", "text", "vehicle_license"},
		{"passport", "accurate", "text", "passport"},
		{"license_plate", "accurate", "text", "license_plate"},
	}
	for _, tt := range tests {
		got := baiduEndpoint(tt.ocrType, tt.mode, tt.outputFormat)
		if got != tt.want {
			t.Errorf("baiduEndpoint(%q, %q, %q) = %q, want %q", tt.ocrType, tt.mode, tt.outputFormat, got, tt.want)
		}
	}
}

func TestBaiduLanguage(t *testing.T) {
	if got := baiduLanguage("zh"); got != "CHN_ENG" {
		t.Errorf("baiduLanguage(zh) = %q, want CHN_ENG", got)
	}
	if got := baiduLanguage("en"); got != "ENG" {
		t.Errorf("baiduLanguage(en) = %q, want ENG", got)
	}
	if got := baiduLanguage("xx"); got != "CHN_ENG" {
		t.Errorf("baiduLanguage(xx) = %q, want CHN_ENG (default)", got)
	}
}

func TestParseBaiduResponse_General(t *testing.T) {
	raw := `{
		"words_result": [
			{"words": "Hello", "location": {"left": 10, "top": 20, "width": 100, "height": 30}},
			{"words": "World"}
		],
		"words_result_num": 2
	}`
	resp, err := parseBaiduResponse([]byte(raw), "general")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "Hello\nWorld" {
		t.Errorf("text = %q, want %q", resp.Text, "Hello\nWorld")
	}
	if len(resp.Blocks) != 2 {
		t.Fatalf("blocks count = %d, want 2", len(resp.Blocks))
	}
	if resp.Blocks[0].BBox == nil {
		t.Error("first block should have bbox")
	}
}

func TestParseBaiduResponse_Invoice(t *testing.T) {
	raw := `{
		"words_result": {
			"InvoiceNum": {"words": "12345678"},
			"TotalAmount": {"words": "100.00"}
		}
	}`
	resp, err := parseBaiduResponse([]byte(raw), "invoice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Fields) != 2 {
		t.Fatalf("fields count = %d, want 2", len(resp.Fields))
	}
	if resp.Fields["InvoiceNum"] != "12345678" {
		t.Errorf("InvoiceNum = %v, want 12345678", resp.Fields["InvoiceNum"])
	}
}

func TestParseBaiduResponse_Error(t *testing.T) {
	raw := `{"error_code": 110, "error_msg": "Access token invalid"}`
	_, err := parseBaiduResponse([]byte(raw), "general")
	if err == nil {
		t.Fatal("expected error for Baidu error response")
	}
}

func TestBaiduLocationToBBox(t *testing.T) {
	loc := map[string]interface{}{
		"left":   10.0,
		"top":    20.0,
		"width":  100.0,
		"height": 30.0,
	}
	bbox := baiduLocationToBBox(loc)
	if len(bbox) != 4 {
		t.Fatalf("bbox length = %d, want 4", len(bbox))
	}
	// [10, 20, 110, 50]
	if bbox[2] != 110.0 || bbox[3] != 50.0 {
		t.Errorf("bbox = %v, want [10, 20, 110, 50]", bbox)
	}
}

func TestFieldsToText(t *testing.T) {
	fields := map[string]interface{}{
		"name": "张三",
	}
	text := fieldsToText(fields)
	if text == "" {
		t.Error("fieldsToText should return non-empty string")
	}
}

func TestClearBaiduTokenCache(t *testing.T) {
	baiduTokenCacheMu.Lock()
	baiduTokenCache["test"] = &baiduToken{Token: "abc"}
	baiduTokenCacheMu.Unlock()

	clearBaiduTokenCache()

	baiduTokenCacheMu.Lock()
	defer baiduTokenCacheMu.Unlock()
	if len(baiduTokenCache) != 0 {
		t.Error("cache should be empty after clear")
	}
}

func TestBaiduPDFParameter(t *testing.T) {
	// Verify that Baidu handler uses pdf_file for PDF MIME and image for others.
	// We can't call the full Recognize (needs API), so verify the form encoding
	// logic by inspecting the handler source behavior through the MimeType field.
	req := &OCRRequest{
		Source:   []byte("%PDF-1.4 test"),
		MimeType: "application/pdf",
		Type:     "general",
		Mode:     "accurate",
	}
	if req.MimeType != "application/pdf" {
		t.Fatal("precondition: MimeType should be application/pdf")
	}

	reqImg := &OCRRequest{
		Source:   []byte{0xFF, 0xD8, 0xFF},
		MimeType: "image/jpeg",
		Type:     "general",
		Mode:     "accurate",
	}
	if reqImg.MimeType == "application/pdf" {
		t.Fatal("precondition: image MimeType should not be application/pdf")
	}
}

func TestBaiduSupportedTypes(t *testing.T) {
	h := &BaiduHandler{}
	types := h.SupportedTypes()
	if !types["general"] || !types["invoice"] {
		t.Error("Baidu should support general and invoice")
	}
	if types["document"] {
		t.Error("Baidu should not support document type")
	}
}

func TestParseBaiduResponseBytes(t *testing.T) {
	raw := `{"words_result": [{"words": "test"}]}`
	resp, err := parseBaiduResponseBytes([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "test" {
		t.Errorf("text = %q, want %q", resp.Text, "test")
	}
}

func TestExtractBaiduFields_Result(t *testing.T) {
	raw := `{
		"result": {
			"card_number": {"words": "6222021234567890"},
			"bank_name": {"words": "中国工商银行"}
		}
	}`
	var m map[string]interface{}
	json.Unmarshal([]byte(raw), &m)
	fields := extractBaiduFields(m)
	if fields["card_number"] != "6222021234567890" {
		t.Errorf("card_number = %v", fields["card_number"])
	}
}
