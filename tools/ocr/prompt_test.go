package ocr

import (
	"strings"
	"testing"
)

func TestBuildOCRSystemPrompt_General(t *testing.T) {
	prompt := buildOCRSystemPrompt("general", "text", "accurate", "")
	if !strings.Contains(prompt, "OCR") {
		t.Error("general prompt should mention OCR")
	}
	if !strings.Contains(prompt, "最高识别精度") {
		t.Error("accurate mode should add precision instruction")
	}
}

func TestBuildOCRSystemPrompt_Table(t *testing.T) {
	prompt := buildOCRSystemPrompt("table", "text", "standard", "")
	if !strings.Contains(prompt, "表格") {
		t.Error("table prompt should mention table")
	}
	if strings.Contains(prompt, "最高识别精度") {
		t.Error("standard mode should not add precision instruction")
	}
	if !strings.Contains(prompt, "简洁") {
		t.Error("standard mode should add conciseness instruction")
	}
}

func TestBuildOCRSystemPrompt_Markdown(t *testing.T) {
	prompt := buildOCRSystemPrompt("general", "markdown", "standard", "")
	if !strings.Contains(prompt, "Markdown") {
		t.Error("markdown format should add Markdown instruction")
	}
}

func TestBuildOCRSystemPrompt_JSON(t *testing.T) {
	prompt := buildOCRSystemPrompt("invoice", "json", "standard", "")
	if !strings.Contains(prompt, "JSON") {
		t.Error("json format should add JSON instruction")
	}
	if !strings.Contains(prompt, "fields") {
		t.Error("invoice json should mention fields")
	}
}

func TestBuildOCRSystemPrompt_Language(t *testing.T) {
	prompt := buildOCRSystemPrompt("general", "text", "standard", "ja")
	if !strings.Contains(prompt, "ja") {
		t.Error("language hint should be included")
	}
}

func TestTypePrompt_AllTypes(t *testing.T) {
	types := []string{"general", "table", "handwriting", "invoice", "receipt", "id_card", "bank_card", "license", "vehicle_license", "passport", "license_plate", "document"}
	for _, typ := range types {
		p := typePrompt(typ)
		if p == "" {
			t.Errorf("typePrompt(%q) returned empty", typ)
		}
	}
}

func TestJsonFormatInstruction_StructuredTypes(t *testing.T) {
	structured := []string{"invoice", "receipt", "id_card", "bank_card", "license", "vehicle_license", "passport", "license_plate"}
	for _, typ := range structured {
		inst := jsonFormatInstruction(typ)
		if !strings.Contains(inst, "fields") {
			t.Errorf("jsonFormatInstruction(%q) should mention fields", typ)
		}
	}

	general := jsonFormatInstruction("general")
	if !strings.Contains(general, "blocks") {
		t.Error("general json should mention blocks")
	}
}
