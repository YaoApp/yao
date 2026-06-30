package task

import (
	"context"
	"encoding/json"
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
	stopOnce   sync.Once
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

	resetOrphanedScheduledTasks()

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

// resetOrphanedScheduledTasks resets scheduled tasks that are stuck in "running"/"queued"
// due to a server crash or restart. Only affects tasks that have a schedule configured.
func resetOrphanedScheduledTasks() {
	if capsule.Global == nil {
		return
	}
	now := time.Now()
	_, err := capsule.Global.Query().Table(tableTask()).
		WhereIn("run_status", []interface{}{"running", "queued"}).
		WhereNotNull("schedule").
		WhereNull("deleted_at").
		Update(map[string]interface{}{
			"run_status":    "failed",
			"error_message": "server restarted while task was running",
			"completed_at":  now,
			"updated_at":    now,
		})
	if err != nil {
		log.Warn("resetOrphanedScheduledTasks: %v", err)
	}
}

func (se *scheduleEngineImpl) Stop() {
	se.stopOnce.Do(func() {
		if se.tickerDone != nil {
			close(se.tickerDone)
		}
		if se.cancel != nil {
			se.cancel()
		}
	})
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
	if isTaskRunning(entry.ChatID) {
		se.mu.Lock()
		entry.Running = false
		se.mu.Unlock()
		return
	}

	writeScheduleLog(entry.ChatID)
	auth := loadTaskAuth(entry.ChatID)

	var promptContent interface{}
	var locale string
	if si := getScheduledInstruction(entry.ChatID); si != nil && si.Prompt != "" {
		promptContent = si.Prompt
		locale = si.Locale
	} else {
		promptContent = GetOriginalPrompt(se.ctx, entry.ChatID)
	}
	if promptContent == nil || promptContent == "" {
		log.Warn("schedule trigger skipped for %s: no instruction or original prompt", entry.ChatID)
		se.mu.Lock()
		entry.Running = false
		se.mu.Unlock()
		return
	}

	_, err := Run(se.ctx, auth, entry.ChatID, &RunReq{
		Messages: []InputMessage{{Role: "user", Content: promptContent}},
		Source:   "repeat",
		Locale:   locale,
	})

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

// getScheduledInstruction reads and parses the instruction JSON from agent_task table.
func getScheduledInstruction(chatID string) *ScheduledInstruction {
	row, err := capsule.Global.Query().Table(tableTask()).
		Select("instruction").
		Where("chat_id", "=", chatID).
		First()
	if err != nil || row == nil {
		return nil
	}
	return parseInstructionJSON(row["instruction"])
}

// parseInstructionJSON deserializes the instruction column value into ScheduledInstruction.
func parseInstructionJSON(raw interface{}) *ScheduledInstruction {
	if raw == nil {
		return nil
	}
	var data []byte
	switch v := raw.(type) {
	case string:
		if v == "" {
			return nil
		}
		data = []byte(v)
	case []byte:
		data = v
	default:
		d, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		data = d
	}
	var si ScheduledInstruction
	if err := json.Unmarshal(data, &si); err != nil {
		return nil
	}
	return &si
}

func loadScheduledTasks() []*ScheduleEntry {
	if capsule.Global == nil {
		return nil
	}
	rows, err := capsule.Global.Query().Table(tableTask()).
		Select("chat_id", "schedule").
		WhereNotNull("schedule").
		WhereNull("deleted_at").
		Get()
	if err != nil {
		log.Warn("loadScheduledTasks query error: %v", err)
		return nil
	}
	var entries []*ScheduleEntry
	for _, row := range rows {
		chatID := getString(row, "chat_id")
		if chatID == "" {
			continue
		}

		cfg := parseScheduleConfig(row["schedule"])
		if !cfg.Enabled {
			continue
		}

		lastRun := getLastTriggeredAt(chatID)
		entries = append(entries, &ScheduleEntry{
			ChatID:  chatID,
			Config:  cfg,
			LastRun: lastRun,
		})
	}
	return entries
}

func parseScheduleConfig(raw interface{}) ScheduleConfig {
	var cfg ScheduleConfig
	if raw == nil {
		return cfg
	}
	switch v := raw.(type) {
	case string:
		if v == "" {
			return cfg
		}
		json.Unmarshal([]byte(v), &cfg)
	case []byte:
		json.Unmarshal(v, &cfg)
	default:
		b, _ := json.Marshal(raw)
		json.Unmarshal(b, &cfg)
	}
	return cfg
}

func getLastTriggeredAt(chatID string) time.Time {
	row, err := capsule.Global.Query().Table(tableScheduleLog()).
		Select("triggered_at").
		Where("chat_id", "=", chatID).
		OrderByDesc("triggered_at").
		First()
	if err != nil || row == nil {
		return time.Time{}
	}
	if t, ok := row["triggered_at"].(time.Time); ok {
		return t
	}
	if s, ok := row["triggered_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
			return t
		}
	}
	return time.Time{}
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
	case "m", "minutes":
		return time.Duration(value) * time.Minute
	case "h", "hours":
		return time.Duration(value) * time.Hour
	case "d", "days":
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

func isTaskRunning(chatID string) bool {
	_, exists := daemonRegistry.Load(chatID)
	return exists
}

func writeScheduleLog(chatID string) {
	err := capsule.Global.Query().Table(tableScheduleLog()).Insert(map[string]interface{}{
		"chat_id":      chatID,
		"triggered_at": time.Now(),
	})
	if err != nil {
		log.Warn("schedule log write failed for %s: %v", chatID, err)
	}
}
