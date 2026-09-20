package ocr

// validOCRTypes enumerates all recognized OCR type values.
// Includes both internal names and Tao native names.
var validOCRTypes = map[string]bool{
	// Internal names
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
	// Tao native names
	"general_basic":  true,
	"accurate_basic": true,
	"idcard":         true,
	"bankcard":       true,
}

// validOutputFormats enumerates allowed output_format values.
var validOutputFormats = map[string]bool{
	"text":     true,
	"json":     true,
	"markdown": true,
}

// isValidType reports whether the given type value is a recognized OCR type.
func isValidType(t string) bool {
	return validOCRTypes[t]
}

// isValidOutputFormat reports whether the given format is allowed.
func isValidOutputFormat(f string) bool {
	return validOutputFormats[f]
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
