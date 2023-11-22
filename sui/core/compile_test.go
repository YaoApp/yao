package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompile(t *testing.T) {
	prepare(t)
	defer clean()

	page := testPage(t)
	html, err := page.Compile(&BuildOption{})
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	assert.Contains(t, html, "input.data")
}
