package component

import (
	"fmt"
	"net/url"
	"strings"
)

var hanlders = map[string]ComputeHanlder{
	"Get":           Get,
	"Trim":          Trim,
	"Hide":          Hide,
	"Concat":        Concat,
	"Download":      Download,
	"Upload":        Upload,
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

// Upload return the file download path
func Upload(args ...interface{}) (interface{}, error) {

	if len(args) < 5 {
		return nil, fmt.Errorf("Upload args[0]~args[4] is required")
	}

	if args[0] == nil {
		return "", nil
	}

	files := []string{}
	switch values := args[0].(type) {
	case []interface{}:
		for i := range values {
			file := fmt.Sprintf("%v", values[i])
			if file != "" {
				files = append(files, fmt.Sprintf("%v", file))
			}
		}
		break

	case []string:
		for _, file := range values {
			if file != "" {
				files = append(files, fmt.Sprintf("%v", file))
			}
		}
		break

	case string:
		if values != "" {
			files = append(files, fmt.Sprintf("%v", values))
		}
		break

	case map[string]interface{}:
		for name := range values {
			file := fmt.Sprintf("%v", values[name])
			if file != "" {
				files = append(files, fmt.Sprintf("%v", file))
			}
		}
		break
	}

	id, ok := args[3].(string)
	if !ok {
		return nil, fmt.Errorf("Upload args[3] is not string")
	}

	path, ok := args[4].(string)
	if !ok {
		return nil, fmt.Errorf("Upload args[4] is not string")
	}

	widget := "table"
	pinfo := strings.Split(path, ".")
	if len(pinfo) >= 2 {
		widget = pinfo[1]
	}

	preifx := fmt.Sprintf("/api/__yao/%s/%s/download/%s?name=", widget, id, url.QueryEscape(path))
	res := []string{}
	for _, file := range files {
		file = strings.TrimSpace(file)
		if strings.HasPrefix(file, "http") {
			res = append(res, file)
			continue
		}
		res = append(res, strings.TrimPrefix(file, preifx))
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res, nil
}

// Download return the file download path
func Download(args ...interface{}) (interface{}, error) {

	if len(args) < 5 {
		return nil, fmt.Errorf("Download args[0]~args[4] is required")
	}

	if args[0] == nil {
		return "", nil
	}

	files := []string{}
	switch values := args[0].(type) {
	case []interface{}:
		for i := range values {
			file := fmt.Sprintf("%v", values[i])
			if file != "" {
				files = append(files, fmt.Sprintf("%v", file))
			}
		}
		break

	case []string:
		for _, file := range values {
			if file != "" {
				files = append(files, fmt.Sprintf("%v", file))
			}
		}
		break

	case string:
		if values != "" {
			files = append(files, fmt.Sprintf("%v", values))
		}
		break

	case map[string]interface{}:
		for name := range values {
			file := fmt.Sprintf("%v", values[name])
			if file != "" {
				files = append(files, fmt.Sprintf("%v", file))
			}
		}
		break
	}

	id, ok := args[3].(string)
	if !ok {
		return nil, fmt.Errorf("Download args[3] is not string")
	}

	path, ok := args[4].(string)
	if !ok {
		return nil, fmt.Errorf("Download args[4] is not string")
	}

	widget := "table"
	pinfo := strings.Split(path, ".")
	if len(pinfo) >= 2 {
		widget = pinfo[1]
	}

	res := []string{}
	for _, file := range files {

		file = strings.TrimSpace(file)
		if strings.HasPrefix(file, "http") {
			res = append(res, file)
			continue
		}

		file = fmt.Sprintf("/api/__yao/%s/%s/download/%s?name=%s", widget, id, url.QueryEscape(path), file)
		res = append(res, file)
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res, nil
}
