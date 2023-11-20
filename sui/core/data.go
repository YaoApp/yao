package core

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
)

// Data get the data
func (page *Page) Data(request *Request) (map[string]interface{}, map[string]interface{}, error) {

	setting := map[string]interface{}{
		"title": strings.ToUpper(page.Name),
	}

	if page.Codes.DATA.Code != "" {
		err := jsoniter.UnmarshalFromString(page.Codes.DATA.Code, &setting)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, setting, nil
}

// Exec get the data
func (page *Page) Exec(request *Request) (map[string]interface{}, error) {

	if page.Codes.DATA.Code == "" {
		return map[string]interface{}{}, nil
	}

	data := map[string]interface{}{}
	err := jsoniter.UnmarshalFromString(page.Codes.DATA.Code, &data)
	if err != nil {
		return nil, err
	}

	err = page.execMap(request, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (page *Page) execMap(r *Request, m map[string]interface{}) error {

	for key, value := range m {

		if strings.HasPrefix(key, "$") {
			res, err := page.call(r, value)
			if err != nil {
				return err
			}
			newKey := key[1:]
			m[newKey] = res
			delete(m, key)
			continue
		}

		res, err := page.execValue(r, value)
		if err != nil {
			return err
		}
		m[key] = res

	}

	return nil
}

func (page *Page) execValue(r *Request, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "$") {
			return page.call(r, strings.TrimLeft(v, "$"))
		}
		return v, nil

	case []interface{}:
		for i, item := range v {
			res, err := page.execValue(r, item)
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
		return page.execValue(r, interfaceSlice)

	case map[string]interface{}:

		if _, ok := v["process"].(string); ok {
			if call, _ := v["__exec"].(bool); call {
				res, err := page.call(r, v)
				if err != nil {
					return nil, err
				}
				return res, nil
			}
		}

		err := page.execMap(r, v)
		if err != nil {
			return nil, err
		}
		return v, nil

	default:
		return v, nil
	}
}

func (page *Page) call(r *Request, p interface{}) (interface{}, error) {

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
			args, err := page.parseArgs(r, args)
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

func (page *Page) parseArgs(r *Request, args []interface{}) ([]interface{}, error) {

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
			res, err := page.parseArgs(r, v)
			if err != nil {
				return nil, err
			}
			args[i] = res

		case map[string]interface{}:
			res, err := page.parseArgs(r, []interface{}{v})
			if err != nil {
				return nil, err
			}
			args[i] = res[0]
		}
	}

	return args, nil
}
