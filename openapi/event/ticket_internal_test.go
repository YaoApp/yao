package event

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
	eventtypes "github.com/yaoapp/yao/event/types"
)

func TestWSTicketIssueAndOneTimeConsume(t *testing.T) {
	tm := newTicketManager(nil)
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/events/ticket", nil)
	c.Set("__user_id", "user-1")
	c.Set("__team_id", "team-1")
	c.Set("__subject", "subject-1")
	c.Set("__client_id", "client-1")
	c.Set("__scope", "event:events:read:team")

	tm.issue(c)
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotEmpty(t, body.Ticket)
	require.Equal(t, int(wsTicketTTL.Seconds()), body.ExpiresIn)

	info, ok := tm.consume(body.Ticket)
	require.True(t, ok)
	require.Equal(t, "user-1", info.UserID)
	require.Equal(t, "team-1", info.TeamID)
	require.Equal(t, "event:events:read:team", info.Scope)

	_, ok = tm.consume(body.Ticket)
	require.False(t, ok)
}

func TestTicketTokenSources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	require.Empty(t, ticketToken(c))

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/events?ticket=query-token", nil)
	require.Equal(t, "query-token", ticketToken(c))

	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	c.Request.Header.Set("Sec-WebSocket-Protocol", "yao-events, yao-ticket.header-token")
	require.Equal(t, "header-token", ticketToken(c))
}

func TestEventAllowedComputerScope(t *testing.T) {
	owner := &process.AuthorizedInfo{UserID: "u1", TeamID: "t1"}
	teammate := &process.AuthorizedInfo{UserID: "u2", TeamID: "t1"}
	other := &process.AuthorizedInfo{UserID: "u3", TeamID: "t2"}
	ev := &eventtypes.Event{Type: "computer.status", Payload: map[string]any{
		"computer_id": "cmp-1", "status": "online",
		"__yao_created_by": "u1", "__yao_team_id": "t1",
	}}

	require.True(t, eventAllowed(owner, ev))
	require.True(t, eventAllowed(teammate, ev))
	require.False(t, eventAllowed(other, ev))

	require.False(t, eventAllowed(other, &eventtypes.Event{Type: "computer.created", Payload: map[string]any{
		"computer_id": "cmp-2", "status": "pending",
		"__yao_created_by": "u1", "__yao_team_id": "t1",
	}}))
}
