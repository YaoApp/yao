package component

import (
	"fmt"
	"reflect"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

// "$C(value)", "$C(props)", "$C(type)"}
var defaults = []CArg{
	{IsExp: true, key: "value", value: nil},
	{IsExp: true, key: "props", value: nil},
	{IsExp: true, key: "type", value: nil},
	{IsExp: true, key: "id", value: nil},
	{IsExp: true, key: "path", value: nil},
}

// NewExp create a new exp CArg
func NewExp(key string) CArg {
	return CArg{IsExp: true, key: key, value: nil}
}

// Value compute value
func (compute *Compute) Value(data maps.MapStr, sid string, global map[string]interface{}) (interface{}, error) {

	if compute.Process == "" {
		return nil, fmt.Errorf("compute process is required")
	}

	// Build-In handlers
	args := compute.GetArgs(data)
	if handler, has := hanlders[compute.Process]; has {
		return handler(args...)
	}

	if !strings.Contains(compute.Process, ".") {
		return nil, fmt.Errorf("compute %s does not found", compute.Process)
	}

	// Run process
	process, err := process.Of(compute.Process, args...)
	if err != nil {
		return nil, err
	}

	res, err := process.WithSID(sid).WithGlobal(global).Exec()
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetArgs return args
func (compute *Compute) GetArgs(data maps.MapStr) []interface{} {
	args := []interface{}{}
	for _, arg := range compute.Args {
		args = append(args, arg.Value(data))
	}
	return args
}

// Value compute arg value
func (arg CArg) Value(data maps.MapStr) interface{} {
	if !arg.IsExp {
		return arg.value
	}
	return data.Get(arg.key)
}

// MarshalJSON  Custom JSON parse
func (compute Compute) MarshalJSON() ([]byte, error) {

	if compute.Args == nil || len(compute.Args) == 0 || reflect.DeepEqual(compute.Args, defaults) {
		return jsoniter.Marshal(compute.Process)
	}

	return jsoniter.Marshal(computeAlias(compute))
}

// UnmarshalJSON  Custom JSON parse
func (compute *Compute) UnmarshalJSON(data []byte) error {

	// allow null
	if data == nil || len(data) < 1 || (len(data) == 2 && data[0] == '"' && data[1] == '"') {
		*compute = Compute{Args: []CArg{}}
		return fmt.Errorf("Compute should be {} or string")
	}

	switch data[0] {

	case '[':
		*compute = Compute{Args: []CArg{}}
		return fmt.Errorf("Compute should be {} or string")

	case '{': // json
		var new computeAlias
		err := jsoniter.Unmarshal(data, &new)
		if err != nil {
			return err
		}
		new.Process = strings.TrimSpace(new.Process)
		*compute = Compute(new)
		return nil

	default:
		compute.Process = strings.TrimSpace((strings.Trim(string(data), `"`)))
		compute.Args = defaults
		return nil
	}
}

// MarshalJSON for JSON parse
func (arg CArg) MarshalJSON() ([]byte, error) {
	if arg.IsExp {
		return []byte(fmt.Sprintf(`"$C(%s)"`, arg.key)), nil
	}

	if v, ok := arg.value.(string); ok && strings.HasPrefix(v, "::") {
		return jsoniter.Marshal(fmt.Sprintf("\\%s", v))
	}

	return jsoniter.Marshal(arg.value)
}

// UnmarshalJSON for JSON parse
func (arg *CArg) UnmarshalJSON(data []byte) error {

	if data == nil || len(data) < 1 {
		*arg = CArg{value: nil, IsExp: false}
		return nil
	}

	// "$C(value)", "$C(props)", "$C(type)"}
	if len(data) > 3 && data[0] == '"' && data[1] == '$' && data[2] == 'C' && data[3] == '(' {
		key := strings.TrimSpace(strings.TrimRight(strings.TrimLeft(string(data), `"$C(`), `)"`))
		*arg = CArg{key: key, IsExp: true}
		return nil

	} else if len(data) > 4 && data[0] == '"' && data[1] == '\\' && data[2] == '\\' && data[3] == ':' && data[4] == ':' {

		//  ["$C(row.type)", "\\::", "$C(value)", "-", "$C(row.status)"]
		value := string(data[3 : len(data)-1])
		*arg = CArg{value: value, IsExp: false}
		return nil
	}

	var v interface{}
	err := jsoniter.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	*arg = CArg{value: v, IsExp: false}
	return nil
}
