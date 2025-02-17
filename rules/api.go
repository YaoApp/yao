package rules

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/helper"
)

// Guard table widget guard
func Guard(c *gin.Context, rule string) {
	sid, exists := c.Get("__sid")
	if !exists {
		abort(c, 400, "session id is not found")
		return
	}
	user, err := session.Global().ID(sid.(string)).Get("user")
	if err != nil {
		abort(c, 400, "user is not found")
		return
	}
	ruleIds := any.Of(user).MapStr().Get("rule_ids")
	ruleIds = any.Of(ruleIds).CStrings()
	if !helper.ContainsString(ruleIds.([]string), rule) && !helper.ContainsString(ruleIds.([]string), "*") {
		abort(c, 400, fmt.Sprintf("no permission for this action: %s", rule))
		return
	}
}

func abort(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"code": code, "message": message})
	c.Abort()
}
