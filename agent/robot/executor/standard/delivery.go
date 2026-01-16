package standard

import (
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunDelivery executes P4: Delivery phase
// Delivers results to configured targets
//
// Input:
//   - TaskResults (from P3)
//   - Delivery config from robot
//
// Output:
//   - DeliveryResult with success status
//
// Delivery Types:
//   - DeliveryEmail: Send email
//   - DeliveryNotify: Send notification
//   - DeliveryWebhook: Call webhook
//   - DeliveryStore: Store to database
//
// TODO: Implement real delivery
func (e *Executor) RunDelivery(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	e.simulateStreamDelay()

	exec.Delivery = &robottypes.DeliveryResult{
		Type:    robottypes.DeliveryNotify,
		Success: true,
	}
	return nil
}
