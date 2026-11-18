package acl

import "testing"

func TestTeamSelectionBypassHelpers(t *testing.T) {
	if !hasTeamSelectionScope("openid profile builtin:teams:selection user:profile:read:own") {
		t.Fatal("expected team selection scope")
	}
	if hasTeamSelectionScope("openid profile") {
		t.Fatal("unexpected team selection scope")
	}
	for _, request := range []*AccessRequest{
		{Method: "GET", Path: "/user/profile"},
		{Method: "GET", Path: "/user/teams"},
		{Method: "GET", Path: "/user/teams/config"},
		{Method: "POST", Path: "/user/teams/select"},
	} {
		if !isTeamSelectionEndpoint(request) {
			t.Fatalf("expected selection endpoint: %+v", request)
		}
	}
	if isTeamSelectionEndpoint(&AccessRequest{Method: "GET", Path: "/user/teams/current"}) {
		t.Fatal("unexpected selection endpoint")
	}
}
