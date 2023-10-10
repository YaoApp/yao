package local

import (
	"testing"

	"github.com/yaoapp/yao/sui/core"
)

func TestTemplateBuild(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	err = tmpl.Build(&core.BuildOption{SSR: true})
	if err != nil {
		t.Fatalf("Components error: %v", err)
	}
}
