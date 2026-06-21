package task

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/output/message"
)

// ErrDaemonStopping is returned when subscribing to a daemon that is shutting down
var ErrDaemonStopping = errors.New("daemon is stopping")

// Subscribe creates a message subscription for a task.
// If the daemon is alive, messages come from ringBuffer + live channel.
// If the daemon has stopped, messages are loaded from DB.
func Subscribe(ctx context.Context, auth *process.AuthorizedInfo, chatID string, opts *SubscribeOpts) (*Subscription, error) {
	dc, exists := GetDaemon(chatID)
	if exists {
		sub, err := dc.Subscribe(opts)
		if err == nil {
			return sub, nil
		}
	}

	ch := make(chan *message.Message, 64)
	doneCh := make(chan struct{})
	go func() {
		defer close(ch)
		msgs := loadMessagesFromDB(chatID, opts.AfterSeq)
		for _, m := range msgs {
			select {
			case ch <- m:
			case <-doneCh:
				return
			}
		}
	}()

	var once sync.Once
	return &Subscription{
		Ch:     ch,
		Cancel: func() { once.Do(func() { close(doneCh) }) },
	}, nil
}

// Subscribe on DaemonContext: dual-channel design for atomic replay + live
func (dc *DaemonContext) Subscribe(opts *SubscribeOpts) (*Subscription, error) {
	liveCh := make(chan *message.Message, 64)
	outputCh := make(chan *message.Message, 64)
	subID := uuid.New().String()

	dc.mu.Lock()
	if dc.subscribers == nil {
		dc.mu.Unlock()
		return nil, ErrDaemonStopping
	}

	var replay []*message.Message
	switch opts.Replay {
	case ReplayAll:
		replay = make([]*message.Message, len(dc.ringBuffer))
		copy(replay, dc.ringBuffer)
	case ReplayAfter:
		for _, m := range dc.ringBuffer {
			if m.Metadata != nil && int64(m.Metadata.Sequence) > opts.AfterSeq {
				replay = append(replay, m)
			}
		}
	}
	dc.subscribers[subID] = liveCh
	dc.mu.Unlock()

	doneCh := make(chan struct{})
	go func() {
		defer close(outputCh)
		for _, m := range replay {
			select {
			case outputCh <- m:
			case <-doneCh:
				return
			}
		}
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
	return &Subscription{
		Ch: outputCh,
		Cancel: func() {
			cancelOnce.Do(func() {
				dc.mu.Lock()
				delete(dc.subscribers, subID)
				dc.mu.Unlock()
				close(doneCh)
			})
		},
	}, nil
}

// loadMessagesFromDB loads messages from agent_message table
func loadMessagesFromDB(chatID string, afterSeq int64) []*message.Message {
	qb := capsule.Global.Query().Table(tableMessage()).
		Select("type", "props", "sequence").
		Where("chat_id", "=", chatID).
		OrderBy("sequence", "asc")

	if afterSeq > 0 {
		qb.Where("sequence", ">", afterSeq)
	}

	rows, err := qb.Limit(1000).Get()
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
