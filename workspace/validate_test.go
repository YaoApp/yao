package workspace_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/workspace"
)

func TestValidateID(t *testing.T) {
	maxValid := "a" + strings.Repeat("b", 127) // 128 chars total
	tooLong := "a" + strings.Repeat("b", 128)  // 129 chars total

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"empty is ok", "", false},
		{"simple alphanumeric", "abc123", false},
		{"with hyphens", "ws-abc123def456", false},
		{"with underscores", "my_workspace_01", false},
		{"with dots", "v1.0.0", false},
		{"single char", "a", false},
		{"max length 128", maxValid, false},
		{"DefaultWorkspaceID format", "ws-a1b2c3d4e5f6", false},

		// rejection cases
		{"starts with dot", ".hidden", true},
		{"starts with hyphen", "-bad", true},
		{"starts with underscore", "_bad", true},
		{"double dot traversal", "a..b", true},
		{"pure double dot", "..", true},
		{"contains slash", "a/b", true},
		{"contains backslash", "a\\b", true},
		{"contains space", "a b", true},
		{"too long", tooLong, true},
		{"path traversal prefix", "../etc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := workspace.ValidateID(tt.id)
			if tt.wantErr {
				assert.Error(t, err, "ValidateID(%q) should fail", tt.id)
			} else {
				assert.NoError(t, err, "ValidateID(%q) should pass", tt.id)
			}
		})
	}
}

func TestValidateID_DefaultWorkspaceID(t *testing.T) {
	id := workspace.DefaultWorkspaceID("user1", "node1")
	assert.NoError(t, workspace.ValidateID(id), "DefaultWorkspaceID output must pass ValidateID")
}
