package message

import "github.com/yaoapp/yao/attachment"

// Message the message
type Message struct {
	ID              string                  `json:"id,omitempty"`               // id for the message
	ToolID          string                  `json:"tool_id,omitempty"`          // tool_id for the message
	Text            string                  `json:"text,omitempty"`             // text content
	Type            string                  `json:"type,omitempty"`             // error, text, plan, table, form, page, file, video, audio, image, markdown, json ...
	Props           map[string]interface{}  `json:"props,omitempty"`            // props for the types
	IsDone          bool                    `json:"done,omitempty"`             // Mark as a done message from agent
	IsNew           bool                    `json:"new,omitempty"`              // Mark as a new message from agent
	IsDelta         bool                    `json:"delta,omitempty"`            // Mark as a delta message from agent
	Actions         []Action                `json:"actions,omitempty"`          // Conversation Actions for frontend
	Attachments     []attachment.Attachment `json:"attachments,omitempty"`      // File attachments
	Role            string                  `json:"role,omitempty"`             // user, assistant, system ...
	Name            string                  `json:"name,omitempty"`             // name for the message
	AssistantID     string                  `json:"assistant_id,omitempty"`     // assistant_id (for assistant role = assistant )
	AssistantName   string                  `json:"assistant_name,omitempty"`   // assistant_name (for assistant role = assistant )
	AssistantAvatar string                  `json:"assistant_avatar,omitempty"` // assistant_avatar (for assistant role = assistant )
	Mentions        []Mention               `json:"menions,omitempty"`          // Mentions for the message ( for user  role = user )
	Data            map[string]interface{}  `json:"-"`                          // data for the message
	Pending         bool                    `json:"-"`                          // pending for the message
	Hidden          bool                    `json:"hidden,omitempty"`           // hidden for the message (not show in the UI and history)
	Retry           bool                    `json:"retry,omitempty"`            // retry for the message
	Silent          bool                    `json:"silent,omitempty"`           // silent for the message (not show in the UI and history)
	IsTool          bool                    `json:"-"`                          // is tool for the message for native tool_calls
	IsBeginTool     bool                    `json:"-"`                          // is new tool for the message for native tool_calls
	IsEndTool       bool                    `json:"-"`                          // is end tool for the message for native tool_calls
	Result          any                     `json:"result,omitempty"`           // result for the message
	Begin           int64                   `json:"begin,omitempty"`            // begin at for the message // timestamp
	End             int64                   `json:"end,omitempty"`              // end at for the message // timestamp
}

// Mention represents a mention
type Mention struct {
	ID     string `json:"assistant_id"`     // assistant_id
	Name   string `json:"name"`             // name
	Avatar string `json:"avatar,omitempty"` // avatar
}

// Action the action
type Action struct {
	Name    string      `json:"name,omitempty"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}
