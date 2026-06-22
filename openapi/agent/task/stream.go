package task

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	tasksvc "github.com/yaoapp/yao/agent/task"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

func handleSSE(c *gin.Context) {
	auth := toProcessAuth(authorized.GetInfo(c))
	chatID := c.Param("chat_id")
	afterSeq := parseInt64(c.Query("since"), 0)

	opts := &tasksvc.SubscribeOpts{Replay: tasksvc.ReplayAfter, AfterSeq: afterSeq}
	if afterSeq == 0 {
		opts.Replay = tasksvc.ReplayAll
	}

	sub, err := tasksvc.Subscribe(c.Request.Context(), auth, chatID, opts)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	defer sub.Cancel()

	c.Header("Content-Type", "text/event-stream;charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	flusher, _ := c.Writer.(http.Flusher)

	for {
		select {
		case msg, ok := <-sub.Ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(msg)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			if flusher != nil {
				flusher.Flush()
			}

		case <-ticker.C:
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			if flusher != nil {
				flusher.Flush()
			}

		case <-c.Request.Context().Done():
			return
		}
	}
}
