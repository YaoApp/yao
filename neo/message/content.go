package message

import "fmt"

const (
	// ContentStatusPending the content status pending
	ContentStatusPending = iota
	// ContentStatusDone the content status done
	ContentStatusDone
	// ContentStatusError the content status error
	ContentStatusError
)

// Content the content
type Content struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Bytes  []byte `json:"bytes"`
	Type   string `json:"type"`   // text, function, error
	Status uint8  `json:"status"` // 0: pending, 1: done
}

// NewContent create a new content
func NewContent(typ string) *Content {
	if typ == "" {
		typ = "text"
	}

	return &Content{
		Bytes:  []byte{},
		Type:   typ,
		Status: ContentStatusPending,
	}
}

// String the content string
func (c *Content) String() string {
	if c.Type == "function" {
		return fmt.Sprintf(`{"id":"%s","type": "function", "function": {"name": "%s", "arguments": "%s"}}`, c.ID, c.Name, c.Bytes)
	}
	return string(c.Bytes)
}

// SetID set the content id
func (c *Content) SetID(id string) {
	c.ID = id
}

// SetName set the content name
func (c *Content) SetName(name string) {
	c.Name = name
}

// SetType set the content type
func (c *Content) SetType(typ string) {
	c.Type = typ
}

// Append append the content
func (c *Content) Append(data string) {
	c.Bytes = append(c.Bytes, []byte(data)...)
}

// SetStatus set the content status
func (c *Content) SetStatus(status uint8) {
	c.Status = status
}
