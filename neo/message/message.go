package message

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openai"
)

var locker = sync.Mutex{}

// Message the message
type Message struct {
	ID              string                 `json:"id,omitempty"`               // id for the message
	ToolID          string                 `json:"tool_id,omitempty"`          // tool_id for the message
	Text            string                 `json:"text,omitempty"`             // text content
	Type            string                 `json:"type,omitempty"`             // error, text, plan, table, form, page, file, video, audio, image, markdown, json ...
	Props           map[string]interface{} `json:"props,omitempty"`            // props for the types
	IsDone          bool                   `json:"done,omitempty"`             // Mark as a done message from neo
	IsNew           bool                   `json:"new,omitempty"`              // Mark as a new message from neo
	IsDelta         bool                   `json:"delta,omitempty"`            // Mark as a delta message from neo
	Actions         []Action               `json:"actions,omitempty"`          // Conversation Actions for frontend
	Attachments     []Attachment           `json:"attachments,omitempty"`      // File attachments
	Role            string                 `json:"role,omitempty"`             // user, assistant, system ...
	Name            string                 `json:"name,omitempty"`             // name for the message
	AssistantID     string                 `json:"assistant_id,omitempty"`     // assistant_id (for assistant role = assistant )
	AssistantName   string                 `json:"assistant_name,omitempty"`   // assistant_name (for assistant role = assistant )
	AssistantAvatar string                 `json:"assistant_avatar,omitempty"` // assistant_avatar (for assistant role = assistant )
	Mentions        []Mention              `json:"menions,omitempty"`          // Mentions for the message ( for user  role = user )
	Data            map[string]interface{} `json:"-"`                          // data for the message
	Pending         bool                   `json:"-"`                          // pending for the message
	Hidden          bool                   `json:"hidden,omitempty"`           // hidden for the message (not show in the UI and history)
	Retry           bool                   `json:"retry,omitempty"`            // retry for the message
	Silent          bool                   `json:"silent,omitempty"`           // silent for the message (not show in the UI and history)
	IsTool          bool                   `json:"-"`                          // is tool for the message for native tool_calls
	IsBeginTool     bool                   `json:"-"`                          // is new tool for the message for native tool_calls
	IsEndTool       bool                   `json:"-"`                          // is end tool for the message for native tool_calls
	Result          any                    `json:"result,omitempty"`           // result for the message
	Begin           int64                  `json:"begin,omitempty"`            // begin at for the message // timestamp
	End             int64                  `json:"end,omitempty"`              // end at for the message // timestamp
}

// Mention represents a mention
type Mention struct {
	ID     string `json:"assistant_id"`     // assistant_id
	Name   string `json:"name"`             // name
	Avatar string `json:"avatar,omitempty"` // avatar
}

// Attachment represents a file attachment
type Attachment struct {
	Name        string `json:"name,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Bytes       int64  `json:"bytes,omitempty"`
	CreatedAt   int64  `json:"created_at,omitempty"`
	FileID      string `json:"file_id,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
	AssistantID string `json:"assistant_id,omitempty"`
}

// Action the action
type Action struct {
	Name    string      `json:"name,omitempty"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// New create a new message
func New() *Message {
	return &Message{Actions: []Action{}, Props: map[string]interface{}{}}
}

// NewHistory create a new message from history
func NewHistory(history map[string]interface{}) ([]Message, error) {
	if history == nil {
		return []Message{}, nil
	}

	var copy map[string]interface{} = map[string]interface{}{}
	for key, value := range history {
		if key != "content" {
			copy[key] = value
		}
	}

	globalMessage := New().Map(copy)
	messages := []Message{}
	if content, ok := history["content"].(string); ok {
		if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
			var msg Message = *globalMessage
			if err := jsoniter.UnmarshalFromString(content, &msg); err != nil {
				return nil, err
			}
			messages = append(messages, msg)
		} else if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
			var msgs []Message
			if err := jsoniter.UnmarshalFromString(content, &msgs); err != nil {
				return nil, err
			}
			for _, msg := range msgs {
				msg.AssistantID = globalMessage.AssistantID
				msg.AssistantName = globalMessage.AssistantName
				msg.AssistantAvatar = globalMessage.AssistantAvatar
				msg.Role = globalMessage.Role
				msg.Name = globalMessage.Name
				msg.Mentions = globalMessage.Mentions
				messages = append(messages, msg)
			}
		} else {
			messages = append(messages, Message{Text: content})
		}
	}

	return messages, nil
}

// NewContent create a new message from content
func NewContent(content string) ([]Message, error) {
	messages := []Message{}
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		var msg Message
		if err := jsoniter.UnmarshalFromString(content, &msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	} else if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
		var msgs []Message
		if err := jsoniter.UnmarshalFromString(content, &msgs); err != nil {
			return nil, err
		}
		for _, msg := range msgs {
			messages = append(messages, msg)
		}
	} else {
		messages = append(messages, Message{Text: content})
	}
	return messages, nil
}

// NewString create a new message from string
func NewString(content string, id ...string) (*Message, error) {
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		var msg Message
		if err := jsoniter.UnmarshalFromString(content, &msg); err != nil {
			return nil, err
		}
		return &msg, nil
	}
	if len(id) > 0 {
		return &Message{ID: id[0], Text: content}, nil
	}
	return &Message{Text: content}, nil
}

// NewStringError create a new message from string error
func NewStringError(content string) (*Message, error) {
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		var msg = New()
		var errorMessage openai.ErrorMessage
		if err := jsoniter.UnmarshalFromString(content, &errorMessage); err != nil {
			msg.Text = err.Error() + "\n" + content
			return msg, nil
		}
		msg.Type = "error"
		msg.Text = errorMessage.Error.Message
		return msg, nil
	}
	return &Message{Text: content}, nil
}

// NewMap create a new message from map
func NewMap(content map[string]interface{}) (*Message, error) {
	return New().Map(content), nil
}

// NewAny create a new message from any content
func NewAny(content interface{}) (*Message, error) {
	switch v := content.(type) {
	case string:
		return NewString(v)
	case map[string]interface{}:
		return NewMap(v)
	}
	return nil, fmt.Errorf("unknown content type: %T", content)
}

// NewOpenAI create a new message from OpenAI response
func NewOpenAI(data []byte, isThinking bool) *Message {

	// For debug environment, print the response data
	if os.Getenv("YAO_AGENT_PRINT_RESPONSE_DATA") == "true" {
		log.Trace("[Response Data] %s", string(data))
	}

	if data == nil || len(data) == 0 {
		return nil
	}

	msg := New()
	text := string(data)
	data = []byte(strings.TrimPrefix(text, "data: "))

	switch {
	case strings.Contains(text, `"object":"chat.completion.chunk"`): // Delta content
		var chunk openai.ChatCompletionChunk
		err := jsoniter.Unmarshal(data, &chunk)
		if err != nil {
			color.Red("JSON parse error: %s", err.Error())
			color.White(string(data))
			msg.Text = "JSON parse error\n" + string(data)
			msg.Type = "error"
			msg.IsDone = true
		}

		// Empty content, then it is a pending message
		if len(chunk.Choices) == 0 {
			msg.Pending = true
			return msg
		}

		// Tool calls
		if len(chunk.Choices[0].Delta.ToolCalls) > 0 || chunk.Choices[0].FinishReason == "tool_calls" {
			msg.Type = "tool_calls_native"
			text := ""
			if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
				id := chunk.Choices[0].Delta.ToolCalls[0].ID
				function := chunk.Choices[0].Delta.ToolCalls[0].Function.Name
				arguments := chunk.Choices[0].Delta.ToolCalls[0].Function.Arguments
				text = arguments
				if id != "" {
					msg.IsBeginTool = true
					msg.IsNew = true // mark as a new message
					text = fmt.Sprintf(`{"id": "%s", "function": "%s", "arguments": %s`, id, function, arguments)
				}
			}

			if chunk.Choices[0].FinishReason == "tool_calls" {
				msg.IsEndTool = true
			}

			msg.Text = text
			return msg
		}

		// Text content
		if chunk.Choices[0].Delta.Content != "" {
			msg.Type = "text"
			msg.Text = chunk.Choices[0].Delta.Content
			msg.IsDone = chunk.Choices[0].FinishReason == "stop" // is done when the content is finished
			return msg
		}

		// Done messages
		if chunk.Choices[0].FinishReason == "stop" || chunk.Choices[0].FinishReason == "tool_calls" {
			msg.IsDone = true
			return msg
		}

		// Reasoning content
		if chunk.Choices[0].Delta.ReasoningContent != "" {
			msg.Type = "think"
			msg.Text = chunk.Choices[0].Delta.ReasoningContent
			return msg
		}
		// Content is empty and is thinking, then it is a thinking message pending
		if isThinking {
			msg.Type = "think"
			msg.Text = ""
			return msg
		}

		msg.Text = ""
		return msg

	case strings.Contains(text, `"usage":`): // usage content
		msg.IsDone = true
		break

	case strings.Contains(text, `[DONE]`):
		msg.IsDone = true
		return msg

	case len(data) > 2 && data[0] == '{' && data[len(data)-1] == '}': // JSON content (error)

		var error openai.Error
		var errorMessage openai.ErrorMessage
		if strings.Contains(string(data), `"error":`) {
			if err := jsoniter.Unmarshal(data, &errorMessage); err != nil {
				color.Red("JSON parse error: %s", err.Error())
				color.White(string(data))
				msg.Text = "JSON parse error\n" + string(data)
				msg.Type = "error"
				msg.IsDone = true
				return msg
			}
			error = errorMessage.Error
		} else {
			err := jsoniter.Unmarshal(data, &error)
			if err != nil {
				color.Red("JSON parse error: %s", err.Error())
				color.White(string(data))
				msg.Text = "JSON parse error\n" + string(data)
				msg.Type = "error"
				msg.IsDone = true
				return msg
			}
		}

		message := error.Message
		if message == "" {
			message = "Unknown error occurred\n" + string(data)
		}

		msg.Type = "error"
		msg.Text = message
		msg.IsDone = true
		return msg

	case !strings.Contains(text, `data: `): // unknown message or uncompleted message
		msg.Pending = true
		msg.Text = text
		return msg

	default: // unknown message
		str := strings.TrimPrefix(strings.Trim(string(data), "\""), "data: ")
		msg.Type = "error"
		msg.Text = str
		return msg
	}

	return msg
}

// String returns the string representation
func (m *Message) String() string {
	typ := m.Type
	if typ == "" {
		typ = "text"
	}

	switch typ {
	case "text", "think", "tool", "tool_calls_native":
		return m.Text

	case "error":
		return m.Text

	default:
		raw, _ := jsoniter.MarshalToString(map[string]interface{}{"type": m.Type, "props": m.Props})
		return raw
	}
}

// SetText set the text
func (m *Message) SetText(text string) *Message {
	m.Text = text
	if m.Data != nil {
		if replaced := helper.Bind(text, m.Data); replaced != nil {
			if replacedText, ok := replaced.(string); ok {
				m.Text = replacedText
			}
		}
	}
	return m
}

// SetProps set the props
func (m *Message) SetProps(props map[string]interface{}) *Message {
	m.Props = props
	return m
}

// Error set the error
func (m *Message) Error(message interface{}) *Message {
	m.Type = "error"
	switch v := message.(type) {
	case error:
		m.Text = v.Error()
	case string:
		m.Text = v
	default:
		m.Text = fmt.Sprintf("%v", message)
	}
	return m
}

// SetContent set the content
func (m *Message) SetContent(content string) *Message {
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		var msg Message
		if err := jsoniter.UnmarshalFromString(content, &msg); err != nil {
			m.Text = err.Error() + "\n" + content
			return m
		}
		*m = msg
	} else {
		m.Text = content
		m.Type = "text"
	}
	return m
}

// AppendTo append the contents
func (m *Message) AppendTo(contents *Contents) *Message {

	// Set type
	if m.Type == "" {
		m.Type = "text"
	}

	switch m.Type {
	case "text", "think", "tool", "tool_calls_native":
		if m.Text != "" {
			if m.IsNew {
				contents.NewText([]byte(m.Text), Extra{ID: m.ID, Begin: m.Begin, End: m.End})
				return m
			}
			contents.AppendText([]byte(m.Text), Extra{ID: m.ID, Begin: m.Begin, End: m.End})
			return m
		}
		return m

	case "loading", "error", "action", "progress": // Ignore progress, loading, action and error messages
		return m

	default:
		if m.IsNew {
			contents.NewType(m.Type, m.Props)
			return m
		}
		contents.UpdateType(m.Type, m.Props)
		return m
	}

}

// Content get the content
func (m *Message) Content() string {
	content := map[string]interface{}{"text": m.Text}
	if m.Attachments != nil {
		content["attachments"] = m.Attachments
	}

	if m.Type != "" {
		content["type"] = m.Type
	}
	contentRaw, _ := jsoniter.MarshalToString(content)
	return contentRaw
}

// ToMap convert to map
func (m *Message) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"content": m.Content(),
		"role":    m.Role,
		"name":    m.Name,
	}
}

// Map set from map
func (m *Message) Map(msg map[string]interface{}) *Message {
	if msg == nil {
		return m
	}

	// Content  {"text": "xxxx",  "attachments": ... }
	if content, ok := msg["content"].(string); ok {
		if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
			var msg Message
			if err := jsoniter.UnmarshalFromString(content, &msg); err != nil {
				m.Text = err.Error() + "\n" + content
				return m
			}
			*m = msg
		} else {
			m.Text = content
			m.Type = "text"
		}
	}

	if role, ok := msg["role"].(string); ok {
		m.Role = role
	}

	if name, ok := msg["name"].(string); ok {
		m.Name = name
	}

	if text, ok := msg["text"].(string); ok {
		m.Text = text
	}
	if typ, ok := msg["type"].(string); ok {
		m.Type = typ
	}
	if done, ok := msg["done"].(bool); ok {
		m.IsDone = done
	}
	if props, ok := msg["props"].(map[string]interface{}); ok {
		m.Props = props
	}

	if isNew, ok := msg["new"].(bool); ok {
		m.IsNew = isNew
	}

	if isDelta, ok := msg["delta"].(bool); ok {
		m.IsDelta = isDelta
	}

	if assistantID, ok := msg["assistant_id"].(string); ok {
		m.AssistantID = assistantID

		// Set name
		if m.Role == "assistant" {
			m.Name = m.AssistantID
		}
	}

	if assistantName, ok := msg["assistant_name"].(string); ok {
		m.AssistantName = assistantName
	}

	if assistantAvatar, ok := msg["assistant_avatar"].(string); ok {
		m.AssistantAvatar = assistantAvatar
	}

	if actions, ok := msg["actions"].([]interface{}); ok {
		for _, action := range actions {
			if v, ok := action.(map[string]interface{}); ok {
				action := Action{}
				if name, ok := v["name"].(string); ok {
					action.Name = name
				}
				if t, ok := v["type"].(string); ok {
					action.Type = t
				}
				if payload, ok := v["payload"].(map[string]interface{}); ok {
					action.Payload = payload
				}
				m.Actions = append(m.Actions, action)
			}
		}
	}
	if data, ok := msg["data"].(map[string]interface{}); ok {
		m.Data = data
	}
	return m
}

// Done set the done flag
func (m *Message) Done() *Message {
	m.IsDone = true
	return m
}

// Assistant set the assistant
func (m *Message) Assistant(id string, name string, avatar string) *Message {
	m.AssistantID = id
	m.AssistantName = name
	m.AssistantAvatar = avatar
	return m
}

// Action add an action
func (m *Message) Action(name string, t string, payload interface{}, next string) *Message {
	if m.Data != nil {
		payload = helper.Bind(payload, m.Data)
	}
	m.Actions = append(m.Actions, Action{
		Name:    name,
		Type:    t,
		Payload: payload,
	})
	return m
}

// Bind replace with data
func (m *Message) Bind(data map[string]interface{}) *Message {
	if data == nil {
		return m
	}
	m.Data = maps.Of(data).Dot()
	return m
}

// Callback callback the message
func (m *Message) Callback(fn interface{}) *Message {
	if fn != nil {
		switch v := fn.(type) {
		case func(msg *Message):
			if v == nil {
				break
			}
			v(m)
			break

		case func():
			if v == nil {
				break
			}
			v()
			break

		default:
			fmt.Println("no match callback")
			break
		}
	}
	return m
}

// Write writes the message to response writer
func (m *Message) Write(w gin.ResponseWriter) bool {

	// Sync write to response writer
	locker.Lock()
	defer locker.Unlock()

	defer func() {
		if r := recover(); r != nil {

			// Ignore if done is true
			if m.IsDone {
				return
			}

			message := "Write Response Exception: (if client close the connection, it's normal) \n  %s\n\n"
			color.Red(message, r)

			// Print the message
			raw, _ := jsoniter.MarshalToString(m)
			color.White("Message:\n %s", raw)
		}
	}()

	// Ignore silent messages
	if m.Silent {
		return true
	}

	data, err := jsoniter.Marshal(m)
	if err != nil {
		log.Error("%s", err.Error())
		return false
	}

	data = append([]byte("data: "), data...)
	data = append(data, []byte("\n\n")...)

	if _, err := w.Write(data); err != nil {
		color.Red("Write JSON Message Error: %s", err.Error())
		return false
	}
	w.Flush()
	return true
}

// WriteError writes an error message to response writer
func (m *Message) WriteError(w gin.ResponseWriter, message string) {
	errMsg := strings.Trim(exception.New(message, 500).Message, "\"")
	data := []byte(fmt.Sprintf(`{"text":"%s","type":"error"`, errMsg))
	if m.IsDone {
		data = []byte(fmt.Sprintf(`{"text":"%s","type":"error","done":true`, errMsg))
	}
	data = append([]byte("data: "), data...)
	data = append(data, []byte("}\n\n")...)

	if _, err := w.Write(data); err != nil {
		color.Red("Write JSON Message Error: %s", message)
	}
	w.Flush()
}

// MarshalJSON implements json.Marshaler interface
func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return jsoniter.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := jsoniter.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (a *Action) MarshalJSON() ([]byte, error) {
	type Alias Action
	return jsoniter.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface
func (a *Action) UnmarshalJSON(data []byte) error {
	type Alias Action
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := jsoniter.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}
