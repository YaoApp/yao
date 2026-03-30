package standard

import (
	"encoding/json"
	"fmt"

	kunlog "github.com/yaoapp/kun/log"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// CallHostAgent calls the Host Agent with structured input and parses structured output.
// The Host Agent mediates all human-robot interactions through three scenarios:
//   - "assign": new task assignment with multi-round confirmation
//   - "guide": guidance during execution
//   - "clarify": answering questions from waiting tasks
func (e *Executor) CallHostAgent(ctx *robottypes.Context, robot *robottypes.Robot, input *robottypes.HostInput, chatID string) (*robottypes.HostOutput, error) {
	if robot == nil {
		return nil, fmt.Errorf("robot cannot be nil")
	}

	// Get agent ID for host phase (per-robot config > global Uses > empty)
	agentID := robottypes.ResolvePhaseAgent(robot.Config, robottypes.PhaseHost)
	if agentID == "" {
		return nil, fmt.Errorf("no Host Agent configured for robot %s (set uses.host in agent.yml or resources.phases in robot config)", robot.MemberID)
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal host input: %w", err)
	}

	kunlog.Info("calling Host Agent %s for scenario=%s chatID=%s", agentID, input.Scenario, chatID)

	caller := NewConversationCaller(chatID)
	caller.Connector = robot.LanguageModel
	caller.Workspace = robot.Workspace
	result, err := caller.CallWithMessages(ctx, agentID, string(inputJSON))
	if err != nil {
		return nil, fmt.Errorf("host agent (%s) call failed: %w", agentID, err)
	}

	data, err := result.GetJSON()
	if err != nil {
		text := result.GetText()
		kunlog.Warn("Host Agent returned non-JSON response, treating as confirm: %s", text)
		return &robottypes.HostOutput{
			Reply:  text,
			Action: robottypes.HostActionConfirm,
		}, nil
	}

	output := &robottypes.HostOutput{}
	raw, _ := json.Marshal(data)
	if err := json.Unmarshal(raw, output); err != nil {
		return &robottypes.HostOutput{
			Reply:  result.GetText(),
			Action: robottypes.HostActionConfirm,
		}, nil
	}

	return output, nil
}
