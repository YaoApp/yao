package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/session"
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

// UserID get the user id from the session
func (agent *DSL) UserID(sid string) (interface{}, error) {
	fieldID := agent.AuthSetting.SessionFields.ID
	return session.Global().ID(sid).Get(fieldID)
}

// GuestID get the guest id from the session
func (agent *DSL) GuestID(sid string) (interface{}, error) {
	fieldGuest := agent.AuthSetting.SessionFields.Guest
	return session.Global().ID(sid).Get(fieldGuest)
}

// UserRoles get the user roles from the session
func (agent *DSL) UserRoles(sid string) (interface{}, error) {
	fieldRoles := agent.AuthSetting.SessionFields.Roles
	return session.Global().ID(sid).Get(fieldRoles)
}

// UserOrGuestID get the user id or guest id from the session
func (agent *DSL) UserOrGuestID(sid string) (interface{}, bool, error) {
	userID, err := agent.UserID(sid)
	if err != nil {
		guestID, err := agent.GuestID(sid)
		if err != nil {
			return nil, false, err
		}
		return guestID, true, nil
	}
	return userID, false, nil
}
