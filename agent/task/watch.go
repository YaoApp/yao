package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/output/message"
	storetypes "github.com/yaoapp/yao/agent/store/types"
)

// ChatStoreFn returns the ChatStore instance. Set by openapi/agent/task init to avoid import cycle.
var ChatStoreFn func() storetypes.ChatStore

// InfoByIDsFn retrieves assistant info by IDs. Set by openapi/agent/task init to avoid import cycle.
var InfoByIDsFn func(ids []string, locale ...string) map[string]*storetypes.AssistantInfo

// Watch creates a WatchStream for a task.
// If the daemon is alive, messages come from ringBuffer (replay) + live channel.
// If the daemon has stopped, messages are loaded from DB with pagination.
func Watch(ctx context.Context, auth *process.AuthorizedInfo, chatID string, opts *WatchOpts) (*WatchStream, error) {
	if opts == nil {
		opts = &WatchOpts{}
	}

	dc, exists := GetDaemon(chatID)
	if exists {
		stream, err := dc.Watch(opts)
		if err == nil {
			fmt.Printf("  • [task.watch] LIVE chatID=%s\n", chatID)
			return stream, nil
		}
		fmt.Printf("  • [task.watch] daemon found but Watch FAILED chatID=%s err=%v\n", chatID, err)
	} else {
		fmt.Printf("  • [task.watch] NO DAEMON chatID=%s → fallback to DB\n", chatID)
	}

	return watchFromDB(chatID, opts)
}

// watchFromDB loads messages from DB via ChatStore and sends a single read_complete event
func watchFromDB(chatID string, opts *WatchOpts) (*WatchStream, error) {
	ch := make(chan *message.Message, 4)
	doneCh := make(chan struct{})

	go func() {
		defer close(ch)

		if ChatStoreFn == nil {
			sendReadComplete(ch, doneCh, false, nil, nil, 0)
			return
		}

		chatStore := ChatStoreFn()
		if chatStore == nil {
			sendReadComplete(ch, doneCh, false, nil, nil, 0)
			return
		}

		filter := storetypes.MessageFilter{}
		if opts.Limit > 0 {
			filter.Limit = opts.Limit
		} else {
			filter.Limit = 100
		}
		if opts.BeforeID > 0 {
			filter.BeforeID = opts.BeforeID
		}

		messages, err := chatStore.GetMessages(chatID, filter)
		if err != nil {
			fmt.Printf("  • [task.watchFromDB] GetMessages ERROR chatID=%s err=%v\n", chatID, err)
			sendReadComplete(ch, doneCh, false, nil, nil, 0)
			return
		}
		if len(messages) == 0 {
			sendReadComplete(ch, doneCh, false, nil, nil, 0)
			return
		}

		assistantIDs := collectAssistantIDs(messages)
		var assistants map[string]*storetypes.AssistantInfo
		if len(assistantIDs) > 0 && InfoByIDsFn != nil {
			assistants = InfoByIDsFn(assistantIDs, opts.Locale)
		}

		hasMore := len(messages) >= filter.Limit
		firstID := messages[0].ID

		sendReadComplete(ch, doneCh, hasMore, messages, assistants, firstID)
	}()

	var once sync.Once
	return &WatchStream{
		Ch:     ch,
		Cancel: func() { once.Do(func() { close(doneCh) }) },
	}, nil
}

func sendReadComplete(ch chan<- *message.Message, doneCh <-chan struct{}, hasMore bool, messages []*storetypes.Message, assistants map[string]*storetypes.AssistantInfo, firstID int64) {
	marker := &message.Message{
		Type: "event",
		Props: map[string]interface{}{
			"event":      "read_complete",
			"live":       false,
			"has_more":   hasMore,
			"first_id":   firstID,
			"messages":   messages,
			"assistants": assistants,
		},
	}
	select {
	case ch <- marker:
	case <-doneCh:
	}
}

func collectAssistantIDs(messages []*storetypes.Message) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, msg := range messages {
		if msg.AssistantID != "" && !seen[msg.AssistantID] {
			seen[msg.AssistantID] = true
			ids = append(ids, msg.AssistantID)
		}
	}
	return ids
}

// Watch on DaemonContext: replay from ringBuffer with limit support, then live stream
func (dc *DaemonContext) Watch(opts *WatchOpts) (*WatchStream, error) {
	liveCh := make(chan *message.Message, 64)
	outputCh := make(chan *message.Message, 64)
	subID := uuid.New().String()

	dc.mu.Lock()
	if dc.subscribers == nil {
		dc.mu.Unlock()
		return nil, ErrDaemonStopping
	}

	var replay []*message.Message
	for _, m := range dc.ringBuffer {
		if opts.AfterSeq > 0 && m.Metadata != nil && int64(m.Metadata.Sequence) <= opts.AfterSeq {
			continue
		}
		replay = append(replay, m)
		if opts.Limit > 0 && len(replay) >= opts.Limit {
			break
		}
	}
	dc.subscribers[subID] = liveCh
	fmt.Printf("  • [task.watch.subscribe] chatID=%s subID=%s totalSubs=%d\n", dc.ChatID, subID, len(dc.subscribers))
	dc.mu.Unlock()

	doneCh := make(chan struct{})
	go func() {
		defer close(outputCh)

		// Phase 1: replay
		for _, m := range replay {
			select {
			case outputCh <- m:
			case <-doneCh:
				return
			}
		}

		// Phase 2: read_complete marker
		hasMore := opts.Limit > 0 && len(replay) >= opts.Limit
		lastSeq := int64(0)
		if len(replay) > 0 && replay[len(replay)-1].Metadata != nil {
			lastSeq = int64(replay[len(replay)-1].Metadata.Sequence)
		}
		marker := &message.Message{
			Type: "event",
			Props: map[string]interface{}{
				"event":    "read_complete",
				"has_more": hasMore,
				"last_seq": lastSeq,
				"live":     !hasMore,
			},
		}
		select {
		case outputCh <- marker:
		case <-doneCh:
			return
		}

		if hasMore {
			return // limit truncated, don't enter live (wait for next read)
		}

		// Phase 3: live (only when not truncated)
		for {
			select {
			case m, ok := <-liveCh:
				if !ok {
					return
				}
				select {
				case outputCh <- m:
				case <-doneCh:
					return
				}
			case <-doneCh:
				return
			}
		}
	}()

	var cancelOnce sync.Once
	return &WatchStream{
		Ch: outputCh,
		Cancel: func() {
			cancelOnce.Do(func() {
				fmt.Printf("  • [task.watch.cancel] chatID=%s subID=%s\n", dc.ChatID, subID)
				dc.mu.Lock()
				delete(dc.subscribers, subID)
				dc.mu.Unlock()
				close(doneCh)
			})
		},
		LiveMode: true,
	}, nil
}

// LoadHistoryMessages loads history messages from DB with complete fields and DESC pagination.
// before=0 loads the latest messages; before>0 loads messages with seq < before (older).
// Returns messages in chronological order (ASC) and whether there are more older messages.
func LoadHistoryMessages(chatID string, before int64, limit int) ([]*message.Message, bool) {
	if limit <= 0 {
		limit = 50
	}

	qb := capsule.Global.Query().Table(tableMessage()).
		Select("type", "props", "sequence", "message_id", "block_id", "thread_id", "role", "created_at").
		Where("chat_id", "=", chatID).
		WhereNull("deleted_at")

	if before > 0 {
		qb.Where("sequence", "<", before)
	}

	// DESC + limit+1 to determine hasMore, then reverse to chronological order
	qb.OrderBy("sequence", "desc").Limit(limit + 1)

	rows, err := qb.Get()
	if err != nil {
		fmt.Printf("  • [task.LoadHistoryMessages] ERROR chatID=%s err=%v\n", chatID, err)
		return nil, false
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	// Reverse to chronological order (oldest first)
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	msgs := make([]*message.Message, 0, len(rows))
	for _, row := range rows {
		msg := &message.Message{
			Type:      getString(row, "type"),
			MessageID: getString(row, "message_id"),
			BlockID:   getString(row, "block_id"),
			ThreadID:  getString(row, "thread_id"),
		}
		if propsRaw, ok := row["props"]; ok && propsRaw != nil {
			if propsStr, ok := propsRaw.(string); ok {
				var props map[string]interface{}
				if err := json.Unmarshal([]byte(propsStr), &props); err == nil {
					msg.Props = props
				}
			}
		}
		seq := getInt(row, "sequence")
		var ts int64
		if t := getTime(row, "created_at"); t != nil {
			ts = t.UnixMilli()
		} else {
			ts = time.Now().UnixMilli()
		}
		msg.Metadata = &message.Metadata{
			Sequence:  seq,
			Timestamp: ts,
		}
		msgs = append(msgs, msg)
	}
	return msgs, hasMore
}
