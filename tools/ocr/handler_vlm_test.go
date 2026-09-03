package ocr

import "testing"

func TestVLMSupportedTypes(t *testing.T) {
	h := &VLMHandler{}
	types := h.SupportedTypes()
	for _, typ := range []string{"general", "table", "handwriting", "document", "invoice", "receipt", "id_card", "bank_card", "license", "vehicle_license", "passport", "license_plate"} {
		if !types[typ] {
			t.Errorf("VLM should support %q", typ)
		}
	}
}

func TestVLMRejectsPDF(t *testing.T) {
	h := &VLMHandler{ConnectorID: "test-connector"}
	req := &OCRRequest{
		Source:   []byte("%PDF-1.4 test"),
		MimeType: "application/pdf",
		Type:     "general",
		Mode:     "accurate",
	}
	_, err := h.Recognize(t.Context(), req)
	if err == nil {
		t.Fatal("VLM handler should reject PDF input")
	}
	if !containsStr(err.Error(), "does not support PDF") {
		t.Errorf("error = %q, want mention of PDF not supported", err.Error())
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
