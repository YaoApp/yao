package docs

import (
	"testing"

	goudoc "github.com/yaoapp/gou/doc"
	"github.com/yaoapp/gou/process"
)

func init() {
	goudoc.Register(&goudoc.Entry{
		Name:  "models.Find",
		Type:  goudoc.TypeProcess,
		Group: "models",
		Desc:  "Find records by conditions",
		Args: []goudoc.TypeValue{
			{Type: "object", Desc: "query conditions"},
		},
		Return: &goudoc.TypeValue{Type: "array", Desc: "matched records"},
	})
	goudoc.Register(&goudoc.Entry{
		Name:  "models.Save",
		Type:  goudoc.TypeProcess,
		Group: "models",
		Desc:  "Save a record",
		Args: []goudoc.TypeValue{
			{Type: "object", Desc: "record data"},
		},
		Return: &goudoc.TypeValue{Type: "number", Desc: "record ID"},
	})
}

func TestListHandler_All(t *testing.T) {
	proc := process.New("tools.doc_list", "", 20)
	result := ListHandler(proc)
	entries, ok := result.([]*goudoc.Entry)
	if !ok {
		t.Fatalf("expected []*goudoc.Entry, got %T", result)
	}
	if len(entries) < 2 {
		t.Errorf("expected at least 2 entries, got %d", len(entries))
	}
}

func TestListHandler_Search(t *testing.T) {
	proc := process.New("tools.doc_list", "Find", 10)
	result := ListHandler(proc)
	entries, ok := result.([]*goudoc.Entry)
	if !ok {
		t.Fatalf("expected []*goudoc.Entry, got %T", result)
	}
	if len(entries) == 0 {
		t.Error("expected at least one result for 'Find'")
	}
	for _, e := range entries {
		t.Logf("found: %s - %s", e.Name, e.Desc)
	}
}

func TestInspectHandler(t *testing.T) {
	proc := process.New("tools.doc_inspect", "models.Find")
	result := InspectHandler(proc)
	if result == nil {
		t.Fatal("expected non-nil result for models.Find")
	}
	entry, ok := result.(*goudoc.Entry)
	if !ok {
		t.Fatalf("expected *goudoc.Entry, got %T", result)
	}
	if entry.Name != "models.Find" {
		t.Errorf("expected name 'models.Find', got '%s'", entry.Name)
	}
}

func TestInspectHandler_NotFound(t *testing.T) {
	proc := process.New("tools.doc_inspect", "nonexistent.process")
	result := InspectHandler(proc)
	if result != nil {
		t.Error("expected nil for non-existent process")
	}
}

func TestValidateHandler_Valid(t *testing.T) {
	proc := process.New("tools.doc_validate", "models.Find")
	result := ValidateHandler(proc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	vr, ok := result.(*goudoc.ValidationResult)
	if !ok {
		t.Fatalf("expected *goudoc.ValidationResult, got %T", result)
	}
	if !vr.Valid {
		t.Error("expected valid=true for models.Find")
	}
}

func TestValidateHandler_Invalid(t *testing.T) {
	proc := process.New("tools.doc_validate", "nonexistent.process")
	result := ValidateHandler(proc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	vr, ok := result.(*goudoc.ValidationResult)
	if !ok {
		t.Fatalf("expected *goudoc.ValidationResult, got %T", result)
	}
	if vr.Valid {
		t.Error("expected valid=false for non-existent process")
	}
}
