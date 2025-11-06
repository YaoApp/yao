package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant"
	chatctx "github.com/yaoapp/yao/agent/context"
)

// Answer reply the message
func (agent *DSL) Answer(ctx chatctx.Context, question string, c *gin.Context) error {
	var err error
	var ast assistant.API = Agent.Assistant
	if ctx.AssistantID != "" {
		ast, err = agent.Select(ctx.AssistantID)
		if err != nil {
			return err
		}
	}
	_, err = ast.Execute(c, ctx, question, nil)
	return err
}

// Select select an assistant
func (agent *DSL) Select(id string) (assistant.API, error) {
	if id == "" {
		return Agent.Assistant, nil
	}
	return assistant.Get(id)
}
