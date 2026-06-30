package task

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"
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

// wsSession manages per-WS-connection state
type wsSession struct {
	mu          sync.Mutex
	activeWatch *tasksvc.WatchStream
	streamDone  chan struct{}
	liveMode    bool // true = current watch is subscribed to a live daemon
}

func (s *wsSession) cancelWatch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeWatch != nil {
		s.activeWatch.Cancel()
		s.activeWatch = nil
	}
}

func handleWS(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	outCh := make(chan *message.Message, 128)
	stopCh := make(chan struct{})
	session := &wsSession{}

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for {
			select {
			case msg, ok := <-outCh:
				if !ok {
					conn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, "stream ended"))
					return
				}
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if conn.WriteJSON(msg) != nil {
					return
				}
			case <-stopCh:
				return
			}
		}
	}()

	normalClose := wsCommandLoop(conn, auth, chatID, session, outCh, stopCh)
	session.cancelWatch()

	if normalClose {
		close(outCh)
	} else {
		close(stopCh)
	}
	<-writerDone
}

// wsCommandLoop returns true = stream ended normally (graceful close); false = abnormal exit
func wsCommandLoop(conn *websocket.Conn, auth *process.AuthorizedInfo, chatID string, session *wsSession, outCh chan<- *message.Message, stopCh <-chan struct{}) bool {
	for {
		session.mu.Lock()
		sd := session.streamDone
		live := session.liveMode
		session.mu.Unlock()

		if sd != nil {
			select {
			case <-sd:
				if live {
					// Live stream ended (daemon completed) → 3s grace then close
					conn.SetReadDeadline(time.Now().Add(3 * time.Second))
				} else {
					// Non-live pipe ended (empty read / DB history) → keep alive
					conn.SetReadDeadline(time.Time{})
				}
			default:
				conn.SetReadDeadline(time.Time{})
			}
		} else {
			conn.SetReadDeadline(time.Time{})
		}

		var cmd tasksvc.WSCommand
		if err := conn.ReadJSON(&cmd); err != nil {
			if isTimeout(err) {
				return true
			}
			return false
		}

		switch cmd.Type {
		case "read":
			handleReadCmd(session, auth, chatID, cmd, outCh, stopCh)
		case "history":
			handleHistoryCmd(session, auth, chatID, cmd, outCh, stopCh)
		case "run":
			handleRunCmd(session, auth, chatID, cmd, outCh, stopCh)
		case "retry":
			handleRetryCmd(session, auth, chatID, cmd, outCh, stopCh)
		case "repeat":
			handleRepeatCmd(session, auth, chatID, cmd, outCh, stopCh)
		case "stop":
			tasksvc.Stop(context.Background(), auth, chatID, false)
			session.cancelWatch()
			sendEvent(outCh, stopCh, "stream_end", nil)
			return true
		case "cancel":
			tasksvc.Stop(context.Background(), auth, chatID, true)
			session.cancelWatch()
			sendEvent(outCh, stopCh, "stream_end", nil)
			return true
		}
	}
}

func handleReadCmd(session *wsSession, auth *process.AuthorizedInfo, chatID string, cmd tasksvc.WSCommand, outCh chan<- *message.Message, stopCh <-chan struct{}) {
	session.cancelWatch()

	stream, err := tasksvc.Watch(context.Background(), auth, chatID, &tasksvc.WatchOpts{
		AfterSeq: cmd.Since,
		BeforeID: cmd.Before,
		Limit:    cmd.Limit,
		Locale:   cmd.Locale,
	})
	if err != nil {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": "watch failed: " + err.Error()})
		return
	}

	session.mu.Lock()
	session.activeWatch = stream
	session.liveMode = stream.LiveMode
	session.mu.Unlock()

	pipeDone := make(chan struct{})
	session.mu.Lock()
	session.streamDone = pipeDone
	session.mu.Unlock()

	go func() {
		defer stream.Cancel()
		defer close(pipeDone)
		for {
			select {
			case msg, ok := <-stream.Ch:
				if !ok {
					return
				}
				select {
				case outCh <- msg:
				case <-stopCh:
					return
				}
			case <-stopCh:
				return
			}
		}
	}()
}

func handleHistoryCmd(session *wsSession, auth *process.AuthorizedInfo, chatID string, cmd tasksvc.WSCommand, outCh chan<- *message.Message, stopCh <-chan struct{}) {
	limit := cmd.Limit
	if limit <= 0 {
		limit = 50
	}

	stream, err := tasksvc.Watch(context.Background(), auth, chatID, &tasksvc.WatchOpts{
		BeforeID: cmd.Before,
		Limit:    limit,
		Locale:   cmd.Locale,
	})
	if err != nil {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": "history load failed: " + err.Error()})
		return
	}
	defer stream.Cancel()

	for msg := range stream.Ch {
		select {
		case outCh <- msg:
		case <-stopCh:
			return
		}
	}
}

func subscribeLiveOnly(session *wsSession, chatID string, outCh chan<- *message.Message, stopCh <-chan struct{}) {
	session.cancelWatch()

	dc, exists := tasksvc.GetDaemon(chatID)
	if !exists {
		sendEvent(outCh, stopCh, "live_status", map[string]any{"status": "idle"})
		return
	}
	stream, err := dc.SubscribeLive()
	if err != nil {
		sendEvent(outCh, stopCh, "live_status", map[string]any{"status": "idle"})
		return
	}
	sendEvent(outCh, stopCh, "live_status", map[string]any{"status": "running"})

	session.mu.Lock()
	session.activeWatch = stream
	session.liveMode = stream.LiveMode
	session.mu.Unlock()

	pipeDone := make(chan struct{})
	session.mu.Lock()
	session.streamDone = pipeDone
	session.mu.Unlock()

	go func() {
		defer stream.Cancel()
		defer close(pipeDone)
		for {
			select {
			case msg, ok := <-stream.Ch:
				if !ok {
					return
				}
				select {
				case outCh <- msg:
				case <-stopCh:
					return
				}
			case <-stopCh:
				return
			}
		}
	}()
}

func handleRunCmd(session *wsSession, auth *process.AuthorizedInfo, chatID string, cmd tasksvc.WSCommand, outCh chan<- *message.Message, stopCh <-chan struct{}) {
	task, _ := tasksvc.Get(context.Background(), auth, chatID)
	if task == nil {
		_, err := tasksvc.CreateFromWS(context.Background(), auth, &tasksvc.CreateFromWSReq{
			ChatID:   chatID,
			Metadata: cmd.Metadata,
		})
		if err != nil {
			sendEvent(outCh, stopCh, "error", map[string]any{"message": "failed to create task: " + err.Error()})
			return
		}
	} else if task.RunStatus == "running" || task.RunStatus == "queued" {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": "task already running"})
		return
	}

	result, err := tasksvc.Run(context.Background(), auth, chatID, &tasksvc.RunReq{
		Messages:    cmd.Messages,
		AssistantID: cmd.AssistantID,
		Model:       cmd.Model,
		Metadata:    cmd.Metadata,
		Priority:    cmd.Priority,
		Source:      "run",
		Locale:      cmd.Locale,
	})
	if err != nil {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": err.Error()})
		return
	}
	if result.Status == "queued" {
		sendEvent(outCh, stopCh, "queued", map[string]any{"position": result.Position})
	}
	subscribeLiveOnly(session, chatID, outCh, stopCh)
}

func handleRetryCmd(session *wsSession, auth *process.AuthorizedInfo, chatID string, cmd tasksvc.WSCommand, outCh chan<- *message.Message, stopCh <-chan struct{}) {
	task, _ := tasksvc.Get(context.Background(), auth, chatID)
	if task == nil {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": "task not found"})
		return
	}
	if task.RunStatus != "failed" && task.RunStatus != "cancelled" {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": "retry only for failed/cancelled tasks"})
		return
	}

	originalPrompt := tasksvc.GetOriginalPrompt(context.Background(), chatID)
	messages := []tasksvc.InputMessage{{Role: "user", Content: originalPrompt}}
	messages = append(messages, cmd.Messages...)

	result, err := tasksvc.Run(context.Background(), auth, chatID, &tasksvc.RunReq{
		Messages: messages,
		Model:    cmd.Model,
		Priority: cmd.Priority,
		Source:   "retry",
		Fresh:    true,
		Locale:   cmd.Locale,
	})
	if err != nil {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": err.Error()})
		return
	}
	if result.Status == "queued" {
		sendEvent(outCh, stopCh, "queued", map[string]any{"position": result.Position})
	}
	subscribeLiveOnly(session, chatID, outCh, stopCh)
}

func handleRepeatCmd(session *wsSession, auth *process.AuthorizedInfo, chatID string, cmd tasksvc.WSCommand, outCh chan<- *message.Message, stopCh <-chan struct{}) {
	task, _ := tasksvc.Get(context.Background(), auth, chatID)
	if task == nil {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": "task not found"})
		return
	}
	if task.RunStatus == "running" || task.RunStatus == "queued" {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": "task already running"})
		return
	}

	var promptContent interface{}
	var instrLocale string
	if task.Instruction != nil && task.Instruction.Prompt != "" {
		promptContent = task.Instruction.Prompt
		instrLocale = task.Instruction.Locale
	} else {
		promptContent = tasksvc.GetOriginalPrompt(context.Background(), chatID)
	}
	if promptContent == nil || promptContent == "" {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": "no instruction available"})
		return
	}

	locale := cmd.Locale
	if locale == "" {
		locale = instrLocale
	}
	messages := []tasksvc.InputMessage{{Role: "user", Content: promptContent}}
	result, err := tasksvc.Run(context.Background(), auth, chatID, &tasksvc.RunReq{
		Messages: messages,
		Model:    cmd.Model,
		Priority: cmd.Priority,
		Source:   "repeat",
		Locale:   locale,
	})
	if err != nil {
		sendEvent(outCh, stopCh, "error", map[string]any{"message": err.Error()})
		return
	}
	if result.Status == "queued" {
		sendEvent(outCh, stopCh, "queued", map[string]any{"position": result.Position})
	}
	subscribeLiveOnly(session, chatID, outCh, stopCh)
}

func sendEvent(outCh chan<- *message.Message, stopCh <-chan struct{}, event string, props map[string]any) {
	if props == nil {
		props = map[string]any{}
	}
	props["event"] = event
	msg := &message.Message{Type: "event", Props: props}
	select {
	case outCh <- msg:
	case <-stopCh:
	}
}

func isTimeout(err error) bool {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	return false
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
