//go:build integration

package image_test

import (
	"testing"

	"github.com/yaoapp/gou/connector"
	image "github.com/yaoapp/yao/tools/image"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestResolveModelName_Override(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	if err != nil {
		t.Fatalf("connector.Select(openai.mock): %v", err)
	}

	got := image.ExportResolveModelName(conn, "gpt-image-1")
	if got != "gpt-image-1" {
		t.Errorf("expected override 'gpt-image-1', got %q", got)
	}
}

func TestResolveModelName_FromConnector(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	if err != nil {
		t.Fatalf("connector.Select(openai.mock): %v", err)
	}

	got := image.ExportResolveModelName(conn, "")
	settings := conn.Setting()
	expectedModel, _ := settings["model"].(string)
	if expectedModel == "" {
		t.Fatal("openai.mock connector has no model in settings")
	}
	if got != expectedModel {
		t.Errorf("expected connector model %q, got %q", expectedModel, got)
	}
}

func TestResolveModelName_EmptyOverride_ReturnsConnectorModel(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	if err != nil {
		t.Fatalf("connector.Select(openai.mock): %v", err)
	}

	override := image.ExportResolveModelName(conn, "dall-e-3")
	noOverride := image.ExportResolveModelName(conn, "")

	if override != "dall-e-3" {
		t.Errorf("with override: expected 'dall-e-3', got %q", override)
	}
	if noOverride == "dall-e-3" {
		t.Error("without override: should return connector model, not 'dall-e-3'")
	}
	if noOverride == "" {
		t.Error("without override: connector model should not be empty")
	}
}
