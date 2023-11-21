package core

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
)

// ExecString get the data
func (r *Request) ExecString(data string) (Data, error) {
	var res Data
	err := jsoniter.UnmarshalFromString(data, &res)
	if err != nil {
		return nil, err
	}
	r.Exec(res)
	return res, nil
}

// Exec get the data
func (r *Request) Exec(m Data) error {

	for key, value := range m {

		if strings.HasPrefix(key, "$") {
			res, err := r.call(value)
			if err != nil {
				return err
			}
			newKey := key[1:]
			m[newKey] = res
			delete(m, key)
			continue
		}

		res, err := r.execValue(value)
		if err != nil {
			return err
		}
		m[key] = res

	}

	return nil
}

func (r *Request) execValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "$") {
			return r.call(strings.TrimLeft(v, "$"))
		}
		return v, nil

	case []interface{}:
		for i, item := range v {
			res, err := r.execValue(item)
			if err != nil {
				return nil, err
			}
			v[i] = res
		}
		return v, nil

	case []string:
		interfaceSlice := make([]interface{}, len(v))
		for i, item := range v {
			interfaceSlice[i] = item
		}
		return r.execValue(interfaceSlice)

	case map[string]interface{}:

		if _, ok := v["process"].(string); ok {
			if call, _ := v["__exec"].(bool); call {
				res, err := r.call(v)
				if err != nil {
					return nil, err
				}
				return res, nil
			}
		}

		err := r.Exec(v)
		if err != nil {
			return nil, err
		}
		return v, nil

	default:
		return v, nil
	}
}

func (r *Request) call(p interface{}) (interface{}, error) {

	processName := ""
	processArgs := []interface{}{r}
	switch v := p.(type) {
	case string:
		processName = v
		break

	case map[string]interface{}:
		if name, ok := v["process"].(string); ok {
			processName = name
		}

		if args, ok := v["args"].([]interface{}); ok {
			args, err := r.parseArgs(args)
			if err != nil {
				return nil, err
			}
			processArgs = append(args, processArgs...)
		}
	}

	if processName == "" {
		return nil, fmt.Errorf("process name is empty")
	}

	process, err := process.Of(processName, processArgs...)
	if err != nil {
		return nil, err
	}

	return process.Exec()
}

func (r *Request) parseArgs(args []interface{}) ([]interface{}, error) {

	data := any.MapOf(map[string]interface{}{
		"param":   r.Params,
		"query":   r.Query,
		"payload": map[string]interface{}{},
		"header":  r.Headers,
		"theme":   r.Theme,
		"locale":  r.Locale,
	}).Dot()

	for i, arg := range args {
		switch v := arg.(type) {

		case string:
			if strings.HasPrefix(v, "$") {
				key := strings.TrimLeft(v, "$")
				args[i] = key
				if data.Has(key) {
					v := data.Get(key)
					if strings.HasPrefix(key, "query.") || strings.HasPrefix(key, "header.") {
						switch arg := v.(type) {
						case []interface{}:
							if len(arg) == 1 {
								args[i] = arg[0]
							}
						case []string:
							if len(arg) == 1 {
								args[i] = arg[0]
							}
						}
					}
				}
			}

		case []interface{}:
			res, err := r.parseArgs(v)
			if err != nil {
				return nil, err
			}
			args[i] = res

		case map[string]interface{}:
			res, err := r.parseArgs([]interface{}{v})
			if err != nil {
				return nil, err
			}
			args[i] = res[0]
		}
	}

	return args, nil
}
