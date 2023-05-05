package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/neo/message"
	"rogchap.com/v8go"
)

// Run the command
func (req *Request) Run(messages []map[string]interface{}, cb func(msg *message.JSON) int) (interface{}, error) {

	prompts, err := req.prepare(messages, cb)
	if err != nil {
		cb(req.msg().Text(fmt.Sprintf("Prepare Before Error: %s\n", err.Error())))
		cb(message.New().Done())
		req.Done()
		return nil, err
	}

	utils.Dump(prompts)

	cb(req.msg().Text(fmt.Sprintf("- Command: %s\n", req.Command.Name)))
	time.Sleep(200 * time.Millisecond)

	cb(req.msg().Text(fmt.Sprintf("- Session: %s\n", req.sid)))
	time.Sleep(200 * time.Millisecond)

	cb(req.msg().Text(fmt.Sprintf("- Request: %s\n", req.sid)))
	time.Sleep(200 * time.Millisecond)

	cb(req.msg().Done())
	return nil, nil
}

// RunPrepare the command
func (req *Request) prepare(messages []map[string]interface{}, cb func(msg *message.JSON) int) ([]Prompt, error) {

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

	return prompts, nil
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

func (req *Request) runScript(id string, args []interface{}, cb func(msg *message.JSON) int) (map[string]interface{}, error) {

	namer := strings.Split(id, ".")
	method := namer[len(namer)-1]
	scriptID := strings.Join(namer[1:len(namer)-1], ".")
	script, err := v8.Select(scriptID)
	if err != nil {
		return nil, err
	}

	// make a new script context
	v8ctx, err := script.NewContext(req.sid, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer v8ctx.Close()

	// make a new bridge function ssWrite
	ssWriteT := v8go.NewFunctionTemplate(v8ctx.Isolate(), func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) != 1 {
			return v8go.Null(v8ctx.Isolate())
		}

		text := args[0].String()
		cb(req.msg().Text(text))
		return v8go.Null(v8ctx.Isolate())
	})

	// make a new bridge function done
	doneT := v8go.NewFunctionTemplate(v8ctx.Isolate(), func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) == 1 {
			text := args[0].String()
			cb(req.msg().Text(text))
		}

		cb(message.New().Done())
		req.Done()
		return v8go.Null(v8ctx.Isolate())
	})

	v8ctx.Global().Set("ssWrite", ssWriteT.GetFunction(v8ctx.Context))
	v8ctx.Global().Set("done", doneT.GetFunction(v8ctx.Context))

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
func (cmd *Command) NewRequest(ctx Context) (*Request, error) {

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
			Command: cmd,
			sid:     ctx.Sid,
			id:      id,
			ctx:     ctx,
		}, nil
	}

	// create a new request
	id = uuid.New().String()
	err := DefaultStore.SetRequest(ctx.Sid, id, cmd.ID)
	if err != nil {
		return nil, err
	}

	return &Request{
		Command: cmd,
		sid:     ctx.Sid,
		id:      id,
		ctx:     ctx,
	}, nil
}

// Done the request done
func (req *Request) Done() {
	if DefaultStore != nil {
		DefaultStore.DelRequest(req.sid)
	}
}
