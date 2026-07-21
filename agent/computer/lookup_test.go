package computer

import (
	"testing"

	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	taiTypes "github.com/yaoapp/yao/tai/types"
)

func TestToNodeCandidate(t *testing.T) {
	meta := &taiTypes.NodeMeta{
		TaiID: "local",
		Mode:  "local",
		System: taiTypes.SystemInfo{
			OS:   "linux",
			Arch: "amd64",
		},
		Capabilities: taiTypes.Capabilities{
			Docker:   true,
			K8s:      false,
			HostExec: true,
			Runners:  []string{"yaocode", "tai"},
		},
	}

	c := toNodeCandidate(meta)
	if c.ID != "local" {
		t.Errorf("ID = %q, want %q", c.ID, "local")
	}
	if !c.IsLocal {
		t.Error("IsLocal should be true for local node in local mode")
	}
	if !c.CanBox {
		t.Error("CanBox should be true when Docker is true")
	}
	if !c.CanHost {
		t.Error("CanHost should be true when HostExec is true")
	}
	if c.OS != "linux" {
		t.Errorf("OS = %q, want %q", c.OS, "linux")
	}
	if c.Arch != "amd64" {
		t.Errorf("Arch = %q, want %q", c.Arch, "amd64")
	}
	if len(c.Runners) != 2 {
		t.Errorf("Runners len = %d, want 2", len(c.Runners))
	}
}

func TestToNodeCandidate_Cloud(t *testing.T) {
	meta := &taiTypes.NodeMeta{
		TaiID: "node-1",
		Mode:  "cloud",
		Capabilities: taiTypes.Capabilities{
			Docker:   false,
			K8s:      true,
			HostExec: false,
		},
	}

	c := toNodeCandidate(meta)
	if c.IsLocal {
		t.Error("IsLocal should be false for cloud node")
	}
	if !c.CanBox {
		t.Error("CanBox should be true when K8s is true")
	}
	if c.CanHost {
		t.Error("CanHost should be false when HostExec is false")
	}
}

func TestDeriveOwnerID(t *testing.T) {
	tests := []struct {
		name string
		auth *oauthtypes.AuthorizedInfo
		want string
	}{
		{
			name: "nil auth",
			auth: nil,
			want: "anonymous",
		},
		{
			name: "team ID present",
			auth: &oauthtypes.AuthorizedInfo{TeamID: "team-1", UserID: "user-1"},
			want: "team-1",
		},
		{
			name: "user ID only",
			auth: &oauthtypes.AuthorizedInfo{UserID: "user-1"},
			want: "user-1",
		},
		{
			name: "empty auth",
			auth: &oauthtypes.AuthorizedInfo{},
			want: "anonymous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveOwnerID(tt.auth)
			if got != tt.want {
				t.Errorf("deriveOwnerID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLookupOpts_EmptyWorkspace(t *testing.T) {
	_, err := Lookup(t.Context(), &LookupOpts{
		WorkspaceID: "",
	})
	if err == nil {
		t.Error("Lookup with empty workspace should return error")
	}
}
