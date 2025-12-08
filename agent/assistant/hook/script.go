package hook

import (
	"strings"

	"github.com/yaoapp/yao/agent/context"
)

// Execute execute the script
func (s *Script) Execute(ctx *context.Context, method string, args ...interface{}) (interface{}, error) {
	if s == nil || s.Script == nil {
		return nil, nil
	}

	var sid = ""
	if ctx.Authorized != nil {
		sid = ctx.Authorized.SessionID
	}

	scriptCtx, err := s.NewContext(sid, nil)
	if err != nil {
		return nil, err
	}
	defer scriptCtx.Close()

	// Set authorized information if available
	if ctx.Authorized != nil {
		scriptCtx.WithAuthorized(ctx.Authorized.AuthorizedToMap())
	}

	// The first argument is the context
	args = append([]interface{}{ctx}, args...)

	// Try to call the method
	result, err := scriptCtx.CallWith(ctx.Context, method, args...)

	// If method doesn't exist (ReferenceError or similar), return nil without error
	if err != nil && (strings.Contains(err.Error(), "is not defined") ||
		strings.Contains(err.Error(), "is not a function") ||
		strings.Contains(err.Error(), "is not a Function")) {
		return nil, nil
	}

	return result, err
}
