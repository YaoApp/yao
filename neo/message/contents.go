package message

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
)

const (
	// ContentStatusPending the content status pending
	ContentStatusPending = iota
	// ContentStatusDone the content status done
	ContentStatusDone
	// ContentStatusError the content status error
	ContentStatusError
)

var tokens = map[string][2]string{
	"think": {"<think>", "</think>"},
	"tool":  {"<tool>", "</tool>"},
}

// Contents the contents
type Contents struct {
	Current int               `json:"current"` // the current content index
	Data    []Data            `json:"data"`    // the data
	token   string            // the current token
	id      string            // the id of the contents
	stack   [][]string        // the token stack
	mapping map[string]string // the mapping of the token stack
}

// Data the data of the content
type Data struct {
	Type  string                 `json:"type"`            // text, function, error, think, tool
	ID    string                 `json:"id"`              // the id of the content
	Bytes []byte                 `json:"bytes"`           // the content bytes
	Props map[string]interface{} `json:"props"`           // the props
	Begin int64                  `json:"begin,omitempty"` // the begin time
	End   int64                  `json:"end,omitempty"`   // the end time
}

// Extra the extra of the content
type Extra struct {
	ID    string `json:"id,omitempty"`    // the id of the content
	Begin int64  `json:"begin,omitempty"` // the begin time
	End   int64  `json:"end,omitempty"`   // the end time
}

// ScanCallbackParams the params of the scan callback
type ScanCallbackParams struct {
	Token     string
	MessageID string
	TokenID   string
	BeganAt   int64
	EndAt     int64
	Begin     bool
	End       bool
	Text      string
	Tails     string
}

// NewContents create a new contents
func NewContents() *Contents {
	return &Contents{
		Current: -1,
		Data:    []Data{},
	}
}

// ScanTokens scan the tokens
func (c *Contents) ScanTokens(messageID string, tokenID string, beganAt int64, cb func(params ScanCallbackParams)) {

	text := strings.TrimSpace(c.Text())

	// check the end of the token
	if c.token != "" {

		token := c.GetToken(c.token)
		tokenType := c.GetTokenType(c.token)
		// Check the end of the token
		if index := strings.Index(text, token[1]); index >= 0 {
			tails := ""
			if index > 0 {
				tails = text[index+len(token[1]):]
			}

			extra := Extra{
				ID:  c.id,
				End: time.Now().UnixNano(),
			}

			c.UpdateType(tokenType, map[string]interface{}{"text": text}, extra)
			c.NewText([]byte(tails), extra) // Create new text with the tails
			cb(ScanCallbackParams{Token: tokenType, MessageID: c.id, TokenID: tokenID, BeganAt: beganAt, Begin: false, End: true, Text: text, Tails: tails, EndAt: extra.End})
			c.ClearToken(c.token) // clear the token
			return
		}

		// call the callback for the scanning of the token
		cb(ScanCallbackParams{Token: tokenType, MessageID: c.id, TokenID: tokenID, BeganAt: beganAt, Begin: false, End: false, Text: text, Tails: "", EndAt: 0})
		return
	}

	// scan the begin of the token
	begin := false
	for name, token := range tokens {
		if index := strings.Index(text, token[0]); index >= 0 {

			c.id = messageID
			if c.id == "" {
				c.id = GenerateNumericID("M")
			}

			tokenType := name
			if tokenID != "" {
				tokenType = c.GetTokenType(tokenID)
			}

			// First time scanning the token, generate the token ID and begin time
			if tokenID == "" || tokenType != name {
				tokenID = GenerateNumericID("T")
				beganAt = time.Now().UnixNano()
				begin = true
				c.token = tokenID
				c.AppendToken(tokenID, name)
				c.UpdateType(name, map[string]interface{}{"text": text, "id": tokenID}, Extra{ID: c.id, Begin: beganAt, End: beganAt})
			}

			cb(ScanCallbackParams{Token: name, MessageID: c.id, TokenID: tokenID, BeganAt: beganAt, Begin: begin, End: false, Text: text, Tails: "", EndAt: 0}) // call the callback
		}
	}
}

// ClearToken clear the token
func (c *Contents) ClearToken(id string) {
	c.token = ""
	next := 0

	if c.stack == nil {
		c.stack = [][]string{}
	}

	if c.mapping == nil {
		c.mapping = map[string]string{}
	}

	for i, node := range c.stack {
		if node[0] == id {
			next = i + 1
			delete(c.mapping, id)
			break
		}
	}

	// Remove the token from the stack, and set the next token
	if next > 0 && next < len(c.stack) {
		c.stack = c.stack[next:]
		c.token = c.stack[len(c.stack)-1][0]
	}
}

// AppendToken append the token to the stack
func (c *Contents) AppendToken(id string, name string) {
	if c.stack == nil {
		c.stack = [][]string{}
	}
	if c.mapping == nil {
		c.mapping = map[string]string{}
	}
	c.stack = append(c.stack, []string{id, name})
	c.mapping[id] = name
	c.token = id
}

// GetTokenType get the token type from the stack
func (c *Contents) GetTokenType(id string) string {
	return c.mapping[id]
}

// GetToken get the token from the stack
func (c *Contents) GetToken(name string) [2]string {
	typ, ok := c.mapping[name]
	if !ok {
		return [2]string{}
	}
	return tokens[typ]
}

// RemoveLastEmpty remove the last empty data
func (c *Contents) RemoveLastEmpty() {
	if c.Current == -1 {
		return
	}

	// Remove the last empty data
	if len(c.Data[c.Current].Bytes) == 0 && c.Data[c.Current].Type == "text" {
		c.Data = c.Data[:c.Current]
		c.Current--
	}
}

// NewText create a new text data and append to the contents
func (c *Contents) NewText(bytes []byte, extra ...Extra) *Contents {
	data := Data{Type: "text", Bytes: bytes}

	if len(extra) > 0 {
		if extra[0].Begin != 0 {
			data.Begin = extra[0].Begin
		}
		if extra[0].End != 0 {
			data.End = extra[0].End
		}
		if extra[0].ID != "" {
			data.ID = extra[0].ID
		}
	}

	c.Data = append(c.Data, data)
	c.Current++
	return c
}

// NewType create a new type data and append to the contents
func (c *Contents) NewType(typ string, props map[string]interface{}, extra ...Extra) *Contents {

	data := Data{
		Type:  typ,
		Props: props,
	}

	if len(extra) > 0 {
		if extra[0].Begin != 0 {
			data.Begin = extra[0].Begin
		}
		if extra[0].End != 0 {
			data.End = extra[0].End
		}
		if extra[0].ID != "" {
			data.ID = extra[0].ID
		}
	}

	c.Data = append(c.Data, data)
	c.Current++
	return c
}

// UpdateType update the type of the current content
func (c *Contents) UpdateType(typ string, props map[string]interface{}, extra ...Extra) *Contents {
	if c.Current == -1 {
		c.NewType(typ, props, extra...)
		return c
	}

	if len(extra) > 0 {
		if extra[0].Begin != 0 {
			c.Data[c.Current].Begin = extra[0].Begin
		}
		if extra[0].End != 0 {
			c.Data[c.Current].End = extra[0].End
		}
		if extra[0].ID != "" {
			c.Data[c.Current].ID = extra[0].ID
		}
	}
	c.Data[c.Current].Type = typ
	if props != nil {
		if c.Data[c.Current].Props == nil {
			c.Data[c.Current].Props = map[string]interface{}{}
		}

		for k, v := range props {
			c.Data[c.Current].Props[k] = v
		}
	}
	return c
}

// NewError create a new error data and append to the contents
func (c *Contents) NewError(err []byte) *Contents {
	c.Data = append(c.Data, Data{
		Type:  "error",
		Bytes: err,
	})
	c.Current++
	return c
}

// AppendText append the text to the current content
func (c *Contents) AppendText(bytes []byte, extra ...Extra) *Contents {
	if c.Current == -1 {
		c.NewText(bytes, extra...)
		return c
	}

	if len(extra) > 0 {
		if extra[0].ID != "" {
			c.Data[c.Current].ID = extra[0].ID
		}
		if extra[0].Begin != 0 {
			c.Data[c.Current].Begin = extra[0].Begin
		}
		if extra[0].End != 0 {
			c.Data[c.Current].End = extra[0].End
		}
	}
	c.Data[c.Current].Bytes = append(c.Data[c.Current].Bytes, bytes...)
	return c
}

// AppendError append the error to the current content
func (c *Contents) AppendError(err []byte) *Contents {
	if c.Current == -1 {
		c.NewError(err)
		return c
	}
	c.Data[c.Current].Bytes = append(c.Data[c.Current].Bytes, err...)
	return c
}

// JSON returns the json representation
func (c *Contents) JSON() string {
	raw, _ := jsoniter.MarshalToString(c.Data)
	return raw
}

// Text returns the text of the current content
func (c *Contents) Text() string {
	if c.Current == -1 {
		return ""
	}
	return string(c.Data[c.Current].Bytes)
}

// CurrentType returns the type of the current content
func (c *Contents) CurrentType() string {
	if c.Current == -1 {
		return ""
	}
	return c.Data[c.Current].Type
}

// Map returns the map representation
func (data *Data) Map() (map[string]interface{}, error) {
	v := map[string]interface{}{"type": data.Type}

	if data.ID != "" {
		v["id"] = data.ID
	}

	if data.Bytes != nil && data.Type == "text" {
		v["text"] = string(data.Bytes)
	}

	if data.Props != nil && data.Type != "text" {
		v["props"] = data.Props
	}

	return v, nil
}

// MarshalJSON returns the json representation
func (data *Data) MarshalJSON() ([]byte, error) {

	v := map[string]interface{}{"type": data.Type}

	if data.ID != "" {
		v["id"] = data.ID
	}

	if data.Bytes != nil && data.Type == "text" {
		v["text"] = string(data.Bytes)
	}

	if data.Props != nil && data.Type != "text" {
		v["props"] = data.Props
	}

	// Add the begin and end time
	if data.Begin != 0 {
		v["begin"] = data.Begin
	}

	if data.End != 0 {
		v["end"] = data.End
	}

	return jsoniter.Marshal(v)
}

// GenerateNumericID generates a 10-digit number using UUID as seed
func GenerateNumericID(prefix string) string {
	// Generate UUID and use it as seed
	id := uuid.New()
	seed := int64(id[0])<<56 | int64(id[1])<<48 | int64(id[2])<<40 | int64(id[3])<<32 |
		int64(id[4])<<24 | int64(id[5])<<16 | int64(id[6])<<8 | int64(id[7])

	// Create a new random source using the seed
	source := rand.NewSource(seed)
	r := rand.New(source)

	// Generate a number between 1000000000 and 9999999999 (10 digits)
	num := r.Int63n(9000000000) + 1000000000

	return fmt.Sprintf("%s%d", prefix, num)
}
