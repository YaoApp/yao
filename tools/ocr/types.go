package ocr

// validOCRTypes enumerates all recognized OCR type values.
var validOCRTypes = map[string]bool{
	"general":         true,
	"table":           true,
	"handwriting":     true,
	"invoice":         true,
	"receipt":         true,
	"id_card":         true,
	"bank_card":       true,
	"license":         true,
	"vehicle_license": true,
	"passport":        true,
	"license_plate":   true,
	"document":        true,
}

// isValidType reports whether the given type value is a recognized OCR type.
func isValidType(t string) bool {
	return validOCRTypes[t]
}

// defaultType returns "general" when t is empty, otherwise t unchanged.
func defaultType(t string) string {
	if t == "" {
		return "general"
	}
	return t
}

// degradeType returns the actual type a handler should use.
// If the handler supports the requested type, it is returned as-is.
// Otherwise it degrades to "general" and sets degradedFrom.
func degradeType(requested string, supported map[string]bool) (actual string, degradedFrom string) {
	if supported[requested] {
		return requested, ""
	}
	return "general", requested
}
