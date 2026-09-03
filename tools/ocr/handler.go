package ocr

import "context"

// ProviderHandler is the unified interface for all OCR providers (traditional API and VLM).
type ProviderHandler interface {
	// Recognize performs OCR on the given request and returns a unified response.
	Recognize(ctx context.Context, req *OCRRequest) (*OCRResponse, error)

	// SupportedTypes returns the set of OCR types this provider supports.
	SupportedTypes() map[string]bool
}

// OCRRequest is the unified input for all OCR provider handlers.
type OCRRequest struct {
	Source       []byte                 // raw file bytes
	MimeType     string                 // e.g. "image/jpeg", "application/pdf"
	SourcePath   string                 // original URI (for logging only)
	Type         string                 // "general" | "table" | "handwriting" | "invoice" | ...
	OutputFormat string                 // "text" | "json" | "markdown"
	Mode         string                 // "accurate" | "standard"
	Language     string                 // ISO 639-1 or empty
	Pages        string                 // PDF page range or empty
	Prompt       string                 // user prompt for VLM-OCR; ignored by traditional handlers
	Extra        map[string]interface{} // provider-specific pass-through
}

// OCRResponse is the unified output from all OCR providers.
type OCRResponse struct {
	Text     string                 // full extracted text
	Blocks   []OCRBlock             // text blocks with coordinates (populated by json output_format)
	Fields   map[string]interface{} // structured key-value data (invoice, ID card, etc.)
	Pages    int                    // total pages detected
	Metadata map[string]string      // extra info: "degraded_from", "provider", "model", etc.
}

// OCRBlock represents a single text region with optional coordinates and confidence.
type OCRBlock struct {
	Text       string    `json:"text"`
	Confidence float64   `json:"confidence,omitempty"`
	BBox       []float64 `json:"bbox,omitempty"` // [x1, y1, x2, y2]; Azure: normalized 0-1, others: pixel coordinates
	Page       int       `json:"page,omitempty"`
}
