package ocr

import "testing"

func TestIsValidType(t *testing.T) {
	valid := []string{"general", "table", "handwriting", "invoice", "receipt", "id_card", "bank_card", "license", "vehicle_license", "passport", "license_plate", "document"}
	for _, v := range valid {
		if !isValidType(v) {
			t.Errorf("expected %q to be valid", v)
		}
	}

	invalid := []string{"", "foo", "General", "INVOICE", "pdf"}
	for _, v := range invalid {
		if isValidType(v) {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}

func TestDefaultType(t *testing.T) {
	if got := defaultType(""); got != "general" {
		t.Errorf("defaultType(\"\") = %q, want \"general\"", got)
	}
	if got := defaultType("table"); got != "table" {
		t.Errorf("defaultType(\"table\") = %q, want \"table\"", got)
	}
}

func TestDegradeType(t *testing.T) {
	supported := map[string]bool{"general": true, "table": true}

	actual, degraded := degradeType("table", supported)
	if actual != "table" || degraded != "" {
		t.Errorf("degradeType(table) = (%q, %q), want (\"table\", \"\")", actual, degraded)
	}

	actual, degraded = degradeType("invoice", supported)
	if actual != "general" || degraded != "invoice" {
		t.Errorf("degradeType(invoice) = (%q, %q), want (\"general\", \"invoice\")", actual, degraded)
	}
}
