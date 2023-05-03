package command

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/yao/neo/message"
)

// Run the command
func (req *Request) Run(messages []map[string]interface{}, cb func(msg *message.JSON) int) (interface{}, error) {

	cb(req.msg().Text(fmt.Sprintf("- Command: %s\n", req.Command.Name)))
	time.Sleep(200 * time.Millisecond)

	cb(req.msg().Text(fmt.Sprintf("- Session: %s\n", req.sid)))
	time.Sleep(200 * time.Millisecond)

	cb(req.msg().Text(fmt.Sprintf("- Request: %s\n", req.sid)))
	time.Sleep(200 * time.Millisecond)

	cb(req.msg().Done())
	return nil, nil
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
	if DefaultStore == nil {
		DefaultStore.DelRequest(req.sid)
	}
}
