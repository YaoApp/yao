package volume

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalRoot_Containment(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir).(*localStorage)

	tests := []struct {
		name      string
		sessionID string
		wantSafe  bool
	}{
		{"normal id", "ws-abc123", true},
		{"dotdot traversal", "..", false},
		{"dotdot prefix", "../etc", false},
		{"embedded dotdot", "a/../../../etc", false},
		{"single dot", ".", true},
		{"nested valid", "valid-id", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vol.root(tt.sessionID)
			if tt.wantSafe {
				assert.Equal(t, filepath.Join(dir, tt.sessionID), result)
			} else {
				assert.Equal(t, filepath.Join(dir, "_invalid_"), result,
					"traversal session ID %q must resolve to _invalid_", tt.sessionID)
			}
		})
	}
}
