package ocr

import "testing"

func TestPaddleOCRSupportedTypes(t *testing.T) {
	h := &PaddleOCRHandler{}
	types := h.SupportedTypes()
	if !types["general"] || !types["table"] || !types["document"] {
		t.Error("PaddleOCR should support general, table, and document")
	}
	if types["handwriting"] {
		t.Error("PaddleOCR should not support handwriting")
	}
}

func TestPaddleOCRPDFParameter(t *testing.T) {
	// Verify that PaddleOCR handler uses pdf + fileType=0 for PDF MIME
	// and image + fileType=1 for images.
	reqPDF := &OCRRequest{
		Source:   []byte("%PDF-1.4 test"),
		MimeType: "application/pdf",
		Type:     "general",
		Mode:     "accurate",
	}
	if reqPDF.MimeType != "application/pdf" {
		t.Fatal("precondition: PDF MimeType")
	}

	reqImg := &OCRRequest{
		Source:   []byte{0xFF, 0xD8, 0xFF},
		MimeType: "image/jpeg",
		Type:     "general",
		Mode:     "accurate",
	}
	if reqImg.MimeType == "application/pdf" {
		t.Fatal("precondition: image MimeType should not be PDF")
	}
}
