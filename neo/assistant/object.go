package assistant

import (
	"fmt"

	"github.com/gin-gonic/gin"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
	chatMessage "github.com/yaoapp/yao/neo/message"
	sui "github.com/yaoapp/yao/sui/core"
	"rogchap.com/v8go"
)

// objectProperties is the properties of the assistant object
var objectProperties = []string{
	"__yao_agent_global",
	"assistant",
	"context",
	"Plan",
	"Send",
	"Call",
	"Assets",
	"Set",
	"Get",
	"Del",
	"Clear",
}

// GlobalVariables is the global variables for the assistant
type GlobalVariables struct {
	Assistant   *Assistant
	Contents    *chatMessage.Contents
	GinContext  *gin.Context
	ChatContext chatctx.Context
}

// JsValue return the javascript value of the global variables
func (global *GlobalVariables) JsValue(ctx *v8go.Context) (*v8go.Value, error) {
	return v8go.NewExternal(ctx.Isolate(), global)
}

// InitObject add the global variables and methods to the script context
func (ast *Assistant) InitObject(v8ctx *v8.Context, c *gin.Context, context chatctx.Context, contents *chatMessage.Contents) {

	// Add global variables to the script context
	global := &GlobalVariables{
		Assistant:   ast,
		Contents:    contents,
		GinContext:  c,
		ChatContext: context,
	}

	// Add global variables to the script context
	v8ctx.WithGlobal("__yao_agent_global", global)

	// Add assistant to the script context
	v8ctx.WithGlobal("assistant", ast.Map())
	v8ctx.WithGlobal("context", context.Map())

	// Add methods to the script contexts
	v8ctx.WithFunction("Send", jsSend)
	v8ctx.WithFunction("Assets", jsAssets)
	v8ctx.WithFunction("MakeCall", jsCall) // Create a new call object
	v8ctx.WithFunction("MakePlan", jsPlan) // Create a new plan object

	// Shared space methods
	v8ctx.WithFunction("Set", jsSet)
	v8ctx.WithFunction("Get", jsGet)
	v8ctx.WithFunction("Del", jsDel)
	v8ctx.WithFunction("Clear", jsClear)

	// Template methods
	v8ctx.WithFunction("Replace", jsReplace)
}

// jsSet function, set a value to the shared space
func jsSet(info *v8go.FunctionCallbackInfo) *v8go.Value {
	global, err := global(info)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	if global.ChatContext.SharedSpace == nil {
		return bridge.JsException(info.Context(), "Shared space is not set")
	}

	args := info.Args()
	if len(args) < 2 {
		return bridge.JsException(info.Context(), "Set requires at least two arguments")
	}

	if !args[0].IsString() {
		return bridge.JsException(info.Context(), "Set requires a valid key")
	}

	// Validate the key
	key := args[0].String()
	if key == "" {
		return bridge.JsException(info.Context(), "Set requires a valid key")
	}

	// Validate the value
	value, err := bridge.GoValue(args[1], info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	// Set the value
	err = global.ChatContext.SharedSpace.Set(key, value)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	return nil
}

// jsGet function, get a value from the shared space
func jsGet(info *v8go.FunctionCallbackInfo) *v8go.Value {
	global, err := global(info)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	if global.ChatContext.SharedSpace == nil {
		return bridge.JsException(info.Context(), "Shared space is not set")
	}

	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Get requires at least one argument")
	}

	if !args[0].IsString() {
		return bridge.JsException(info.Context(), "Get requires a valid key")
	}

	// Get the key
	key := args[0].String()
	if key == "" {
		return bridge.JsException(info.Context(), "Get requires a valid key")
	}

	// Get the value
	value, err := global.ChatContext.SharedSpace.Get(key)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	jsValue, err := bridge.JsValue(info.Context(), value)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	return jsValue
}

// jsDel function, delete a value from the shared space
func jsDel(info *v8go.FunctionCallbackInfo) *v8go.Value {
	global, err := global(info)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	if global.ChatContext.SharedSpace == nil {
		return bridge.JsException(info.Context(), "Shared space is not set")
	}

	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Get requires at least one argument")
	}

	if !args[0].IsString() {
		return bridge.JsException(info.Context(), "Get requires a valid key")
	}

	// Get the key
	key := args[0].String()
	if key == "" {
		return bridge.JsException(info.Context(), "Get requires a valid key")
	}

	err = global.ChatContext.SharedSpace.Delete(key)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	return nil
}

func jsClear(info *v8go.FunctionCallbackInfo) *v8go.Value {
	global, err := global(info)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	if global.ChatContext.SharedSpace == nil {
		return bridge.JsException(info.Context(), "Shared space is not set")
	}

	err = global.ChatContext.SharedSpace.Clear()
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	return nil
}

// jsAssets function, get the assets content
func jsAssets(info *v8go.FunctionCallbackInfo) *v8go.Value {

	global, err := global(info)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	// Get the message
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Assets requires at least one argument")
	}

	// Get the name
	name := args[0].String()

	data := map[string]interface{}{}
	if len(args) > 1 {
		raw, err := bridge.GoValue(args[1], info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), err.Error())
		}

		v, ok := raw.(map[string]interface{})
		if !ok {
			return bridge.JsException(info.Context(), "Assets requires a map")
		}
		data = v
	}

	content, err := global.Assistant.Assets(name, data)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	jsContent, err := bridge.JsValue(info.Context(), content)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	return jsContent
}

// jsSend function, send a message to the http stream connection
func jsSend(info *v8go.FunctionCallbackInfo) *v8go.Value {

	// Get the message
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "SendMessage requires at least one argument")
	}

	input, err := bridge.GoValue(args[0], info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	global, err := global(info)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	// Save history by default
	saveHistory := true
	if len(args) > 1 && args[1].IsBoolean() {
		saveHistory = args[1].Boolean()
	}

	switch v := input.(type) {
	case string:
		// Check if the message is json
		msg, err := message.NewString(v)
		if err != nil {
			return bridge.JsException(info.Context(), err.Error())
		}

		// Append the message to the contents
		if saveHistory {
			msg.AppendTo(global.Contents)
		}
		msg.Write(global.GinContext.Writer)
		return nil

	case map[string]interface{}:
		msg := message.New().Map(v)
		if saveHistory {
			msg.AppendTo(global.Contents)
		}
		msg.Write(global.GinContext.Writer)
		return nil

	default:
		return bridge.JsException(info.Context(), "Send requires a string or a map")
	}
}

func jsReplace(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) < 2 {
		return bridge.JsException(info.Context(), "Replace requires at least two arguments")
	}

	if !args[0].IsString() {
		return bridge.JsException(info.Context(), "the first argument must be a string")
	}
	tmpl := args[0].String()

	raw, err := bridge.GoValue(args[1], info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	data, ok := raw.(map[string]interface{})
	if !ok {
		return bridge.JsException(info.Context(), "the second argument must be a map")
	}

	replaced, _ := sui.Data(data).Replace(tmpl)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	jsReplaced, err := bridge.JsValue(info.Context(), replaced)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	return jsReplaced
}

// global get the global variables
func global(info *v8go.FunctionCallbackInfo) (global *GlobalVariables, err error) {
	return getGlobal(info.Context(), info.This())
}

func getGlobal(ctx *v8go.Context, obj *v8go.Object) (global *GlobalVariables, err error) {
	jsGlobal, err := obj.Get("__yao_agent_global")
	if err != nil {
		return nil, err
	}

	// Convert to go interface
	goGlobal, err := bridge.GoValue(jsGlobal, ctx)
	if err != nil {
		return nil, err
	}

	global, ok := goGlobal.(*GlobalVariables)
	if !ok {
		return nil, fmt.Errorf("global is not a valid GlobalVariables. %#v", goGlobal)
	}

	return global, nil
}
