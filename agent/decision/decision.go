package decision

import (
	"fmt"

	"github.com/yaoapp/gou/llm"
	"github.com/yaoapp/yao/llmprovider"
)

// Call resolves the "decision" role connector and sends a typed decision request.
// The request/response types are defined in gou/llm/decision.go.
func Call(req *llm.DecisionRequest) (*llm.DecisionResponse, error) {
	if llmprovider.Global == nil {
		return nil, fmt.Errorf("decision: llmprovider registry not initialized")
	}
	conn, err := llmprovider.Global.GetDecisionModel()
	if err != nil {
		return nil, fmt.Errorf("decision: resolve connector: %w", err)
	}
	dc, ok := conn.(llm.DecisionConnector)
	if !ok {
		return nil, fmt.Errorf("decision: connector %q does not implement DecisionConnector", conn.ID())
	}
	return dc.Decide(req)
}

// CallByUser resolves the "decision" role for a specific user.
func CallByUser(userID string, req *llm.DecisionRequest) (*llm.DecisionResponse, error) {
	if llmprovider.Global == nil {
		return nil, fmt.Errorf("decision: llmprovider registry not initialized")
	}
	conn, err := llmprovider.Global.GetDecisionModelByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("decision: resolve connector for user %s: %w", userID, err)
	}
	dc, ok := conn.(llm.DecisionConnector)
	if !ok {
		return nil, fmt.Errorf("decision: connector %q does not implement DecisionConnector", conn.ID())
	}
	return dc.Decide(req)
}
