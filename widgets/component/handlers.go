package component

import (
	"fmt"
	"strings"
)

var hanlders = map[string]ComputeHanlder{
	"Get":           Get,
	"Trim":          Trim,
	"Hide":          Hide,
	"Concat":        Concat,
	"QueryString":   Trim,
	"ImagesView":    Trim,
	"ImagesEdit":    Trim,
	"Duration":      Trim,
	"HumanDataTime": Trim,
	"Mapping":       Trim,
	"Currency":      Trim,
}

// Trim string
func Trim(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("Trim args[0] is required")
	}

	if args[0] == nil {
		return "", nil
	}

	v, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("Trim args[0] is not a string value")
	}

	return strings.TrimSpace(v), nil
}

// Concat string
func Concat(args ...interface{}) (interface{}, error) {
	res := ""
	for _, arg := range args {
		if arg == nil {
			continue
		}
		res = fmt.Sprintf("%v%v", res, arg)
	}
	return res, nil
}

// Get value
func Get(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, nil
	}
	return args[0], nil
}

// Hide value
func Hide(args ...interface{}) (interface{}, error) {
	return nil, nil
}
