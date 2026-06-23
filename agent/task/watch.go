package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/output/message"
)

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

// watchFromDB loads messages from DB with pagination and sends read_complete marker
func watchFromDB(chatID string, opts *WatchOpts) (*WatchStream, error) {
	ch := make(chan *message.Message, 64)
	doneCh := make(chan struct{})

	go func() {
		defer close(ch)
		msgs := loadMessagesFromDBPaginated(chatID, opts.AfterSeq, opts.Limit)
		for _, m := range msgs {
			select {
			case ch <- m:
			case <-doneCh:
				return
			}
		}

		hasMore := opts.Limit > 0 && len(msgs) >= opts.Limit
		lastSeq := int64(0)
		if len(msgs) > 0 && msgs[len(msgs)-1].Metadata != nil {
			lastSeq = int64(msgs[len(msgs)-1].Metadata.Sequence)
		}
		marker := &message.Message{
			Type: "event",
			Props: map[string]interface{}{
				"event":    "read_complete",
				"has_more": hasMore,
				"last_seq": lastSeq,
				"live":     false,
			},
		}
		select {
		case ch <- marker:
		case <-doneCh:
		}
	}()

	var once sync.Once
	return &WatchStream{
		Ch:     ch,
		Cancel: func() { once.Do(func() { close(doneCh) }) },
	}, nil
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

// loadMessagesFromDBPaginated loads messages from agent_message table with limit support
func loadMessagesFromDBPaginated(chatID string, afterSeq int64, limit int) []*message.Message {
	qb := capsule.Global.Query().Table(tableMessage()).
		Select("type", "props", "sequence").
		Where("chat_id", "=", chatID).
		OrderBy("sequence", "asc")

	if afterSeq > 0 {
		qb.Where("sequence", ">", afterSeq)
	}
	if limit > 0 {
		qb.Limit(limit)
	} else {
		qb.Limit(1000)
	}

	rows, err := qb.Get()
	if err != nil {
		return nil
	}

	msgs := make([]*message.Message, 0, len(rows))
	for _, row := range rows {
		msg := &message.Message{
			Type: getString(row, "type"),
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
		msg.Metadata = &message.Metadata{Sequence: seq}
		msgs = append(msgs, msg)
	}
	return msgs
}
