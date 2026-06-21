package task

// GlobalScheduleEngine is a no-op stub. Plan 3 replaces it with a real engine.
var GlobalScheduleEngine ScheduleEngine = &scheduleStub{}

// ScheduleEngine defines the interface for schedule management.
type ScheduleEngine interface {
	Start() error
	Stop()
	Update(chatID string, cfg ScheduleConfig)
	Remove(chatID string)
}

type scheduleStub struct{}

func (s *scheduleStub) Start() error                             { return nil }
func (s *scheduleStub) Stop()                                    {}
func (s *scheduleStub) Update(chatID string, cfg ScheduleConfig) {}
func (s *scheduleStub) Remove(chatID string)                     {}
