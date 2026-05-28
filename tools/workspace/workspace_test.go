package workspace

import (
	"testing"

	"github.com/yaoapp/gou/process"
)

func TestResolveOwner(t *testing.T) {
	tests := []struct {
		name string
		auth *process.AuthorizedInfo
		want string
	}{
		{"nil auth", nil, ""},
		{"team present", &process.AuthorizedInfo{TeamID: "team-1", UserID: "user-1"}, "team-1"},
		{"no team, user fallback", &process.AuthorizedInfo{UserID: "user-1"}, "user-1"},
		{"empty both", &process.AuthorizedInfo{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveOwner(tt.auth)
			if got != tt.want {
				t.Errorf("resolveOwner() = %q, want %q", got, tt.want)
			}
		})
	}
}
