package weixin

import (
	"context"
	"fmt"
	"time"

	agentcontext "github.com/yaoapp/yao/agent/context"
	events "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/event"
	weixinapi "github.com/yaoapp/yao/integrations/weixin"
)

const (
	maxConsecutiveFailures = 3
	backoffDuration        = 30 * time.Second
	retryDuration          = 2 * time.Second
	sessionPauseDuration   = 30 * time.Minute
	defaultTimeoutMs       = 35_000
)

func (a *Adapter) pollLoop(ctx context.Context, entry *botEntry) {
	syncBuf := loadSyncBuf(entry.accountID)
	nextTimeoutMs := defaultTimeoutMs
	failures := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resp, err := entry.bot.GetUpdates(ctx, syncBuf, nextTimeoutMs)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			failures++
			if failures >= maxConsecutiveFailures {
				failures = 0
				sleep(ctx, backoffDuration)
			} else {
				sleep(ctx, retryDuration)
			}
			continue
		}

		if resp.ErrCode == weixinapi.SessionExpiredErrCode || resp.Ret == weixinapi.SessionExpiredErrCode {
			log.Warn("weixin session expired, pausing %s robot=%s", sessionPauseDuration, entry.robotID)
			failures = 0
			sleep(ctx, sessionPauseDuration)
			continue
		}

		isApiError := (resp.Ret != 0) || (resp.ErrCode != 0)
		if isApiError {
			failures++
			if failures >= maxConsecutiveFailures {
				failures = 0
				sleep(ctx, backoffDuration)
			} else {
				sleep(ctx, retryDuration)
			}
			continue
		}
		failures = 0

		if resp.LongPollingTimeoutMs > 0 {
			nextTimeoutMs = resp.LongPollingTimeoutMs
		}

		if resp.GetUpdatesBuf != "" && resp.GetUpdatesBuf != syncBuf {
			syncBuf = resp.GetUpdatesBuf
			saveSyncBuf(entry.accountID, syncBuf)
		}

		for i := range resp.Msgs {
			a.handleMessage(ctx, entry, &resp.Msgs[i])
		}
	}
}

func (a *Adapter) handleMessage(ctx context.Context, entry *botEntry, msg *weixinapi.WeixinMessage) {
	var dedupKey string
	switch {
	case msg.MessageID != 0:
		dedupKey = fmt.Sprintf("wx:%s:mid:%d", entry.robotID, msg.MessageID)
	case msg.Seq != 0:
		dedupKey = fmt.Sprintf("wx:%s:seq:%d", entry.robotID, msg.Seq)
	default:
		dedupKey = fmt.Sprintf("wx:%s:%s:%d", entry.robotID, msg.FromUserID, msg.CreateTimeMs)
	}
	if !a.dedup.markSeen(dedupKey) {
		return
	}

	log.Info("incoming msg from=%s context_token=%s", msg.FromUserID, msg.ContextToken)

	groups := []string{"weixin", entry.accountID}
	content, mediaItems := convertMessage(ctx, entry.bot, msg.ItemList, groups)

	if content == "" && len(mediaItems) == 0 {
		return
	}

	var msgContent interface{}
	if len(mediaItems) == 0 {
		msgContent = content
	} else {
		parts := make([]interface{}, 0, 1+len(mediaItems))
		if content != "" {
			parts = append(parts, map[string]interface{}{"type": "text", "text": content})
		}
		for _, m := range mediaItems {
			parts = append(parts, map[string]interface{}{
				"type":      "file",
				"file_url":  m.Wrapper,
				"mime_type": m.MimeType,
				"file_name": m.FileName,
			})
		}
		msgContent = parts
	}

	messageID := ""
	if msg.MessageID != 0 {
		messageID = fmt.Sprintf("%d", msg.MessageID)
	}

	payload := events.MessagePayload{
		RobotID: entry.robotID,
		Messages: []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: msgContent},
		},
		Metadata: &events.MessageMetadata{
			Channel:   "weixin",
			MessageID: messageID,
			AppID:     entry.accountID,
			ChatID:    msg.FromUserID,
			SenderID:  msg.FromUserID,
			Locale:    "zh-cn",
			Extra: map[string]any{
				"context_token": msg.ContextToken,
				"sender_id":     msg.FromUserID,
				"app_id":        entry.accountID,
			},
		},
	}

	if _, err := event.Push(ctx, events.Message, payload); err != nil {
		log.Error("weixin adapter: event.Push failed robot=%s: %v", entry.robotID, err)
	}
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
