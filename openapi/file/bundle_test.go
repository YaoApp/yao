package file_test

import (
	"testing"

	"github.com/yaoapp/yao/openapi/file"
	types "github.com/yaoapp/yao/openapi/oauth/types"
)

func TestBundleWorkspaceOwnerCheck(t *testing.T) {
	tests := []struct {
		name string
		info *types.AuthorizedInfo
		want string
	}{
		{"nil", nil, ""},
		{"team", &types.AuthorizedInfo{TeamID: "team-1", UserID: "user-1"}, "team-1"},
		{"user only", &types.AuthorizedInfo{UserID: "user-1"}, "user-1"},
		{"empty", &types.AuthorizedInfo{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := file.ResolveOwner(tt.info); got != tt.want {
				t.Errorf("ResolveOwner() = %q, want %q", got, tt.want)
			}
		})
	}
}
