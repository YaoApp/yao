package task

import (
	"context"
	"errors"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/agent/output/message"
)

// ErrDaemonStopping is returned when subscribing to a daemon that is shutting down
var ErrDaemonStopping = errors.New("daemon is stopping")

// Subscribe is the legacy API. It delegates to Watch for backward compatibility.
// New code should use Watch() directly.
func Subscribe(ctx context.Context, auth *process.AuthorizedInfo, chatID string, opts *SubscribeOpts) (*Subscription, error) {
	watchOpts := &WatchOpts{}
	if opts != nil {
		watchOpts.AfterSeq = opts.AfterSeq
	}

	stream, err := Watch(ctx, auth, chatID, watchOpts)
	if err != nil {
		return nil, err
	}

	return &Subscription{
		Ch:     stream.Ch,
		Cancel: stream.Cancel,
	}, nil
}

// Subscribe on DaemonContext: legacy method that delegates to Watch for backward compatibility.
func (dc *DaemonContext) Subscribe(opts *SubscribeOpts) (*Subscription, error) {
	watchOpts := &WatchOpts{}
	if opts != nil {
		watchOpts.AfterSeq = opts.AfterSeq
	}
	stream, err := dc.Watch(watchOpts)
	if err != nil {
		return nil, err
	}
	return &Subscription{
		Ch:     stream.Ch,
		Cancel: stream.Cancel,
	}, nil
}

// loadMessagesFromDB is kept for backward compatibility (used by enrich_mail.go etc.)
func loadMessagesFromDB(chatID string, afterSeq int64) []*message.Message {
	msgs, _ := LoadHistoryMessages(chatID, 0, 1000)
	return msgs
}
