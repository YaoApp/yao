package api

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant"
	chatctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/types"
)

// Agent the agent AI assistant
var Agent *API

// API the agent API
type API struct {
	*types.DSL
}

// Answer reply the message
func (agent *API) Answer(ctx chatctx.Context, question string, c *gin.Context) error {
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
func (agent *API) Select(id string) (assistant.API, error) {
	if id == "" {
		return Agent.Assistant, nil
	}
	return assistant.Get(id)
}
