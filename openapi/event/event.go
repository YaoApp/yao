package event

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yaoapp/gou/process"
	yaoevent "github.com/yaoapp/yao/event"
	eventtypes "github.com/yaoapp/yao/event/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Attach registers the /v1/events WebSocket endpoint
func Attach(group *gin.RouterGroup, oauth oauthtypes.OAuth) {
	group.GET("", oauth.Guard, handleEventWS)
}

func handleEventWS(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan *eventtypes.Event, 256)

	subID := yaoevent.Subscribe("*", ch, yaoevent.Filter(func(ev *eventtypes.Event) bool {
		m, ok := ev.Payload.(map[string]any)
		if !ok {
			return false
		}
		switch {
		case strings.HasPrefix(ev.Type, "task."), strings.HasPrefix(ev.Type, "board."):
			return m["__yao_team_id"] == auth.TeamID
		case strings.HasPrefix(ev.Type, "mail."):
			return m["__yao_created_by"] == auth.UserID
		default:
			return false
		}
	}))
	defer yaoevent.Unsubscribe(subID)

	// Reader goroutine: drain client messages (pong, etc.)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			payload, _ := ev.Payload.(map[string]any)
			conn.WriteJSON(map[string]any{
				"type":      ev.Type,
				"timestamp": time.Now().UnixMilli(),
				"data":      stripInternalFields(payload),
			})
		case <-ticker.C:
			conn.WriteJSON(map[string]any{"type": "ping"})
		case <-done:
			return
		case <-c.Request.Context().Done():
			return
		}
	}
}

// stripInternalFields removes __yao_ prefixed fields from the payload
func stripInternalFields(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	clean := make(map[string]any, len(payload))
	for k, v := range payload {
		if !strings.HasPrefix(k, "__yao_") {
			clean[k] = v
		}
	}
	return clean
}

func toProcessAuth(info *oauthtypes.AuthorizedInfo) *process.AuthorizedInfo {
	if info == nil {
		return &process.AuthorizedInfo{}
	}
	return &process.AuthorizedInfo{
		Subject:   info.Subject,
		ClientID:  info.ClientID,
		Scope:     info.Scope,
		SessionID: info.SessionID,
		UserID:    info.UserID,
		TeamID:    info.TeamID,
		TenantID:  info.TenantID,
		Constraints: process.DataConstraints{
			OwnerOnly:   info.Constraints.OwnerOnly,
			CreatorOnly: info.Constraints.CreatorOnly,
			TeamOnly:    info.Constraints.TeamOnly,
		},
	}
}
