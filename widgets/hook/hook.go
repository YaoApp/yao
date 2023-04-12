package hook

import (
	"fmt"

	"github.com/yaoapp/gou/process"
)

// CopyBefore copy a before hook
func CopyBefore(hook *Before, new *Before) {
	if hook != nil && new != nil {
		*hook = *new
	}
}

// CopyAfter copy a after hook
func CopyAfter(hook *After, new *After) {
	if hook != nil && new != nil {
		*hook = *new
	}
}

// Exec execute the hook
func (hook *Before) Exec(args []interface{}, sid string, global map[string]interface{}) ([]interface{}, error) {

	p, err := process.Of(hook.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("%s %s", hook.String(), err.Error())
	}

	res, err := p.WithGlobal(global).WithSID(sid).Exec()
	if err != nil {
		return nil, fmt.Errorf("[%s] %s", hook.String(), err.Error())
	}

	newArgs, ok := res.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s return value is not an array", hook.String())
	}

	if len(newArgs) != len(args) {
		return nil, fmt.Errorf("%s return value is not correct. should: array[%d], got: array[%d]", hook.String(), len(args), len(newArgs))
	}

	return newArgs, nil
}

// Exec execute the hook
func (hook *After) Exec(value interface{}, sid string, global map[string]interface{}) (interface{}, error) {

	args := []interface{}{}
	switch value.(type) {
	case []interface{}:
		args = value.([]interface{})
	default:
		args = append(args, value)
	}

	p, err := process.Of(hook.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("[%s] %s", hook.String(), err.Error())
	}

	res, err := p.WithGlobal(global).WithSID(sid).Exec()
	if err != nil {
		return nil, fmt.Errorf("[%s] %s", hook.String(), err.Error())
	}

	return res, nil
}

// String cast to string
func (hook *Before) String() string {
	return string(*hook)
}

// String cast to string
func (hook *After) String() string {
	return string(*hook)
}
