package task

import (
	"context"
	"sync"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
)

// scheduleEngineImpl implements ScheduleEngine with a single ticker polling loop
type scheduleEngineImpl struct {
	mu         sync.RWMutex
	entries    map[string]*ScheduleEntry
	ticker     *time.Ticker
	tickerDone chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
}

// ScheduleEntry represents a scheduled task entry
type ScheduleEntry struct {
	ChatID    string
	Config    ScheduleConfig
	LastRun   time.Time
	NextRun   time.Time
	FailCount int
	Running   bool
}

// NewScheduleEngine creates a new schedule engine (call from main or init)
func NewScheduleEngine() *scheduleEngineImpl {
	return &scheduleEngineImpl{
		entries: make(map[string]*ScheduleEntry),
	}
}

func (se *scheduleEngineImpl) Start() error {
	se.ctx, se.cancel = context.WithCancel(globalShutdown)
	entries := loadScheduledTasks()
	se.mu.Lock()
	for _, e := range entries {
		se.entries[e.ChatID] = e
	}
	se.mu.Unlock()

	se.ticker = time.NewTicker(time.Minute)
	se.tickerDone = make(chan struct{})
	go se.tickerLoop()
	return nil
}

func (se *scheduleEngineImpl) Stop() {
	if se.tickerDone != nil {
		close(se.tickerDone)
	}
	if se.cancel != nil {
		se.cancel()
	}
}

func (se *scheduleEngineImpl) Update(chatID string, cfg ScheduleConfig) {
	se.mu.Lock()
	defer se.mu.Unlock()
	if !cfg.Enabled {
		delete(se.entries, chatID)
		return
	}
	se.entries[chatID] = &ScheduleEntry{
		ChatID: chatID,
		Config: cfg,
	}
}

func (se *scheduleEngineImpl) Remove(chatID string) {
	se.mu.Lock()
	defer se.mu.Unlock()
	delete(se.entries, chatID)
}

func (se *scheduleEngineImpl) tickerLoop() {
	for {
		select {
		case <-se.tickerDone:
			se.ticker.Stop()
			return
		case now := <-se.ticker.C:
			se.tick(now)
		}
	}
}

func (se *scheduleEngineImpl) tick(now time.Time) {
	se.mu.Lock()
	defer se.mu.Unlock()

	for _, entry := range se.entries {
		if entry.Running {
			continue
		}
		if se.shouldTrigger(entry, now) {
			entry.Running = true
			go se.onTrigger(entry)
		}
	}
}

func (se *scheduleEngineImpl) shouldTrigger(entry *ScheduleEntry, now time.Time) bool {
	switch entry.Config.Mode {
	case "times":
		return matchesTime(entry.Config.Times, now) && !triggeredThisMinute(entry, now)
	case "interval":
		dur := intervalDuration(entry.Config.IntervalValue, entry.Config.IntervalUnit)
		return dur > 0 && now.Sub(entry.LastRun) >= dur
	case "daemon":
		backoff := calcBackoff(entry.FailCount)
		return now.Sub(entry.LastRun) >= backoff
	case "once":
		return entry.LastRun.IsZero()
	}
	return false
}

func (se *scheduleEngineImpl) onTrigger(entry *ScheduleEntry) {
	auth := loadTaskAuth(entry.ChatID)
	_, err := Run(se.ctx, auth, entry.ChatID, &RunReq{})

	se.mu.Lock()
	entry.Running = false
	entry.LastRun = time.Now()
	if err != nil {
		entry.FailCount++
	} else {
		entry.FailCount = 0
	}
	se.mu.Unlock()

	if err != nil {
		log.Warn("schedule trigger failed for %s: %v", entry.ChatID, err)
	}
}

func loadScheduledTasks() []*ScheduleEntry {
	rows, err := capsule.Global.Query().Table(tableTaskConfig()).
		Select("chat_id", "schedule").
		WhereNotNull("schedule").
		Get()
	if err != nil {
		return nil
	}

	var entries []*ScheduleEntry
	for _, row := range rows {
		chatID := getString(row, "chat_id")
		if chatID == "" {
			continue
		}
		entries = append(entries, &ScheduleEntry{
			ChatID: chatID,
			Config: ScheduleConfig{Enabled: true, Mode: "interval"},
		})
	}
	return entries
}

func loadTaskAuth(chatID string) *process.AuthorizedInfo {
	row, err := capsule.Global.Query().Table(tableTask()).
		Select("__yao_created_by", "__yao_team_id").
		Where("chat_id", "=", chatID).
		First()
	if err != nil || row == nil {
		return &process.AuthorizedInfo{}
	}
	return &process.AuthorizedInfo{
		UserID: getString(row, "__yao_created_by"),
		TeamID: getString(row, "__yao_team_id"),
	}
}

func matchesTime(times []string, now time.Time) bool {
	nowStr := now.Format("15:04")
	for _, t := range times {
		if t == nowStr {
			return true
		}
	}
	return false
}

func triggeredThisMinute(entry *ScheduleEntry, now time.Time) bool {
	return entry.LastRun.Truncate(time.Minute).Equal(now.Truncate(time.Minute))
}

func intervalDuration(value int, unit string) time.Duration {
	switch unit {
	case "minutes":
		return time.Duration(value) * time.Minute
	case "hours":
		return time.Duration(value) * time.Hour
	case "days":
		return time.Duration(value) * 24 * time.Hour
	}
	return 0
}

func calcBackoff(failCount int) time.Duration {
	if failCount == 0 {
		return 10 * time.Second
	}
	d := time.Duration(failCount*failCount) * 10 * time.Second
	if d > 5*time.Minute {
		d = 5 * time.Minute
	}
	return d
}
