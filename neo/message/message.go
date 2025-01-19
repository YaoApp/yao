package message

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openai"
)

// Message the message
type Message struct {
	Text            string                 `json:"text,omitempty"`             // text content
	Type            string                 `json:"type,omitempty"`             // error, text, plan, table, form, page, file, video, audio, image, markdown, json ...
	Props           map[string]interface{} `json:"props,omitempty"`            // props for the types
	IsDone          bool                   `json:"done,omitempty"`             // Mark as a done message from neo
	IsNew           bool                   `json:"is_new,omitempty"`           // Mark as a new message from neo
	Actions         []Action               `json:"actions,omitempty"`          // Conversation Actions for frontend
	Attachments     []Attachment           `json:"attachments,omitempty"`      // File attachments
	Role            string                 `json:"role,omitempty"`             // user, assistant, system ...
	Name            string                 `json:"name,omitempty"`             // name for the message
	AssistantID     string                 `json:"assistant_id,omitempty"`     // assistant_id (for assistant role = assistant )
	AssistantName   string                 `json:"assistant_name,omitempty"`   // assistant_name (for assistant role = assistant )
	AssistantAvatar string                 `json:"assistant_avatar,omitempty"` // assistant_avatar (for assistant role = assistant )
	Mentions        []Mention              `json:"menions,omitempty"`          // Mentions for the message ( for user  role = user )
	Data            map[string]interface{} `json:"-"`
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

// NewString create a new message from string
func NewString(content string) (*Message, error) {
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		var msg Message
		if err := jsoniter.UnmarshalFromString(content, &msg); err != nil {
			return nil, err
		}
		return &msg, nil
	}
	return &Message{Text: content}, nil
}

// NewOpenAI create a new message from OpenAI response
func NewOpenAI(data []byte) *Message {
	if data == nil || len(data) == 0 {
		return nil
	}

	msg := New()
	text := string(data)
	data = []byte(strings.TrimPrefix(text, "data: "))

	switch {

	case strings.Contains(text, `"delta":{`) && strings.Contains(text, `"tool_calls"`):
		var toolCalls openai.ToolCalls
		if err := jsoniter.Unmarshal(data, &toolCalls); err != nil {
			msg.Text = err.Error() + "\n" + string(data)
			return msg
		}

		msg.Type = "tool_calls"
		if len(toolCalls.Choices) > 0 && len(toolCalls.Choices[0].Delta.ToolCalls) > 0 {
			msg.Props["id"] = toolCalls.Choices[0].Delta.ToolCalls[0].ID
			msg.Props["function"] = toolCalls.Choices[0].Delta.ToolCalls[0].Function.Name
			msg.Text = toolCalls.Choices[0].Delta.ToolCalls[0].Function.Arguments
		}

	case strings.Contains(text, `"delta":{`) && strings.Contains(text, `"content":`):
		var message openai.Message
		if err := jsoniter.Unmarshal(data, &message); err != nil {
			msg.Text = err.Error() + "\n" + string(data)
			return msg
		}

		msg.Type = "text"
		if len(message.Choices) > 0 {
			msg.Text = message.Choices[0].Delta.Content
		}

	case strings.Contains(text, `[DONE]`):
		msg.IsDone = true

	case strings.Contains(text, `"finish_reason":"stop"`):
		msg.IsDone = true

	case strings.Contains(text, `"finish_reason":"tool_calls"`):
		msg.IsDone = true

	default:
		str := strings.TrimPrefix(strings.Trim(string(data), "\""), "data: ")
		msg.Type = "error"
		msg.Text = str
	}

	return msg
}

// String returns the string representation
func (m *Message) String() string {
	if m.Text != "" {
		return m.Text
	}
	return ""
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

	switch m.Type {
	case "text":
		if m.Text != "" {
			contents.AppendText([]byte(m.Text))
		}

	case "tool_calls":

		// Set function name
		if name, ok := m.Props["function"].(string); ok && name != "" {
			contents.NewFunction(name, []byte(m.Text))
		}

		// Set id
		if id, ok := m.Props["id"].(string); ok && id != "" {
			contents.SetFunctionID(id)
		}

		contents.AppendFunction([]byte(m.Text))
	}
	return m
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

	if isNew, ok := msg["is_new"].(bool); ok {
		m.IsNew = isNew
	}

	if assistantID, ok := msg["assistant_id"].(string); ok {
		m.AssistantID = assistantID
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

// Write writes the message to response writer
func (m *Message) Write(w gin.ResponseWriter) bool {
	defer func() {
		if r := recover(); r != nil {
			message := "Write Response Exception: (if client close the connection, it's normal) \n  %s\n\n"
			color.Red(message, r)
		}
	}()

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
