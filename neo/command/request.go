package command

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/conversation"
	"github.com/yaoapp/yao/neo/message"
	"rogchap.com/v8go"
)

// Run the command
func (req *Request) Run(messages []map[string]interface{}, cb func(msg *message.JSON) int) error {

	// Enter the command mode
	if input, ok := messages[len(messages)-1]["content"].(string); ok {
		match := reCmdOnly.FindSubmatch([]byte(strings.TrimSpace(input)))
		if match != nil {
			fmt.Printf("Match Command: %s  | %s\n", match[1], input)
			cb(req.msg().Text("Enter the command Mode"))
			cb(req.msg().Done())
			return nil
		}
	}

	if config.Conf.Mode == "development" {
		utils.Dump("----Request Run ----")
		fmt.Printf("Command Request: %s %s\n", req.Command.ID, req.sid)
		fmt.Printf("Command Process: %s\n", req.Command.Process)
		fmt.Printf("Command Prepare Before: %s\n", req.Command.Prepare.Before)
		fmt.Printf("Command Prepare Before: %s\n", req.Command.Prepare.After)
	}

	input, err := req.prepare(messages, cb)
	if err != nil {
		req.error(err, cb)
		return err
	}

	if config.Conf.Mode == "development" {
		utils.Dump("----Input After Prepare ----", input)
	}

	args, err := req.parseArgs(input, cb)
	if err != nil {
		cb(req.msg().Text("\n\n" + err.Error()))
		cb(req.msg().Done())
		return nil
	}

	if config.Conf.Mode == "development" {
		utils.Dump("---- Command Args ----", args)
	}

	// Send the command to the service
	if req.Command.Optional.Confirm {
		req.confirm(args, cb)
		return nil
	}

	// Execute the command by script
	if strings.HasPrefix(req.Command.Process, "scripts.") || strings.HasPrefix(req.Command.Process, "studio.") {

		res, err := req.runScript(req.Command.Process, args, cb)
		if err != nil {
			cb(req.msg().Text("\n\n" + err.Error()))
			return err
		}

		msg := req.msg().Bind(res)
		if req.Actions != nil && len(req.Actions) > 0 {
			for _, action := range req.Actions {
				msg.Action(action.Name, action.Type, action.Payload, action.Next)
			}
		}

		cb(msg.Done())
		return nil
	}

	// Other process
	p, err := process.Of(req.Command.Process, args...)
	if err != nil {
		return err
	}

	res, err := p.Exec()
	if err != nil {
		cb(req.msg().Text("\n\n" + err.Error()))
		return err
	}

	msg := req.msg()
	if data, ok := res.(map[string]interface{}); ok {
		msg = msg.Bind(data)
	}

	if req.Actions != nil && len(req.Actions) > 0 {
		for _, action := range req.Actions {
			msg.Action(action.Name, action.Type, action.Payload, action.Next)
		}
	}

	// DONE
	cb(msg.Done())
	return nil
}

// confirm the command
func (req *Request) confirm(args []interface{}, cb func(msg *message.JSON) int) {

	payload := map[string]interface{}{
		"method": "ExecCommand",
		"args": []interface{}{
			req.id,
			req.Command.Process,
			args,
			map[string]interface{}{"stack": req.ctx.Stack, "path": req.ctx.Path},
		},
	}

	msg := req.msg().
		Action("ExecCommand", "Service.__neo", payload, "").
		Confirm().
		Done()

	if req.Actions != nil && len(req.Actions) > 0 {
		for _, action := range req.Actions {
			msg.Action(action.Name, action.Type, action.Payload, action.Next)
		}
	}

	cb(msg)
}

// validate the command
func (req *Request) parseArgs(input interface{}, cb func(msg *message.JSON) int) ([]interface{}, error) {
	args := []interface{}{}
	data := map[string]interface{}{}

	switch v := input.(type) {
	case string:
		err := jsoniter.Unmarshal([]byte(v), &data)
		if err != nil {
			return nil, err
		}
		break

	case []byte:
		err := jsoniter.Unmarshal(v, &data)
		if err != nil {
			return nil, err
		}
		break

	case map[string]interface{}:
		data = v
		break

	default:
		err := fmt.Errorf("\nInvalid input type: %T", v)
		req.error(err, cb)
		return nil, err
	}

	// validate the args
	if req.Command.Args != nil && len(req.Command.Args) > 0 {
		for _, arg := range req.Command.Args {
			v, ok := data[arg.Name]
			if arg.Required && !ok {
				err := fmt.Errorf("\nMissing required argument: %s", arg.Name)
				return nil, err
			}

			// @todo: validate the type
			args = append(args, v)
		}
	}

	return args, nil
}

// RunPrepare the command
func (req *Request) prepare(messages []map[string]interface{}, cb func(msg *message.JSON) int) (interface{}, error) {

	if config.Conf.Mode == "development" {
		utils.Dump("----Messages Before Prepare ----", messages)
	}

	// Before hook
	data, err := req.prepareBefore(messages, cb)
	if err != nil {
		return nil, err
	}

	// replace the pro
	prompts := []Prompt{}
	if req.Command.Prepare.Prompts != nil && len(req.Command.Prepare.Prompts) > 0 {
		prompts = append(prompts, req.Command.Prepare.Prompts...)
	}

	if data != nil {
		data = maps.Of(data).Dot()
		for i, prompt := range prompts {
			prompts[i] = prompt.Replace(data)
		}
	}

	question, err := req.question(messages)
	if err != nil {
		req.error(err, cb)
		return nil, err
	}

	chatMessages, err := req.messages(prompts, question)
	if err != nil {
		req.error(err, cb)
		return nil, err
	}

	if config.Conf.Mode == "development" {
		utils.Dump("----Command Prompts ----", chatMessages)
	}

	// chat with AI
	content := []byte{}
	_, ex := req.AI.ChatCompletionsWith(req.ctx, chatMessages, req.Prepare.Option, func(data []byte) int {
		msg := message.NewOpenAI(data)
		if msg != nil {
			if msg.IsDone() {
				return 0
			}
			content = msg.Append(content)
			cb(req.msg().Text(msg.String()))
		}
		return 1
	})

	if ex != nil {
		req.error(fmt.Errorf(ex.Message), cb)
		return nil, fmt.Errorf("Chat error: %s", ex.Message)
	}
	defer req.saveHistory(content, chatMessages)

	// After hook
	args, err := req.prepareAfter(string(content), cb)
	if err != nil {
		log.Error("Prepare after error: %s", err.Error())
		fmt.Println(err)
		return content, nil
	}

	return args, nil
}

// prepareBefore hook
func (req *Request) prepareBefore(messages []map[string]interface{}, cb func(msg *message.JSON) int) (map[string]interface{}, error) {

	if req.Prepare.Before == "" {
		return nil, nil
	}

	// prepare the args
	args := []interface{}{
		map[string]interface{}{"stack": req.ctx.Stack, "path": req.ctx.Path},
		messages,
	}

	return req.runScript(req.Prepare.Before, args, cb)
}

// prepareAfter hook
func (req *Request) prepareAfter(content string, cb func(msg *message.JSON) int) (interface{}, error) {

	if req.Prepare.After == "" {
		return content, nil
	}

	// prepare the args
	args := []interface{}{
		content,
		map[string]interface{}{"stack": req.ctx.Stack, "path": req.ctx.Path}, // context
	}

	return req.runScript(req.Prepare.After, args, cb)
}

// saveHistory save the history
func (req *Request) saveHistory(content []byte, messages []map[string]interface{}) {

	if len(content) > 0 && req.sid != "" && len(messages) > 0 {
		err := req.conversation.SaveRequest(
			req.sid,
			req.id,
			req.Command.ID,
			[]map[string]interface{}{
				{"role": "user", "content": messages[len(messages)-1]["content"], "name": req.sid},
				{"role": "assistant", "content": string(content), "name": req.sid},
			},
		)

		if err != nil {
			log.Error("Save request error: %s", err.Error())
		}
	}
}

func (req *Request) error(err error, cb func(msg *message.JSON) int) {
	cb(req.msg().Text(err.Error()))
	cb(req.msg().Done())
	// req.Done()
}

func (req *Request) question(messages []map[string]interface{}) (string, error) {
	if len(messages) < 1 {
		return "", fmt.Errorf("No messages")
	}

	question, ok := messages[len(messages)-1]["content"].(string)
	if !ok {
		return "", fmt.Errorf("messages content is not string")
	}

	return question, nil
}

func (req *Request) messages(prompts []Prompt, question string) ([]map[string]interface{}, error) {
	messages := []map[string]interface{}{}
	for _, prompt := range prompts {
		message := map[string]interface{}{"role": prompt.Role, "content": prompt.Content}
		if prompt.Name != "" {
			message["name"] = prompt.Name
		}
		messages = append(messages, message)
	}

	history, err := req.conversation.GetRequest(req.sid, req.id)
	if err != nil {
		return nil, err
	}
	messages = append(messages, history...)
	messages = append(messages, map[string]interface{}{"role": "user", "content": question, "name": req.sid})
	return messages, nil
}

func (req *Request) runScript(id string, args []interface{}, cb func(msg *message.JSON) int) (map[string]interface{}, error) {

	namer := strings.Split(id, ".")
	method := namer[len(namer)-1]
	scriptID := strings.Join(namer[1:len(namer)-1], ".")

	var err error
	var script *v8.Script
	if namer[0] == "scripts" {
		script, err = v8.Select(scriptID)
	} else if namer[0] == "studio" {
		script, err = v8.SelectRoot(scriptID)
	}

	if err != nil {
		return nil, err
	}

	// make a new script context
	v8ctx, err := script.NewContext(req.sid, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer v8ctx.Close()

	v8ctx.WithFunction("ssWrite", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) != 1 {
			return v8go.Null(info.Context().Isolate())
		}

		text := args[0].String()
		cb(req.msg().Text(text))
		return v8go.Null(info.Context().Isolate())
	})

	v8ctx.WithFunction("done", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) == 1 {
			text := args[0].String()
			cb(req.msg().Text(text))
		}

		cb(req.msg().Done())
		// req.Done()
		return v8go.Null(v8ctx.Context.Isolate())
	})

	res, err := v8ctx.CallWith(req.ctx, method, args...)
	if err != nil {
		return nil, err
	}

	// return data
	switch v := res.(type) {

	case bool:
		return nil, nil

	case map[string]interface{}:
		return v, nil

	default:
		return nil, fmt.Errorf("script return type is not supported")
	}

}

func (req *Request) msg() *message.JSON {
	return message.New().Command(req.Command.Name, req.Command.ID, req.id)
}

// NewRequest create a new request
func (cmd *Command) NewRequest(ctx Context, conversation conversation.Conversation) (*Request, error) {

	if DefaultStore == nil {
		return nil, fmt.Errorf("command store is not set")
	}

	if ctx.Sid == "" {
		return nil, fmt.Errorf("context sid is request")
	}

	// continue the request
	id, cid, has := DefaultStore.GetRequest(ctx.Sid)
	if has {
		if cid != cmd.ID {
			return nil, fmt.Errorf("request id is not match")
		}
		return &Request{
			Command:      cmd,
			sid:          ctx.Sid,
			id:           id,
			ctx:          ctx,
			conversation: conversation,
		}, nil
	}

	// create a new request
	id = uuid.New().String()
	err := DefaultStore.SetRequest(ctx.Sid, id, cmd.ID)
	if err != nil {
		return nil, err
	}

	return &Request{
		Command:      cmd,
		sid:          ctx.Sid,
		id:           id,
		ctx:          ctx,
		conversation: conversation,
	}, nil
}

// Done the request done
func (req *Request) Done() {
	if DefaultStore != nil {
		DefaultStore.DelRequest(req.sid)
	}
}
