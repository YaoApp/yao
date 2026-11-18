package acl

import (
	"context"
	"testing"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

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
	if isTeamSelectionEndpoint(&AccessRequest{Method: "POST", Path: "/user/teams/invitations/inv_123/accept"}) {
		t.Fatal("accept must not be a team-selection endpoint")
	}
	if !isInvitationAcceptEndpoint(&AccessRequest{Method: "POST", Path: "/user/teams/invitations/inv_123/accept"}) {
		t.Fatal("expected invitation accept endpoint")
	}
	for _, request := range []*AccessRequest{
		{Method: "GET", Path: "/user/teams/invitations/inv_123/accept"},
		{Method: "POST", Path: "/user/teams/invitations/inv_123"},
		{Method: "POST", Path: "/user/teams/invitations//accept"},
	} {
		if isInvitationAcceptEndpoint(request) {
			t.Fatalf("unexpected invitation accept endpoint: %+v", request)
		}
	}
}

func TestInvitationAcceptBypass(t *testing.T) {
	acl := &ACL{}
	request := &AccessRequest{Method: "POST", Path: "/user/teams/invitations/inv_123/accept"}
	allowed, endpoint, err := acl.enforce(context.Background(), &types.AuthorizedInfo{UserID: "user-1"}, request)
	if err != nil {
		t.Fatalf("unexpected bypass error: %v", err)
	}
	if !allowed || endpoint != nil {
		t.Fatalf("expected authenticated accept bypass, allowed=%v endpoint=%+v", allowed, endpoint)
	}
}
