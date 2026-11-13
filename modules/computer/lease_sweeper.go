package computer

import (
	"context"
	"time"
)

// RunLeaseSweeper periodically expires due computer leases until ctx is done.
func (s *Service) RunLeaseSweeper(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.ExpireDue(ctx, time.Now().UTC())
		}
	}
}
