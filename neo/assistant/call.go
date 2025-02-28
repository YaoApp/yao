package assistant

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	chatctx "github.com/yaoapp/yao/neo/context"
	chatMessage "github.com/yaoapp/yao/neo/message"
	"rogchap.com/v8go"
)

// objectCall is the object for the call function
type objectCall struct{}

// OptionsCall is the options for the call function
type OptionsCall struct {
	Retry   OptionsCallRetry       `json:"retry,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// OptionsCallRetry is the retry options for the call function
type OptionsCallRetry struct {
	Times    int    `json:"times,omitempty"`
	Delay    int    `json:"delay,omitempty"`
	DelayMax int    `json:"delay_max,omitempty"`
	Prompt   string `json:"prompt,omitempty"`
}

// allowedEvents is the allowed events for the call function
var allowedEvents = map[string]bool{
	"done":    true,
	"retry":   true,
	"error":   true,
	"message": true,
}

var callProps = []string{
	"assistant_id",
	"input",
	"options",
}

// jsNewPlan create a plan object and return it
func jsCall(info *v8go.FunctionCallbackInfo) *v8go.Value {

	args := info.Args()
	if len(args) < 2 {
		return bridge.JsException(info.Context(), "Run requires at least two arguments")
	}

	options := v8go.Undefined(info.Context().Isolate())
	if len(args) > 2 {
		options = args[2]
	}

	// Export the object
	obj := &objectCall{}
	objectTmpl := obj.ExportObject(info)
	this, err := objectTmpl.NewInstance(info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	// Copy global properties
	global := info.This()
	for _, prop := range objectProperties {
		if !global.Has(prop) {
			continue
		}
		value, err := global.Get(prop)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get property %s: %s", prop, err.Error()))
		}
		this.Set(prop, value)
	}

	this.Set("assistant_id", args[0])
	this.Set("input", args[1])
	this.Set("options", options)
	return this.Value
}

// ExportObject Export as a FS Object
func (obj *objectCall) ExportObject(info *v8go.FunctionCallbackInfo) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(info.Context().Isolate())
	tmpl.Set("On", v8go.NewFunctionTemplate(info.Context().Isolate(), obj.on))   // On the call
	tmpl.Set("Run", v8go.NewFunctionTemplate(info.Context().Isolate(), obj.run)) // Run the call
	return tmpl
}

// on bind the callback to the call object
func (obj *objectCall) on(info *v8go.FunctionCallbackInfo) *v8go.Value {

	args := info.Args()
	if len(args) < 2 {
		return bridge.JsException(info.Context(), "On requires at least one argument")
	}

	if !args[0].IsString() {
		return bridge.JsException(info.Context(), "The first argument should be a string")
	}

	name := args[0].String()
	if !allowedEvents[name] {
		return bridge.JsException(info.Context(), fmt.Sprintf("Invalid event %s", name))
	}

	cb := args[1]
	if !cb.IsFunction() {
		return bridge.JsException(info.Context(), fmt.Sprintf("The second argument should be a function for event %s", name))
	}

	this := info.This()
	this.Set(fmt.Sprintf("on_%s", name), cb)
	return this.Value
}

// run run the call
func (obj *objectCall) run(info *v8go.FunctionCallbackInfo) *v8go.Value {

	this := info.This()
	args := info.Args()

	global, err := getGlobal(info.Context(), this)
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	goArgs := []interface{}{}
	jsArgs := []v8go.Valuer{}
	if len(args) > 0 {
		for _, arg := range args {
			v, err := bridge.GoValue(arg, info.Context())
			if err != nil {
				return bridge.JsException(info.Context(), err.Error())
			}
			goArgs = append(goArgs, v)
			jsArgs = append(jsArgs, arg)
		}
	}

	// Get the assistant id
	jsAssistantID, err := this.Get("assistant_id")
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	assistantID := jsAssistantID.String()

	// Get the input
	jsInput, err := this.Get("input")
	if err != nil {
		return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the input: %s", err.Error()))
	}

	input, err := bridge.GoValue(jsInput, info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), fmt.Sprintf("Failed to unmarshal the input: %s", err.Error()))
	}

	// Options
	options := OptionsCall{
		Retry: OptionsCallRetry{
			Times:    3,
			Delay:    200,
			DelayMax: 5000,
			Prompt:   "Please fix the error. \n {{ error }}",
		},
		Options: map[string]interface{}{},
	}

	// Get the options
	if this.Has("options") {
		jsOptions, err := this.Get("options")
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the options: %s", err.Error()))
		}

		err = bridge.Unmarshal(jsOptions, &options)
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("Failed to unmarshal the options: %s", err.Error()))
		}
	}

	// Get the assistant
	newAst, err := Get(assistantID)
	if err != nil {
		return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the assistant: %s", err.Error()))
	}

	// Get the message event ( it will be used for the message event )
	eventMessage := ""
	goCallProps := map[string]interface{}{}
	if this.Has("on_message") {
		jsEventMessage, err := this.Get("on_message")
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the message: %s", err.Error()))
		}
		eventMessage = jsEventMessage.String()

		for _, prop := range callProps {
			if this.Has(prop) {
				value, err := this.Get(prop)
				if err != nil {
					return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the %s property: %s", prop, err.Error()))
				}
				goValue, err := bridge.GoValue(value, info.Context())
				if err != nil {
					return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the %s property: %s", prop, err.Error()))
				}
				goCallProps[prop] = goValue
			}
		}
	}

	// Update the chat context
	var chatCtx chatctx.Context = global.ChatContext
	chatCtx.AssistantID = assistantID
	chatCtx.ChatID = fmt.Sprintf("chat_%s", uuid.New().String()) // New chat id
	chatCtx.Silent = true

	// Define the callback function
	var cb func(msg *chatMessage.Message) = nil
	var output = []chatMessage.Message{}
	cb = func(msg *chatMessage.Message) {
		output = append(output, *msg)
		if eventMessage != "" {
			err := obj.triggerAnonymous(chatCtx, global, goCallProps, eventMessage, goArgs, msg)
			if err != nil {
				color.Red("Failed to trigger the message event: %s", err.Error())
				log.Error("Failed to trigger the message event: %s", err.Error())
				return
			}
		}
	}

	// Execute the assistant
	err = newAst.Execute(global.GinContext, chatCtx, input, options.Options, cb) // Execute the assistant
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	// Copy props
	for name, value := range goCallProps {
		info.Context().Global().Set(name, value)
	}

	// Trigger the done event
	exception := obj.trigger(info, "done", jsArgs...)
	if exception != nil {
		return exception
	}

	return nil
}

func (obj *objectCall) triggerAnonymous(chatCtx chatctx.Context, global *GlobalVariables, goCallProps map[string]interface{}, source string, bindArgs []interface{}, fnArgs ...interface{}) error {

	ctx, err := global.Assistant.Script.NewContext(global.ChatContext.Sid, nil)
	if err != nil {
		return err
	}
	defer ctx.Close()

	// Update Context
	global.Assistant.InitObject(ctx, global.GinContext, chatCtx, global.Contents)

	// Copy props
	for k, v := range goCallProps {
		ctx.WithGlobal(k, v)
	}

	// Add the args
	ctx.WithGlobal("args", bindArgs)
	_, err = ctx.CallAnonymousWith(context.Background(), source, fnArgs...)
	if err != nil {
		return err
	}
	return nil

}

// trigger trigger the callback
func (obj *objectCall) trigger(info *v8go.FunctionCallbackInfo, name string, fnArgs ...v8go.Valuer) *v8go.Value {
	// Try to get the callback
	this := info.This()
	if this.Has(fmt.Sprintf("on_%s", name)) {
		event, err := this.Get(fmt.Sprintf("on_%s", name))
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the %s callback: %s	", name, err.Error()))
		}

		if event.IsFunction() {

			cb, err := event.AsFunction()
			if err != nil {
				return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the %s callback: %s", name, err.Error()))
			}

			result, err := cb.Call(this, fnArgs...)
			if err != nil {
				return bridge.JsException(info.Context(), fmt.Sprintf("Failed to trigger the %s callback: %s", name, err.Error()))
			}
			return result
		}
	}

	return nil
}

// jsCallBackup is the backup function for the call function
// func jsCallBackup(info *v8go.FunctionCallbackInfo) *v8go.Value {

// 	// Get the args
// 	args := info.Args()
// 	if len(args) < 2 {
// 		return bridge.JsException(info.Context(), "Run requires at least two arguments")
// 	}

// 	// Get the assistant id
// 	assistantID := args[0].String()

// 	// Get the assistant
// 	newAst, err := Get(assistantID)
// 	if err != nil {
// 		return bridge.JsException(info.Context(), err.Error())
// 	}

// 	// Get the input
// 	input := args[1].String()

// 	// Get the global variables
// 	global, err := global(info)
// 	if err != nil {
// 		return bridge.JsException(info.Context(), err.Error())
// 	}

// 	// Update Context
// 	chatContext := global.ChatContext
// 	chatContext.AssistantID = assistantID
// 	chatContext.ChatID = fmt.Sprintf("chat_%s", uuid.New().String()) // New chat id
// 	chatContext.Silent = true                                        // Silent mode

// 	var cb func(msg *chatMessage.Message) = nil
// 	if len(args) > 2 {

// 		// Rest args
// 		var jsArgs *v8go.Value
// 		goArgs := []interface{}{}
// 		if len(args) > 3 {
// 			jsArgs = args[3]
// 			if jsArgs != nil {
// 				if jsArgs.IsArray() {
// 					v, err := bridge.GoValue(jsArgs, info.Context())
// 					if err != nil {
// 						return bridge.JsException(info.Context(), err.Error())
// 					}
// 					arr, ok := v.([]interface{})
// 					if !ok {
// 						return bridge.JsException(info.Context(), "Invalid arguments")
// 					}
// 					goArgs = arr
// 				} else {
// 					v, err := bridge.GoValue(jsArgs, info.Context())
// 					if err != nil {
// 						return bridge.JsException(info.Context(), err.Error())
// 					}
// 					goArgs = []interface{}{v}
// 				}
// 			}
// 		}

// 		// Parse the callback
// 		funcType := "method"
// 		name := ""
// 		userArgs := []interface{}{}
// 		if args[2].IsFunction() {
// 			funcType = "anonymous"
// 		} else {
// 			goValue, err := bridge.GoValue(args[2], info.Context())
// 			if err != nil {
// 				return bridge.JsException(info.Context(), err.Error())
// 			}
// 			switch v := goValue.(type) {
// 			case string:
// 				name = v
// 			case map[string]interface{}:
// 				if fname, ok := v["name"].(string); ok {
// 					name = fname
// 				}
// 				if args, ok := v["args"].([]interface{}); ok {
// 					userArgs = args
// 				}
// 			}

// 			if strings.Contains(name, ".") {
// 				funcType = "process"
// 			}
// 		}

// 		switch funcType {
// 		case "anonymous":
// 			source := args[2].String()
// 			cb = func(msg *chatMessage.Message) {
// 				cbArgs := []interface{}{msg}
// 				cbArgs = append(cbArgs, goArgs...)
// 				ctx, err := global.Assistant.Script.NewContext(global.ChatContext.Sid, nil)
// 				if err != nil {
// 					fmt.Println("Failed to create context", err.Error())
// 					return
// 				}
// 				defer ctx.Close()

// 				global.Assistant.InitObject(ctx, global.GinContext, chatContext, global.Contents)
// 				_, err = ctx.CallAnonymousWith(context.Background(), source, cbArgs...)
// 				if err != nil {
// 					log.Error("Failed to call the method: %s", err.Error())
// 					color.Red("Failed to call the method: %s", err.Error())
// 					return
// 				}
// 			}
// 			break

// 		case "process":

// 			cb = func(msg *chatMessage.Message) {
// 				cbArgs := []interface{}{}
// 				cbArgs = append(cbArgs, msg)
// 				cbArgs = append(cbArgs, userArgs...)
// 				p, err := process.Of(name, cbArgs...)
// 				if err != nil {
// 					log.Error("Failed to get the process: %s", err.Error())
// 					color.Red("Failed to get the process: %s", err.Error())
// 					return
// 				}
// 				err = p.Execute()
// 				if err != nil {
// 					log.Error("Failed to execute the process: %s", err.Error())
// 					color.Red("Failed to execute the process: %s", err.Error())
// 					return
// 				}
// 				defer p.Release()
// 			}

// 		case "method":

// 			cb = func(msg *chatMessage.Message) {
// 				cbArgs := []interface{}{}
// 				cbArgs = append(cbArgs, msg)
// 				cbArgs = append(cbArgs, userArgs...)
// 				ctx, err := global.Assistant.Script.NewContext(global.ChatContext.Sid, nil)
// 				if err != nil {
// 					return
// 				}
// 				defer ctx.Close()

// 				global.Assistant.InitObject(ctx, global.GinContext, global.ChatContext, global.Contents)
// 				_, err = ctx.CallWith(context.Background(), name, cbArgs...)
// 				if err != nil {
// 					log.Error("Failed to call the method: %s", err.Error())
// 					color.Red("Failed to call the method: %s", err.Error())
// 					return
// 				}
// 			}
// 		}

// 	}

// 	// Parse the options
// 	options := map[string]interface{}{}
// 	if len(args) > 4 {
// 		optionsRaw, err := bridge.GoValue(args[4], info.Context())
// 		if err != nil {
// 			return bridge.JsException(info.Context(), err.Error())
// 		}

// 		// Parse the options
// 		if optionsRaw != nil {
// 			switch v := optionsRaw.(type) {
// 			case string:
// 				err := jsoniter.UnmarshalFromString(v, &options)
// 				if err != nil {
// 					return bridge.JsException(info.Context(), err.Error())
// 				}
// 			case map[string]interface{}:
// 				options = v
// 			default:
// 				return bridge.JsException(info.Context(), "Invalid options")
// 			}
// 		}
// 	}

// 	err = newAst.Execute(global.GinContext, chatContext, input, options, cb) // Execute the assistant
// 	if err != nil {
// 		return bridge.JsException(info.Context(), err.Error())
// 	}
// 	return nil
// }
