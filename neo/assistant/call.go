package assistant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	chatctx "github.com/yaoapp/yao/neo/context"
	chatMessage "github.com/yaoapp/yao/neo/message"
	sui "github.com/yaoapp/yao/sui/core"
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
		Retry: OptionsCallRetry{
			Times:    3,
			Delay:    200,
			DelayMax: 1000,
			Prompt:   "{{ input }}\n**Answer is not correct, please try again.**\nError:\n{{ error }} \nAssistant's last answer:\n{{ output }}",
		},
		Silent:  true,
		Options: map[string]interface{}{}, // LLM API options
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
	chatCtx.ChatID = fmt.Sprintf("call_%s", uuid.New().String()) // New chat id
	chatCtx.Silent = options.Silent                              // Check the silent mode

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
		result, err = obj.retry(jsArgs, err, input, output, info, options)
		if err != nil {
			return bridge.JsException(info.Context(), err.Error())
		}
	}

	// Copy props
	for name, value := range goCallProps {
		info.Context().Global().Set(name, value)
	}

	// Trigger the done event
	doneResult, err := obj.trigger(info, "done", jsArgs...)
	if err != nil {
		result, err = obj.retry(jsArgs, err, input, output, info, options)
		if err != nil {
			return bridge.JsException(info.Context(), err.Error())
		}
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

func (obj *objectCall) retry(jsArgs []v8go.Valuer, err error, input interface{}, output []chatMessage.Message, info *v8go.FunctionCallbackInfo, options OptionsCall) (*v8go.Value, error) {

	// Retry times, if not set, return the error
	if options.Retry.Times <= 0 {
		return nil, err
	}

	this := info.This()
	errmsg := exception.Trim(err)

	// Get current retry times
	jsTimes, retryErr := this.Get("retry_times")
	if retryErr != nil {
		return nil, fmt.Errorf("%s occurred but failed to get the retry times: %s", errmsg, retryErr.Error())
	}

	times := int(jsTimes.Int32())
	if times > options.Retry.Times {
		return nil, fmt.Errorf("%s occurred, max retry times reached", errmsg)
	}

	// Update the retry times
	times = times + 1
	this.Set("retry_times", int32(times))

	// Message content
	content := ""
	for _, msg := range output {
		if msg.Type == "text" && msg.IsDelta {
			content += msg.Text
		}
	}

	// Delay
	delay := options.Retry.Delay * int(times)
	if delay > options.Retry.DelayMax {
		delay = options.Retry.DelayMax
	}

	// Retry delay (millisecond)
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	// Format the input
	var lastUserMessage *chatMessage.Message = nil
	var inputMessages []*chatMessage.Message = nil
	var lastUserMessageIndex int = 0
	switch v := input.(type) {
	case string:
		lastUserMessage = &chatMessage.Message{Type: "text", Text: v, Role: "user"}
		inputMessages = []*chatMessage.Message{lastUserMessage}

	case []interface{}:
		// Get the last user message
		raw, parseErr := jsoniter.Marshal(v)
		if parseErr != nil {
			return nil, fmt.Errorf("%s occurred but failed to marshal the input: %s", errmsg, parseErr.Error())
		}

		parseErr = jsoniter.Unmarshal(raw, &inputMessages)
		if parseErr != nil {
			return nil, fmt.Errorf("%s occurred but failed to unmarshal the input: %s", errmsg, parseErr.Error())
		}

		// Get the last user message
		for i := len(inputMessages) - 1; i >= 0; i-- {
			if inputMessages[i].Type == "text" && inputMessages[i].Role == "user" {
				lastUserMessage = inputMessages[i]
				lastUserMessageIndex = i
				break
			}
		}

	case *chatMessage.Message:
		lastUserMessage = v
		inputMessages = []*chatMessage.Message{lastUserMessage}

	case map[string]interface{}:
		text, ok := v["text"].(string)
		if !ok {
			return nil, fmt.Errorf("%s occurred but failed to get the text", errmsg)
		}

		if v["role"] != "user" {
			return nil, fmt.Errorf("%s occurred but the role is not user", errmsg)
		}

		lastUserMessage = &chatMessage.Message{Type: "text", Text: text, Role: "user"}
		inputMessages = []*chatMessage.Message{lastUserMessage}
	}

	// Get the prompt from the options
	promptTmpl := options.Retry.Prompt
	data := sui.Data{
		"error":  errmsg,
		"output": strings.TrimSpace(content),
		"input":  lastUserMessage.Text,
	}
	prompt, _ := data.Replace(promptTmpl)

	// Custom retry prompt by hooking the retry event
	if this.Has("on_retry") {
		info.Context().Global().Set("error", errmsg) // Set error
		jsDelay, _ := bridge.JsValue(info.Context(), delay)
		jsPrompt, _ := bridge.JsValue(info.Context(), prompt)
		newPrompt, retryErr := obj.trigger(info, "retry", jsTimes, jsDelay, jsPrompt)
		if retryErr != nil {
			return nil, fmt.Errorf("%s occurred but failed to trigger the retry event: %s", errmsg, retryErr.Error())
		}
		// Update the prompt
		if newPrompt.IsString() {
			prompt = newPrompt.String()
		}
	}

	// Generate the new input with the prompt
	// Update the input
	inputMessages[lastUserMessageIndex].Text = prompt
	jsInput, inputErr := bridge.JsValue(info.Context(), inputMessages)
	if inputErr != nil {
		return nil, fmt.Errorf("%s occurred but failed to update the input: %s", errmsg, inputErr.Error())
	}
	// Update the input
	this.Set("retry_input", jsInput)

	// Call the run function
	run, funcErr := this.Get("Run")
	if funcErr != nil {
		return nil, fmt.Errorf("%s occurred but failed to get the run function: %s", errmsg, funcErr.Error())
	}

	if !run.IsFunction() {
		return nil, fmt.Errorf("%s occurred but the run function is not a function", errmsg)
	}

	fn, fnErr := run.AsFunction()
	if fnErr != nil {
		return nil, fmt.Errorf("%s occurred but failed to get the run function: %s", errmsg, fnErr.Error())
	}

	// Call the run function
	result, resErr := fn.Call(this, jsArgs...)
	if resErr != nil {
		return nil, fmt.Errorf("%s (%d)", exception.Trim(resErr), times-1)
	}

	return result, nil
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
