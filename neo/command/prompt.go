package command

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/maps"
)

// Replace the prompt with the context
func (prompt Prompt) Replace(data maps.MapStrAny) Prompt {

	v := map[string]interface{}{"role": prompt.Role, "content": prompt.Content}
	if prompt.Name != "" {
		v["name"] = prompt.Name
	}

	replaced := helper.Bind(v, data)
	res, ok := replaced.(map[string]interface{})
	if !ok {
		return prompt
	}

	if res["role"] == nil {
		prompt.Role = ""
	} else if role, ok := res["role"].(string); ok {
		prompt.Role = role
	}

	if res["name"] == nil {
		prompt.Name = ""
	} else if name, ok := res["name"].(string); ok {
		prompt.Name = name
	}

	switch content := res["content"].(type) {
	case string:
		prompt.Content = content

	default:
		if content == nil {
			prompt.Content = ""

		} else if bytes, err := jsoniter.Marshal(content); err == nil {
			prompt.Content = string(bytes)
		}
	}

	return prompt
}
