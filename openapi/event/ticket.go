package event

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

const (
	wsTicketTTL            = 30 * time.Second
	wsTicketProtocolPrefix = "yao-ticket."
	wsEventsProtocol       = "yao-events"
)

type wsTicket struct {
	UserID    string `json:"user_id"`
	TeamID    string `json:"team_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Subject   string `json:"subject,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Scope     string `json:"scope,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
	IssuedAt  int64  `json:"issued_at"`
}

type memoryTicket struct {
	ticket  wsTicket
	expires time.Time
}

type ticketManager struct {
	store  store.Store
	prefix string
	ttl    time.Duration

	mu  sync.Mutex
	mem map[string]memoryTicket
}

func newTicketManager(oauthSvc oauthtypes.OAuth) *ticketManager {
	tm := &ticketManager{ttl: wsTicketTTL, mem: map[string]memoryTicket{}}
	if provider, ok := oauthSvc.(interface {
		GetStore() store.Store
		GetKeyPrefix() string
	}); ok {
		tm.store = provider.GetStore()
		tm.prefix = provider.GetKeyPrefix()
	}
	return tm
}

func ticketToken(c *gin.Context) string {
	if token := strings.TrimSpace(c.Query("ticket")); token != "" {
		return token
	}
	for _, protocol := range strings.Split(c.GetHeader("Sec-WebSocket-Protocol"), ",") {
		protocol = strings.TrimSpace(protocol)
		if strings.HasPrefix(protocol, wsTicketProtocolPrefix) {
			return strings.TrimPrefix(protocol, wsTicketProtocolPrefix)
		}
	}
	return ""
}

func (tm *ticketManager) issue(c *gin.Context) {
	info := authorized.GetInfo(c)
	if info == nil || info.UserID == "" {
		response.RespondWithError(c, http.StatusUnauthorized, &response.ErrorResponse{
			Code: "unauthorized", ErrorDescription: "User not authenticated",
		})
		return
	}
	token, err := newTicketToken()
	if err != nil {
		response.RespondWithError(c, http.StatusInternalServerError, &response.ErrorResponse{
			Code: "server_error", ErrorDescription: "Failed to create event ticket",
		})
		return
	}
	payload := wsTicket{
		UserID: info.UserID, TeamID: info.TeamID, SessionID: info.SessionID,
		Subject: info.Subject, ClientID: info.ClientID, Scope: info.Scope,
		TenantID: info.TenantID, IssuedAt: time.Now().Unix(),
	}
	raw, _ := json.Marshal(payload)
	if tm.store != nil {
		if err := tm.store.Set(tm.key(token), string(raw), tm.ttl); err != nil {
			response.RespondWithError(c, http.StatusServiceUnavailable, &response.ErrorResponse{
				Code: "event_ticket_unavailable", ErrorDescription: "Failed to store event ticket",
			})
			return
		}
	} else {
		tm.mu.Lock()
		tm.mem[token] = memoryTicket{ticket: payload, expires: time.Now().Add(tm.ttl)}
		tm.mu.Unlock()
	}
	response.RespondWithSuccess(c, http.StatusOK, gin.H{"ticket": token, "expires_in": int(tm.ttl.Seconds())})
}

func (tm *ticketManager) consume(token string) (*oauthtypes.AuthorizedInfo, bool) {
	if token == "" {
		return nil, false
	}
	var raw interface{}
	ok := false
	if tm.store != nil {
		raw, ok = tm.store.GetDel(tm.key(token))
	} else {
		tm.mu.Lock()
		entry, found := tm.mem[token]
		delete(tm.mem, token)
		tm.mu.Unlock()
		if found && time.Now().Before(entry.expires) {
			return ticketInfo(entry.ticket), true
		}
		return nil, false
	}
	if !ok {
		return nil, false
	}
	var data []byte
	switch value := raw.(type) {
	case string:
		data = []byte(value)
	case []byte:
		data = value
	default:
		return nil, false
	}
	var ticket wsTicket
	if err := json.Unmarshal(data, &ticket); err != nil {
		return nil, false
	}
	if ticket.UserID == "" || time.Since(time.Unix(ticket.IssuedAt, 0)) > tm.ttl {
		return nil, false
	}
	return ticketInfo(ticket), true
}

func (tm *ticketManager) key(token string) string {
	return tm.prefix + "event:ticket:" + token
}

func ticketInfo(t wsTicket) *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: t.UserID, TeamID: t.TeamID, SessionID: t.SessionID,
		Subject: t.Subject, ClientID: t.ClientID, Scope: t.Scope,
		TenantID: t.TenantID,
	}
}

func newTicketToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
