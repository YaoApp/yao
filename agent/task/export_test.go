package task

import "time"

var (
	// Table name exports for integration tests
	ExportTableTask = tableTask
	ExportTableChat = tableChat

	// Plan 1 exports
	ExportRowToTask        = rowToTask
	ExportMetaString       = metaString
	ExportGetString        = getString
	ExportGetStringDefault = getStringDefault
	ExportGetStringPtr     = getStringPtr
	ExportGetInt           = getInt
	ExportGetBool          = getBool
	ExportGetTime          = getTime

	// Plan 2 exports
	ExportConfigReqToMap = configReqToMap

	// Plan 3 exports - daemon
	ExportNewDaemonContext        = newDaemonContext
	ExportNewDaemonResponseWriter = NewDaemonResponseWriter

	// Plan 3 exports - schedule
	ExportCalcBackoff      = calcBackoff
	ExportIntervalDuration = intervalDuration
	ExportMatchesTime      = matchesTime

	// Plan 3 exports - extract/enrich
	ExportCleanMarkdownFences = cleanMarkdownFences
	ExportIsValidPriority     = isValidPriority
	ExportIsValidMailPriority = isValidMailPriority
	ExportExtractRecentText   = extractRecentText

	// Plan 3 exports - run core logic
	ExportInputToAgentMessages = inputToAgentMessages
	ExportGetStringVal         = getStringVal
	ExportToOAuthInfo          = toOAuthInfo
	ExportContentText          = contentText

	// Watch/WatchStream exports
	ExportGetOriginalPrompt       = GetOriginalPrompt
	ExportExtractContentFromProps = extractContentFromProps

	// Plan 3 exports - daemon registry
	ExportGetDaemon        = GetDaemon
	ExportUnregisterDaemon = UnregisterDaemon

	// enrichTaskResult exports
	ExportBuildEnrichResultPrompt = buildEnrichResultPrompt
)

// ExportRegisterDaemon stores a DaemonContext in the global registry for testing
func ExportRegisterDaemon(chatID string, dc *DaemonContext) {
	daemonRegistry.Store(chatID, dc)
}

// ExportShouldTrigger wraps scheduleEngineImpl.shouldTrigger for external testing
func ExportShouldTrigger(entry *ScheduleEntry, now time.Time) bool {
	se := &scheduleEngineImpl{entries: make(map[string]*ScheduleEntry)}
	return se.shouldTrigger(entry, now)
}

// NewTestQuotaManager creates a QuotaManager with a fixed default limit for testing.
func NewTestQuotaManager(limit int) *QuotaManager {
	qm := &QuotaManager{
		running: make(map[string]int),
		queue:   make(map[string]*priorityQueue),
		limits:  make(map[string]int),
	}
	qm.limits["team1"] = limit
	qm.limits["new-team"] = limit
	return qm
}

// QueueEntry is a test-only wrapper around queueEntry to expose it to external tests.
type QueueEntry struct{ inner *queueEntry }

// Ready returns the channel that's closed when the entry is ready to run.
func (e *QueueEntry) Ready() <-chan struct{} { return e.inner.ready }

// ExportEnqueue wraps Enqueue and returns the exported QueueEntry type.
func ExportEnqueue(qm *QuotaManager, teamID, chatID string, priority int) *QueueEntry {
	return &QueueEntry{inner: qm.Enqueue(teamID, chatID, priority)}
}

// NewTestQuotaManagerNoLimits creates a QuotaManager with NO team-specific limits,
// so resolveLimit falls back to defaultQuotaLimit.
func NewTestQuotaManagerNoLimits() *QuotaManager {
	return &QuotaManager{
		running: make(map[string]int),
		queue:   make(map[string]*priorityQueue),
		limits:  make(map[string]int),
	}
}

// ExportNewScheduleEngine exposes NewScheduleEngine for integration tests.
var ExportNewScheduleEngine = NewScheduleEngine

// ExportLoadMessagesFromDB exposes loadMessagesFromDB for integration tests.
var ExportLoadMessagesFromDB = loadMessagesFromDB
