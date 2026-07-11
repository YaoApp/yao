package types_test

import (
	"testing"

	"github.com/yaoapp/yao/tai/types"
)

func TestIsPublicNode(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"local", true},
		{"cloud", true},
		{"tunnel", false},
		{"direct", false},
		{"", false},
		{"unknown", false},
	}
	for _, tt := range tests {
		if got := types.IsPublicNode(tt.mode); got != tt.want {
			t.Errorf("IsPublicNode(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}
