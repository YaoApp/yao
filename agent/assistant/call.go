package assistant

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	chatctx "github.com/yaoapp/yao/agent/context"
	chatMessage "github.com/yaoapp/yao/agent/message"
	"rogchap.com/v8go"
)

// objectCall is the object for the call function
type objectCall struct{}

// OptionsCall is the options for the call function
type OptionsCall struct {
	Retry   OptionsCallRetry       `json:"retry,omitempty"`   // Retry options
	Options map[string]interface{} `json:"options,omitempty"` // LLM API options
	Silent  bool                   `json:"silent,omitempty"`  // Silent mode, default is true
}

// OptionsCallRetry is the retry options for the call function
type OptionsCallRetry struct {
	Times    int    `json:"times,omitempty"`     // Retry times, default is 3
	Delay    int    `json:"delay,omitempty"`     // Retry delay, default is 200
	DelayMax int    `json:"delay_max,omitempty"` // Retry delay max, default is 5000
	Prompt   string `json:"prompt,omitempty"`    // Retry prompt, default is "Please fix the error. \n {{ error }}"
}

// allowedEvents is the allowed events for the call function
var allowedEvents = map[string]bool{
	"done":    true,
	"retry":   true,
	"message": true,
}

var callProps = []string{
	"assistant_id",
	"input",
	"options",
	"retry_times",
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
	this.Set("retry_times", int32(1))
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

	// Get the retry input
	if this.Has("retry_input") {

		jsRetryInput, err := this.Get("retry_input")
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the retry input: %s", err.Error()))
		}

		input, err = bridge.GoValue(jsRetryInput, info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("Failed to unmarshal the retry input: %s", err.Error()))
		}
	}

	// Options
	options := OptionsCall{
		// Retry: OptionsCallRetry{
		// 	Times:    3,
		// 	Delay:    200,
		// 	DelayMax: 1000,
		// 	Prompt:   "{{ input }}\n**Answer is not correct, please try again.**\nError:\n{{ error }} \nAssistant's last answer:\n{{ output }}",
		// },
		Silent:  true,
		Options: map[string]interface{}{}, // LLM API options
	}

	// Get the options
	if this.Has("options") {
		jsOptions, err := this.Get("options")
		if err != nil {
			return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the options: %s", err.Error()))
		}

		// Check if the options is undefined
		if !jsOptions.IsUndefined() {
			err = bridge.Unmarshal(jsOptions, &options)
			if err != nil {
				return bridge.JsException(info.Context(), fmt.Sprintf("Failed to unmarshal the options: %s", err.Error()))
			}
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
	chatCtx.ChatID = fmt.Sprintf("call_%s", uuid.New().String()) // New chat id
	chatCtx.Silent = options.Silent
	chatCtx.Referer = chatctx.RefererScript // Set the referer to hookscript
	chatCtx.Args = goArgs                   // Arguments for call

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
	result, err := newAst.Execute(global.GinContext, chatCtx, input, options.Options, cb) // Execute the assistant
	if err != nil {
		// result, err = obj.retry(jsArgs, err, input, output, info, options)
		// if err != nil {
		// 	return bridge.JsException(info.Context(), err.Error())
		// }
		return bridge.JsException(info.Context(), err.Error())
	}

	// Copy props
	for name, value := range goCallProps {
		info.Context().Global().Set(name, value)
	}

	// Trigger the done event
	doneResult, err := obj.trigger(info, "done", jsArgs...)
	if err != nil {
		// result, err = obj.retry(jsArgs, err, input, output, info, options)
		// if err != nil {
		// 	return bridge.JsException(info.Context(), err.Error())
		// }
		return bridge.JsException(info.Context(), err.Error())
	}

	// Return the done result
	if doneResult != nil && !doneResult.IsUndefined() {
		return doneResult
	}

	// Return Value
	switch v := result.(type) {
	case *v8go.Value:
		return v
	case error:
		return bridge.JsException(info.Context(), v.Error())
	}

	// Return Value
	jsResult, err := bridge.JsValue(info.Context(), result)
	if err != nil {
		return bridge.JsException(info.Context(), fmt.Sprintf("Failed to get the result: %s", err.Error()))
	}
	return jsResult
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
func (obj *objectCall) trigger(info *v8go.FunctionCallbackInfo, name string, fnArgs ...v8go.Valuer) (*v8go.Value, error) {
	// Try to get the callback
	this := info.This()
	if this.Has(fmt.Sprintf("on_%s", name)) {
		event, err := this.Get(fmt.Sprintf("on_%s", name))
		if err != nil {
			return nil, err
		}

		if event.IsFunction() {

			cb, err := event.AsFunction()
			if err != nil {
				return nil, err
			}

			result, err := cb.Call(this, fnArgs...)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	return nil, nil
}
