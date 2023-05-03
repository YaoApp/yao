package message

import (
	"io"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openai"
)

// JSON the JSON message
type JSON struct{ *Message }

// New create a new JSON message
func New() *JSON {
	return &JSON{makeMessage()}
}

// NewOpenAI create a new JSON message
func NewOpenAI(data []byte) *JSON {

	if data == nil || len(data) == 0 {
		return nil
	}

	msg := makeMessage()
	data = []byte(strings.TrimPrefix(string(data), "data: "))
	switch {

	case strings.Contains(string(data), `"delta":{"content"`):
		var message openai.Message
		err := jsoniter.Unmarshal(data, &message)
		if err != nil {
			msg.Text = err.Error()
			return &JSON{msg}
		}

		if len(message.Choices) > 0 {
			msg.Text = message.Choices[0].Delta.Content
		}
		break

	case strings.Contains(string(data), `[DONE]`):
		msg.Done = true
		break

	default:
		return nil
	}

	return &JSON{msg}
}

// Text set the text
func (json *JSON) Text(text string) *JSON {
	json.Message.Text = text
	return json
}

// Done set the done
func (json *JSON) Done() *JSON {
	json.Message.Done = true
	return json
}

// Confirm set the confirm
func (json *JSON) Confirm() *JSON {
	json.Message.Confirm = true
	return json
}

// Command set the command
func (json *JSON) Command(name, id, request string) *JSON {
	json.Message.Command = &Command{
		ID:      id,
		Name:    name,
		Reqeust: request,
	}
	return json
}

// Action set the action
func (json *JSON) Action(name string, t string, payload interface{}, next string) *JSON {
	json.Message.Actions = append(json.Message.Actions, Action{
		Name:    name,
		Type:    t,
		Payload: payload,
		Next:    next,
	})
	return json
}

// IsDone check if the message is done
func (json *JSON) IsDone() bool {
	return json.Message.Done
}

// Write the message
func (json *JSON) Write(w io.Writer) bool {

	data, err := jsoniter.Marshal(json.Message)
	if err != nil {
		log.Error("%s", err.Error())
		return false
	}

	data = append([]byte("data: "), data...)
	data = append(data, []byte("\n\n")...)

	_, err = w.Write(data)
	if err != nil {
		log.Error("%s", err.Error())
		return false
	}

	return true
}

// Append the message
func (json *JSON) Append(content []byte) []byte {
	return append(content, []byte(json.Message.Text)...)
}
