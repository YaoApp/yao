package test

import (
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/sui/core"
	"rogchap.com/v8go"
)

// SUITestContext provides the test execution context for SUI backend scripts.
// It wraps a loaded Script and a mock Request, exposing call/callWithRequest
// to JS test functions.
type SUITestContext struct {
	Script  *core.Script
	Request *core.Request
	Prefix  string
	Sid     string
}

// NewSUITestContext creates a new SUI test context for a page
func NewSUITestContext(script *core.Script, prefix string, sid string) *SUITestContext {
	return &SUITestContext{
		Script: script,
		Prefix: prefix,
		Sid:    sid,
		Request: &core.Request{
			Method:  "GET",
			Sid:     sid,
			Payload: map[string]interface{}{},
			Params:  map[string]string{},
		},
	}
}

// Call invokes an Api-prefixed method on the backend script (sui.Run path).
// The JS function is looked up as <Prefix><method> (e.g. "ApiGetDashboard").
func (ctx *SUITestContext) Call(method string, args ...interface{}) (interface{}, error) {
	scriptCtx, err := ctx.Script.NewContext(ctx.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer scriptCtx.Close()

	if ctx.Request.Authorized != nil {
		scriptCtx.WithAuthorized(ctx.Request.Authorized)
	}

	fnName := ctx.Prefix + method
	global := scriptCtx.Global()
	if !global.Has(fnName) {
		return nil, fmt.Errorf("method %s not found (looked for %s)", method, fnName)
	}

	return scriptCtx.Call(fnName, args...)
}

// CallWithRequest invokes a method on the backend script, appending *Request
// as the last argument (page-render @Method path).
func (ctx *SUITestContext) CallWithRequest(method string, args ...interface{}) (interface{}, error) {
	return ctx.Script.Call(ctx.Request, method, args...)
}

// NewSUITestContextObject creates a JavaScript object exposing the SUITestContext to V8
func NewSUITestContextObject(v8ctx *v8go.Context, ctx *SUITestContext) (*v8go.Value, error) {
	iso := v8ctx.Isolate()

	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("call", ctx.callMethod(iso, v8ctx))
	tmpl.Set("callWithRequest", ctx.callWithRequestMethod(iso, v8ctx))
	tmpl.Set("setAuthorized", ctx.setAuthorizedMethod(iso, v8ctx))
	tmpl.Set("reset", ctx.resetMethod(iso))

	instance, err := tmpl.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	obj, err := instance.Value.AsObject()
	if err != nil {
		return nil, err
	}

	reqObj, err := ctx.buildRequestObject(v8ctx)
	if err != nil {
		return nil, err
	}
	obj.Set("request", reqObj)

	return instance.Value, nil
}

// callMethod implements ctx.call(method, ...args) in JS
func (ctx *SUITestContext) callMethod(iso *v8go.Isolate, v8ctx *v8go.Context) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		jsArgs := info.Args()
		if len(jsArgs) < 1 {
			throwJSError(v8ctx, "call requires at least 1 argument (method name)")
			return v8go.Undefined(iso)
		}

		method := jsArgs[0].String()
		goArgs := make([]interface{}, 0, len(jsArgs)-1)
		for _, arg := range jsArgs[1:] {
			val, err := bridge.GoValue(arg, v8ctx)
			if err != nil {
				goArgs = append(goArgs, arg.String())
				continue
			}
			goArgs = append(goArgs, val)
		}

		result, err := ctx.Call(method, goArgs...)
		if err != nil {
			return bridge.JsException(v8ctx, err)
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err)
		}
		return jsVal
	})
}

// callWithRequestMethod implements ctx.callWithRequest(method, ...args) in JS
func (ctx *SUITestContext) callWithRequestMethod(iso *v8go.Isolate, v8ctx *v8go.Context) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		jsArgs := info.Args()
		if len(jsArgs) < 1 {
			throwJSError(v8ctx, "callWithRequest requires at least 1 argument (method name)")
			return v8go.Undefined(iso)
		}

		method := jsArgs[0].String()
		goArgs := make([]interface{}, 0, len(jsArgs)-1)
		for _, arg := range jsArgs[1:] {
			val, err := bridge.GoValue(arg, v8ctx)
			if err != nil {
				goArgs = append(goArgs, arg.String())
				continue
			}
			goArgs = append(goArgs, val)
		}

		result, err := ctx.CallWithRequest(method, goArgs...)
		if err != nil {
			return bridge.JsException(v8ctx, err)
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err)
		}
		return jsVal
	})
}

// setAuthorizedMethod implements ctx.setAuthorized(auth) in JS
func (ctx *SUITestContext) setAuthorizedMethod(iso *v8go.Isolate, v8ctx *v8go.Context) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		jsArgs := info.Args()
		if len(jsArgs) < 1 {
			return v8go.Undefined(iso)
		}

		val, err := bridge.GoValue(jsArgs[0], v8ctx)
		if err != nil {
			return v8go.Undefined(iso)
		}

		if authMap, ok := val.(map[string]interface{}); ok {
			ctx.Request.Authorized = authMap
		}
		return v8go.Undefined(iso)
	})
}

// resetMethod implements ctx.reset() in JS — clears authorized, payload, params
func (ctx *SUITestContext) resetMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx.Request.Authorized = nil
		ctx.Request.Payload = map[string]interface{}{}
		ctx.Request.Params = map[string]string{}
		ctx.Request.Query = nil
		ctx.Request.Headers = nil
		ctx.Request.Body = nil
		return v8go.Undefined(iso)
	})
}

// buildRequestObject creates a JS object representing ctx.request
func (ctx *SUITestContext) buildRequestObject(v8ctx *v8go.Context) (*v8go.Value, error) {
	reqMap := map[string]interface{}{
		"sid":        ctx.Request.Sid,
		"method":     ctx.Request.Method,
		"payload":    ctx.Request.Payload,
		"params":     ctx.Request.Params,
		"authorized": ctx.Request.Authorized,
	}
	return bridge.JsValue(v8ctx, reqMap)
}

func throwJSError(v8ctx *v8go.Context, msg string) {
	bridge.JsException(v8ctx, fmt.Errorf("%s", msg))
}
