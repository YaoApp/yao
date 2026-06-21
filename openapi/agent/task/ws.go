package task

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/output/message"
	tasksvc "github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func handleWS(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")
	afterSeq := parseInt64(c.Query("since"), 0)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	replayMode := tasksvc.ReplayAll
	if afterSeq > 0 {
		replayMode = tasksvc.ReplayAfter
	}
	sub, err := tasksvc.Subscribe(c.Request.Context(), auth, chatID, &tasksvc.SubscribeOpts{
		Replay:   replayMode,
		AfterSeq: afterSeq,
	})
	if err != nil {
		conn.WriteJSON(map[string]any{"error": err.Error()})
		return
	}
	defer sub.Cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wsWriter(conn, sub.Ch)
	}()

	wsReader(conn, auth, chatID)
	sub.Cancel()
	<-done
}

func wsReader(conn *websocket.Conn, auth *process.AuthorizedInfo, chatID string) {
	for {
		var cmd tasksvc.WSCommand
		if err := conn.ReadJSON(&cmd); err != nil {
			return
		}

		switch cmd.Type {
		case "run":
			isFirstRun := false
			task, _ := tasksvc.Get(context.Background(), auth, chatID)
			if task == nil {
				tasksvc.CreateFromWS(context.Background(), auth, &tasksvc.CreateFromWSReq{
					ChatID:   chatID,
					Metadata: cmd.Metadata,
				})
				isFirstRun = true
			} else {
				isFirstRun = (task.RunCount == 0)
			}

			_, err := tasksvc.Run(context.Background(), auth, chatID, &tasksvc.RunReq{
				Messages:    cmd.Messages,
				AssistantID: cmd.AssistantID,
				Metadata:    cmd.Metadata,
				Priority:    cmd.Priority,
			})

			if err == nil && isFirstRun {
				if firstMsg := tasksvc.ExtractFirstUserMessage(cmd.Messages); firstMsg != "" {
					tasksvc.ExtractTaskMetadata(chatID, firstMsg, auth)
				}
			}

		case "input":
			tasksvc.Input(context.Background(), auth, chatID, &tasksvc.InputReq{Messages: cmd.Messages})

		case "stop":
			tasksvc.Stop(context.Background(), auth, chatID, false)

		case "cancel":
			tasksvc.Stop(context.Background(), auth, chatID, true)
		}
	}
}

func wsWriter(conn *websocket.Conn, ch <-chan *message.Message) {
	for msg := range ch {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteJSON(msg); err != nil {
			return
		}
	}
}

func parseInt64(s string, def int64) int64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}
