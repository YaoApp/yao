package types

import (
	"time"

	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// BuildTriggerInput builds TriggerInput from trigger data
// Shared helper used by all executor implementations
func BuildTriggerInput(trigger robottypes.TriggerType, data interface{}) *robottypes.TriggerInput {
	input := &robottypes.TriggerInput{}

	switch trigger {
	case robottypes.TriggerClock:
		input.Clock = robottypes.NewClockContext(time.Now(), "")

	case robottypes.TriggerHuman:
		if req, ok := data.(*robottypes.InterveneRequest); ok {
			input.Action = req.Action
			input.Messages = req.Messages
		}

	case robottypes.TriggerEvent:
		if req, ok := data.(*robottypes.EventRequest); ok {
			input.Source = robottypes.EventSource(req.Source)
			input.EventType = req.EventType
			input.Data = req.Data
		}
	}

	return input
}
