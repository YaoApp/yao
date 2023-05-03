package command

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func output(format string, args ...interface{}) []byte {
	content := fmt.Sprintf(format, args...)
	return []byte(fmt.Sprintf(`{"id":"chatcmpl-7Atx502nGBuYcvoZfIaWU4FREI1mT","object":"chat.completion.chunk","created":1682832715,"model":"gpt-3.5-turbo-0301","choices":[{"delta":{"content":"%s"},"index":0,"finish_reason":null}]}`, content))
}

// Run the command
func (req *Request) Run(messages []map[string]interface{}, cb func(data []byte) int) (interface{}, error) {

	cb(output("- Command: %s\\n", req.Command.ID))
	time.Sleep(200 * time.Millisecond)

	cb(output("- Session: %s\\n", req.sid))
	time.Sleep(200 * time.Millisecond)

	cb(output("- Request: %s\\n", req.id))
	time.Sleep(200 * time.Millisecond)

	cb([]byte(`[DONE]`))
	return nil, nil
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
