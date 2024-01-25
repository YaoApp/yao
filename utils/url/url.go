package url

import (
	"fmt"
	"net/url"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/exception"
)

// ProcessParseQuery  utils.url.ParseQuery
func ProcessParseQuery(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	queryString := process.ArgsString(0)
	params, err := url.ParseQuery(queryString)
	if err != nil {
		exception.New("make audio captcha error: %s", 500, err).Throw()
	}
	return params
}

// ProcessParseURL utils.url.ParseURL
func ProcessParseURL(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	rawurl := process.ArgsString(0)
	u, err := url.Parse(rawurl)
	if err != nil {
		exception.New("parse url error: %s", 500, err).Throw()
	}
	return map[string]interface{}{
		"scheme": u.Scheme,
		"host":   u.Host,
		"domain": u.Hostname(),
		"path":   u.Path,
		"port":   u.Port(),
		"query":  u.Query(),
		"url":    u.String(),
	}
}

// ProcessQueryParam handle the get Template request
func ProcessQueryParam(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	switch v := process.Args[0].(type) {

	case url.Values:
		return types.URLToQueryParam(v)

	case map[string][]string:
		return types.URLToQueryParam(v)

	case map[string]interface{}:
		values := url.Values{}
		for key, value := range v {
			switch val := value.(type) {
			case []string:
				for _, v := range val {
					values.Add(key, v)
				}

			case []interface{}:
				for _, v := range val {
					values.Add(key, fmt.Sprintf("%v", v))
				}

			default:
				values.Set(key, fmt.Sprintf("%v", value))
			}
		}
		return types.URLToQueryParam(values)
	}

	v, _ := types.AnyToQueryParam(process.Args[0])
	return v
}
