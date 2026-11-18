package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth"
)

func TestRespondAlreadyMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	respondAlreadyMember(c, "team-1", "Acme")

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status=%d want %d", recorder.Code, http.StatusConflict)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "already_member" {
		t.Fatalf("error=%v", body["error"])
	}
	if body["team_id"] != "team-1" {
		t.Fatalf("team_id=%v", body["team_id"])
	}
	if body["team_name"] != "Acme" {
		t.Fatalf("team_name=%v", body["team_name"])
	}
}

func TestGinTeamInvitationAcceptAuthAndValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func(method, target, body, userID string) (*gin.Context, *httptest.ResponseRecorder) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		if userID != "" {
			c.Set("__user_id", userID)
		}
		return c, recorder
	}

	t.Run("unauthenticated", func(t *testing.T) {
		c, recorder := newContext(http.MethodPost, "/user/teams/invitations/inv_1/accept", `{"token":"t"}`, "")
		GinTeamInvitationAccept(c)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d want 401", recorder.Code)
		}
	})

	t.Run("missing invitation id", func(t *testing.T) {
		c, recorder := newContext(http.MethodPost, "/user/teams/invitations//accept", `{"token":"t"}`, "user-1")
		GinTeamInvitationAccept(c)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d want 400", recorder.Code)
		}
	})

	t.Run("provider unavailable", func(t *testing.T) {
		old := oauth.OAuth
		oauth.OAuth = nil
		defer func() { oauth.OAuth = old }()

		c, recorder := newContext(http.MethodPost, "/user/teams/invitations/inv_1/accept", `{"token":"t"}`, "user-1")
		c.Params = gin.Params{{Key: "invitation_id", Value: "inv_1"}}
		GinTeamInvitationAccept(c)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d want 500", recorder.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		c, recorder := newContext(http.MethodPost, "/user/teams/invitations/inv_1/accept", `{}`, "user-1")
		c.Params = gin.Params{{Key: "invitation_id", Value: "inv_1"}}
		GinTeamInvitationAccept(c)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d want 400", recorder.Code)
		}
	})
}
