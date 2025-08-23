package neo

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/yao/neo/assistant"
	chatctx "github.com/yaoapp/yao/neo/context"
)

// Answer reply the message
func (neo *DSL) Answer(ctx chatctx.Context, question string, c *gin.Context) error {
	var err error
	var ast assistant.API = Neo.Assistant
	if ctx.AssistantID != "" {
		ast, err = neo.Select(ctx.AssistantID)
		if err != nil {
			return err
		}
	}
	_, err = ast.Execute(c, ctx, question, nil)
	return err
}

// Select select an assistant
func (neo *DSL) Select(id string) (assistant.API, error) {
	if id == "" {
		return Neo.Assistant, nil
	}
	return assistant.Get(id)
}

// UserID get the user id from the session
func (neo *DSL) UserID(sid string) (interface{}, error) {
	fieldID := neo.AuthSetting.SessionFields.ID
	return session.Global().ID(sid).Get(fieldID)
}

// GuestID get the guest id from the session
func (neo *DSL) GuestID(sid string) (interface{}, error) {
	fieldGuest := neo.AuthSetting.SessionFields.Guest
	return session.Global().ID(sid).Get(fieldGuest)
}

// UserRoles get the user roles from the session
func (neo *DSL) UserRoles(sid string) (interface{}, error) {
	fieldRoles := neo.AuthSetting.SessionFields.Roles
	return session.Global().ID(sid).Get(fieldRoles)
}

// UserOrGuestID get the user id or guest id from the session
func (neo *DSL) UserOrGuestID(sid string) (interface{}, bool, error) {
	userID, err := neo.UserID(sid)
	if err != nil {
		guestID, err := neo.GuestID(sid)
		if err != nil {
			return nil, false, err
		}
		return guestID, true, nil
	}
	return userID, false, nil
}
