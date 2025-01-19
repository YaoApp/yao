package message

import (
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

// Contents the contents
type Contents struct {
	Current int    `json:"current"` // the current content index
	Data    []Data `json:"data"`    // the data
}

// Data the data of the content
type Data struct {
	Type      string `json:"type"`      // text, function, error, ...
	ID        string `json:"id"`        // the id of the content
	Function  string `json:"function"`  // the function name
	Bytes     []byte `json:"bytes"`     // the content bytes
	Arguments []byte `json:"arguments"` // the function arguments
}

// NewContents create a new contents
func NewContents() *Contents {
	return &Contents{
		Current: -1,
		Data:    []Data{},
	}
}

// NewText create a new text data and append to the contents
func (c *Contents) NewText(bytes []byte) *Contents {
	c.Data = append(c.Data, Data{
		Type:  "text",
		Bytes: bytes,
	})
	c.Current++
	return c
}

// NewFunction create a new function data and append to the contents
func (c *Contents) NewFunction(function string, arguments []byte) *Contents {
	c.Data = append(c.Data, Data{
		Type:      "function",
		Function:  function,
		Arguments: arguments,
	})
	c.Current++
	return c
}

// SetFunctionID set the id of the current function content
func (c *Contents) SetFunctionID(id string) *Contents {
	if c.Current == -1 {
		c.NewFunction("", []byte{})
	}
	c.Data[c.Current].ID = id
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
func (c *Contents) AppendText(bytes []byte) *Contents {
	if c.Current == -1 {
		c.NewText(bytes)
		return c
	}
	c.Data[c.Current].Bytes = append(c.Data[c.Current].Bytes, bytes...)
	return c
}

// AppendFunction append the function to the current content
func (c *Contents) AppendFunction(arguments []byte) *Contents {
	if c.Current == -1 {
		c.NewFunction("", arguments)
		return c
	}
	c.Data[c.Current].Arguments = append(c.Data[c.Current].Arguments, arguments...)
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

// Map returns the map representation
func (data *Data) Map() (map[string]interface{}, error) {
	v := map[string]interface{}{"type": data.Type}

	if data.ID != "" {
		v["id"] = data.ID
	}

	if data.Bytes != nil {
		v["text"] = string(data.Bytes)
	}

	if data.Arguments != nil {
		var vv interface{} = nil
		err := jsoniter.Unmarshal(data.Arguments, &vv)
		if err != nil {
			return nil, err
		}
		v["arguments"] = vv
	}

	if data.Function != "" {
		v["function"] = data.Function
	}

	return v, nil
}

// MarshalJSON returns the json representation
func (data *Data) MarshalJSON() ([]byte, error) {

	v := map[string]interface{}{"type": data.Type}

	if data.ID != "" {
		v["id"] = data.ID
	}

	if data.Bytes != nil {
		v["text"] = string(data.Bytes)
	}

	if data.Arguments != nil {
		var vv interface{} = nil
		err := jsoniter.Unmarshal(data.Arguments, &vv)
		if err != nil {
			return nil, err
		}
		v["arguments"] = vv
	}

	if data.Function != "" {
		v["function"] = data.Function
	}

	return jsoniter.Marshal(v)
}
