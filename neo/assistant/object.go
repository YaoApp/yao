package assistant

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
	chatMessage "github.com/yaoapp/yao/neo/message"
	"rogchap.com/v8go"
)

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
	v8ctx.WithGlobal("__yao_agent_global", &GlobalVariables{
		Assistant:   ast,
		Contents:    contents,
		GinContext:  c,
		ChatContext: context,
	})

	// Add assistant to the script context
	v8ctx.WithGlobal("assistant", ast.Map())
	v8ctx.WithGlobal("context", context.Map())

	// Add methods to the script contexts
	v8ctx.WithFunction("Plan", jsPlan) // Create a new plan object
	v8ctx.WithFunction("Send", jsSend)
	v8ctx.WithFunction("Call", jsCall)
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

func jsCall(info *v8go.FunctionCallbackInfo) *v8go.Value {

	// Get the args
	args := info.Args()
	if len(args) < 2 {
		return bridge.JsException(info.Context(), "Run requires at least two arguments")
	}

	// Get the assistant id
	assistantID := args[0].String()

	// Get the assistant
	newAst, err := Get(assistantID)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	// Get the input
	input := args[1].String()

	// Get the global variables
	global, err := global(info)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	// Update Context
	chatContext := global.ChatContext
	chatContext.AssistantID = assistantID
	chatContext.ChatID = fmt.Sprintf("chat_%s", uuid.New().String()) // New chat id
	chatContext.Silent = true                                        // Silent mode

	var cb func(msg *chatMessage.Message)
	if len(args) > 2 {

		// Parse the callback
		funcType := "method"
		name := ""
		userArgs := []interface{}{}
		if args[2].IsFunction() {
			funcType = "anonymous"
		} else {
			goValue, err := bridge.GoValue(args[2], info.Context())
			if err != nil {
				return bridge.JsException(info.Context(), err.Error())
			}
			switch v := goValue.(type) {
			case string:
				name = v
			case map[string]interface{}:
				if fname, ok := v["name"].(string); ok {
					name = fname
				}
				if args, ok := v["args"].([]interface{}); ok {
					userArgs = args
				}
			}

			if strings.Contains(name, ".") {
				funcType = "process"
			}
		}

		switch funcType {
		case "anonymous":
			source := args[2].String()
			cb = func(msg *chatMessage.Message) {
				cbArgs := []interface{}{msg}
				ctx, err := global.Assistant.Script.NewContext(global.ChatContext.Sid, nil)
				if err != nil {
					fmt.Println("Failed to create context", err.Error())
					return
				}
				defer ctx.Close()

				global.Assistant.InitObject(ctx, global.GinContext, chatContext, global.Contents)
				_, err = ctx.CallAnonymousWith(context.Background(), source, cbArgs...)
				if err != nil {
					log.Error("Failed to call the method: %s", err.Error())
					color.Red("Failed to call the method: %s", err.Error())
					return
				}
			}
			break

		case "process":

			cb = func(msg *chatMessage.Message) {
				cbArgs := []interface{}{}
				cbArgs = append(cbArgs, msg)
				cbArgs = append(cbArgs, userArgs...)
				p, err := process.Of(name, cbArgs...)
				if err != nil {
					log.Error("Failed to get the process: %s", err.Error())
					color.Red("Failed to get the process: %s", err.Error())
					return
				}
				err = p.Execute()
				if err != nil {
					log.Error("Failed to execute the process: %s", err.Error())
					color.Red("Failed to execute the process: %s", err.Error())
					return
				}
				defer p.Release()
			}

		case "method":

			cb = func(msg *chatMessage.Message) {
				cbArgs := []interface{}{}
				cbArgs = append(cbArgs, msg)
				cbArgs = append(cbArgs, userArgs...)
				ctx, err := global.Assistant.Script.NewContext(global.ChatContext.Sid, nil)
				if err != nil {
					return
				}
				defer ctx.Close()

				global.Assistant.InitObject(ctx, global.GinContext, global.ChatContext, global.Contents)
				_, err = ctx.CallWith(context.Background(), name, cbArgs...)
				if err != nil {
					log.Error("Failed to call the method: %s", err.Error())
					color.Red("Failed to call the method: %s", err.Error())
					return
				}
			}
		}

	}

	// Parse the options
	options := map[string]interface{}{}
	if len(args) > 3 {
		optionsRaw, err := bridge.GoValue(args[3], info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), err.Error())
		}

		// Parse the options
		if optionsRaw != nil {
			switch v := optionsRaw.(type) {
			case string:
				err := jsoniter.UnmarshalFromString(v, &options)
				if err != nil {
					return bridge.JsException(info.Context(), err.Error())
				}
			case map[string]interface{}:
				options = v
			default:
				return bridge.JsException(info.Context(), "Invalid options")
			}
		}
	}

	err = newAst.Execute(global.GinContext, chatContext, input, options, cb) // Execute the assistant
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}
	return nil
}

// global get the global variables
func global(info *v8go.FunctionCallbackInfo) (global *GlobalVariables, err error) {

	jsGlobal, err := info.This().Get("__yao_agent_global")
	if err != nil {
		return nil, err
	}

	// Convert to go interface
	goGlobal, err := bridge.GoValue(jsGlobal, info.Context())
	if err != nil {
		return nil, err
	}

	global, ok := goGlobal.(*GlobalVariables)
	if !ok {
		return nil, fmt.Errorf("global is not a valid GlobalVariables")
	}

	return global, nil
}
