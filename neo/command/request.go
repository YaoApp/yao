package command

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/kun/exception"
)

var requests = sync.Map{}

// Run the command
func (req *Request) Run(cb func(data []byte) int) (interface{}, error) {
	return nil, nil
}

// NewRequest create a new request
func (cmd *Command) NewRequest(ctx Context, messages []map[string]interface{}) (*Request, error) {

	v, ok := requests.Load(ctx.Sid)
	if !ok {
		v = map[string]string{
			"id":  uuid.New().String(),
			"cmd": cmd.ID,
		}
	}

	req, ok := v.(map[string]string)
	if !ok {
		return nil, fmt.Errorf("request id is not string")
	}

	if req["id"] == "" {
		return nil, fmt.Errorf("request id is request")
	}

	if req["cmd"] != cmd.ID {
		defer requests.Delete(ctx.Sid)
		return nil, fmt.Errorf("request id is not match")
	}

	return &Request{
		Command:  cmd,
		messages: messages,
		sid:      ctx.Sid,
		id:       req["id"],
		ctx:      ctx,
	}, nil
}

// Done the request done
func (req *Request) Done() {
	requests.Delete(req.sid)
}

// prepare the command
func (req *Request) prepare(ctx context.Context, data []map[string]interface{}, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (int, *exception.Exception) {
	return 1, nil
}

// before the process
func (req *Request) before(ctx context.Context, data []map[string]interface{}, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	return nil, nil
}

// after the process
func (req *Request) after(ctx context.Context, data []map[string]interface{}, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	return nil, nil
}

// run the process
func (req *Request) process(ctx context.Context, data []map[string]interface{}, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	return nil, nil
}

func (req *Request) saveConversation(ctx context.Context, data []map[string]interface{}, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	return nil, nil
}

func (req *Request) saveData(ctx context.Context, data []map[string]interface{}, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception) {
	return nil, nil
}
