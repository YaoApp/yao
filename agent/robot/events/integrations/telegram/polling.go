package telegram

import (
	"context"
	"time"

	kunlog "github.com/yaoapp/kun/log"
	tgapi "github.com/yaoapp/yao/integrations/telegram"
)

const (
	pollInterval = 60 * time.Second
	pollTimeout  = 30 // seconds, Telegram long-polling timeout per request
)

// pollLoop runs a single goroutine that iterates all registered bots
// every pollInterval, calling getUpdates for each one sequentially.
func (a *Adapter) pollLoop() {
	log.Info("pollLoop started, interval=%s", pollInterval)
	a.pollAll()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			log.Info("pollLoop stopped")
			return
		case <-ticker.C:
			a.pollAll()
		}
	}
}

func (a *Adapter) pollAll() {
	entries := a.snapshot()
	kunlog.Trace("[robot:telegram] pollAll bots=%d", len(entries))
	if len(entries) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), pollInterval)
	defer cancel()

	for _, entry := range entries {
		select {
		case <-a.stopCh:
			return
		default:
		}

		kunlog.Trace("[robot:telegram] polling robot=%s offset=%d", entry.robotID, entry.offset)
		groups := []string{"telegram", entry.robotID}
		msgs, err := entry.bot.GetUpdates(ctx, entry.offset, pollTimeout, groups)
		if err != nil {
			log.Error("getUpdates failed robot=%s: %v", entry.robotID, err)
			continue
		}

		if len(msgs) == 0 {
			continue
		}

		// Advance offset for all received messages
		for _, cm := range msgs {
			if cm.UpdateID >= entry.offset {
				entry.offset = cm.UpdateID + 1
			}
		}

		// Group by chatID, preserving order
		grouped := groupByChatID(msgs)
		log.Info("robot=%s got %d updates in %d chats", entry.robotID, len(msgs), len(grouped))

		for chatID, chatMsgs := range grouped {
			log.Debug("robot=%s chat=%d messages=%d", entry.robotID, chatID, len(chatMsgs))
			a.handleMessages(ctx, entry, chatMsgs)
		}
	}
}

// groupByChatID groups messages by chat ID, preserving chronological order.
func groupByChatID(msgs []*tgapi.ConvertedMessage) map[int64][]*tgapi.ConvertedMessage {
	grouped := make(map[int64][]*tgapi.ConvertedMessage)
	for _, cm := range msgs {
		grouped[cm.ChatID] = append(grouped[cm.ChatID], cm)
	}
	return grouped
}
