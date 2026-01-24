package standard

import (
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// RunLearning executes P5: Learning phase
// Extracts learnings and saves to knowledge base
//
// Input:
//   - Execution summary (all phases)
//
// Output:
//   - LearningEntry list with extracted knowledge
//
// Learning Types:
//   - LearnExecution: Execution patterns
//   - LearnTask: Task-specific insights
//   - LearnError: Error patterns for improvement
//
// TODO: Implement real learning extraction
func (e *Executor) RunLearning(ctx *robottypes.Context, exec *robottypes.Execution, _ interface{}) error {
	// Get robot for locale
	robot := exec.GetRobot()

	// Update UI field with i18n
	locale := getEffectiveLocale(robot, exec.Input)
	e.updateUIFields(ctx, exec, "", getLocalizedMessage(locale, "learning_from_exec"))

	e.simulateStreamDelay()

	exec.Learning = []robottypes.LearningEntry{
		{
			Type:    robottypes.LearnExecution,
			Content: "Completed daily tasks successfully",
		},
	}
	return nil
}
