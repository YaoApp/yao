package pipe

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
)

// UnmarshalJSON Custom JSON unmarshal function
func (whitelist *Whitelist) UnmarshalJSON(data []byte) error {

	var list any
	err := jsoniter.Unmarshal(data, &list)
	if err != nil {
		return err
	}

	switch v := list.(type) {
	case []string:
		list := map[string]bool{}
		for _, name := range v {
			list[name] = true
		}
		*whitelist = list

	case []interface{}:
		list := map[string]bool{}
		for _, name := range v {
			list[fmt.Sprint(name)] = true
		}
		*whitelist = list

	case map[string]interface{}:
		list := map[string]bool{}
		for name := range v {
			list[name] = true
		}
		*whitelist = list

	default:
		return fmt.Errorf("whitelist type error: %#v", v)
	}

	return nil
}

// UnmarshalJSON Custom JSON unmarshal function
func (input Input) UnmarshalJSON(data []byte) error {

	var res any
	err := jsoniter.Unmarshal(data, &res)
	if err != nil {
		return err
	}

	switch v := res.(type) {
	case []string:
		input = []any{}
		for _, name := range v {
			input = append(input, name)
		}

	case []interface{}:
		input = v

	case string:
		input = []any{v}

	default:
		return fmt.Errorf("input type error: %#v", v)
	}

	return nil

}

// UnmarshalJSON Custom JSON unmarshal function
func (args Args) UnmarshalJSON(data []byte) error {

	var res any
	err := jsoniter.Unmarshal(data, &res)
	if err != nil {
		return err
	}

	switch v := res.(type) {
	case []string:
		args = []any{}
		for _, name := range v {
			args = append(args, name)
		}

	case []interface{}:
		args = v

	case string:
		args = []any{v}

	default:
		return fmt.Errorf("input type error: %#v", v)
	}

	return nil
}

// UnmarshalJSON Custom JSON unmarshal function
func (autoFill *AutoFill) UnmarshalJSON(data []byte) error {

	var res any
	err := jsoniter.Unmarshal(data, &res)
	if err != nil {
		return err
	}

	switch v := res.(type) {

	case map[string]interface{}:
		if value, has := v["value"]; has {
			autoFill.Value = fmt.Sprint(value)
		}
		if action, has := v["action"]; has {
			autoFill.Action = fmt.Sprint(action)
		}

	default:
		autoFill.Value = v
	}

	return nil

}
