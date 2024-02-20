package api

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
)

// Request is the request for the page API.
type Request struct {
	File string
	*core.Request
	context *gin.Context
}

// NewRequestContext is the constructor for Request.
func NewRequestContext(c *gin.Context) (*Request, int, error) {

	file, params, err := parserPath(c)
	if err != nil {
		return nil, 404, err
	}

	payload, body, err := payload(c)
	if err != nil {
		return nil, 500, err
	}

	schema := c.Request.URL.Scheme
	if schema == "" {
		schema = "http"
	}

	domain := c.Request.URL.Hostname()
	if domain == "" {
		domain = strings.Split(c.Request.Host, ":")[0]
	}

	path := strings.TrimSuffix(c.Request.URL.Path, ".sui")

	return &Request{
		File:    file,
		context: c,
		Request: &core.Request{
			Method:  c.Request.Method,
			Query:   c.Request.URL.Query(),
			Body:    body,
			Payload: payload,
			Referer: c.Request.Referer(),
			Headers: url.Values(c.Request.Header),
			Params:  params,
			URL: core.ReqeustURL{
				URL:    fmt.Sprintf("%s://%s%s", schema, c.Request.Host, path),
				Host:   c.Request.Host,
				Path:   path,
				Domain: domain,
				Scheme: schema,
			},
		},
	}, 200, nil
}

// Render is the response for the page API.
func (r *Request) Render() (string, int, error) {

	c := core.GetCache(r.File)
	c = nil // disable cache @todo disable cache on development
	if c == nil {
		// Read the file
		content, err := application.App.Read(r.File)
		if err != nil {
			return "", 404, err
		}

		doc, err := core.NewDocument(content)
		if err != nil {
			return "", 500, err
		}

		guard := ""
		guardRedirect := ""
		configText := ""
		configSel := doc.Find("script[name=config]")
		if configSel != nil && configSel.Length() > 0 {
			configText = configSel.Text()
			configSel.Remove()

			var conf core.PageConfig
			err := jsoniter.UnmarshalFromString(configText, &conf)
			if err != nil {
				return "", 500, fmt.Errorf("config error, please re-complie the page %s", err.Error())
			}

			// Redirect the page (should refector before release)
			// guard=cookie-jwt:redirect-url redirect to the url if not authorized
			// guard=cookie-jwt return {code: 403, message: "Not Authorized"}
			guard = conf.Guard
			if strings.Contains(conf.Guard, ":") {
				parts := strings.Split(conf.Guard, ":")
				guard = parts[0]
				guardRedirect = parts[1]
			}
		}

		dataText := ""
		dataSel := doc.Find("script[name=data]")
		if dataSel != nil && dataSel.Length() > 0 {
			dataText = dataSel.Text()
			dataSel.Remove()
		}

		globalDataText := ""
		globalDataSel := doc.Find("script[name=global]")
		if globalDataSel != nil && globalDataSel.Length() > 0 {
			globalDataText = globalDataSel.Text()
			globalDataSel.Remove()
		}

		html, err := doc.Html()
		if err != nil {
			return "", 500, fmt.Errorf("parse error, please re-complie the page %s", err.Error())
		}

		// Save to The Cache
		// c = core.SetCache(r.File, html, dataText, globalDataText)
		c = &core.Cache{
			Data:          dataText,
			Global:        globalDataText,
			HTML:          html,
			Guard:         guard,
			GuardRedirect: guardRedirect,
			Config:        configText,
		}
		log.Trace("The page %s is cached", r.File)
	}

	// Guard the page
	if c.Guard != "" && r.context != nil {

		if guard, has := Guards[c.Guard]; has {
			err := guard(r)
			if err != nil {

				// Redirect the page (should refector before release)
				if c.GuardRedirect != "" {
					redirect := c.GuardRedirect
					data := core.Data{}
					if c.Data != "" {
						data, err = r.Request.ExecString(c.Data)
						if err != nil {
							return "", 500, fmt.Errorf("data error, please re-complie the page %s", err.Error())
						}
					}

					if c.Global != "" {
						global, err := r.Request.ExecString(c.Global)
						if err != nil {
							return "", 500, fmt.Errorf("global data error, please re-complie the page %s", err.Error())
						}
						data["$global"] = global
					}

					redirect, _ = data.Replace(redirect)
					return "", 302, fmt.Errorf("%s", redirect)
				}

				// Return the error
				ex := exception.Err(err, 403)
				return "", ex.Code, fmt.Errorf("%s", ex.Message)
			}
		} else {
			// Process the guard
			err := r.processGuard(c.Guard)
			if err != nil {
				ex := exception.Err(err, 403)
				return "", ex.Code, fmt.Errorf("%s", ex.Message)
			}
		}
	}

	var err error
	data := core.Data{}
	if c.Data != "" {
		data, err = r.Request.ExecString(c.Data)
		if err != nil {
			return "", 500, fmt.Errorf("data error, please re-complie the page %s", err.Error())
		}
	}

	if c.Global != "" {
		global, err := r.Request.ExecString(c.Global)
		if err != nil {
			return "", 500, fmt.Errorf("global data error, please re-complie the page %s", err.Error())
		}
		data["$global"] = global
	}

	// Set the page request data
	data["$payload"] = r.Request.Payload
	data["$query"] = r.Request.Query
	data["$param"] = r.Request.Params
	data["$url"] = r.Request.URL

	printData := false
	if r.Query != nil && r.Query.Has("__sui_print_data") {
		printData = true
	}

	parser := core.NewTemplateParser(data, &core.ParserOption{PrintData: printData, Request: true})
	html, err := parser.Render(c.HTML)
	if err != nil {
		return "", 500, fmt.Errorf("render error, please re-complie the page %s", err.Error())
	}

	return html, 200, nil
}

func parserPath(c *gin.Context) (string, map[string]string, error) {

	params := map[string]string{}

	parts := strings.Split(strings.TrimSuffix(c.Request.URL.Path, ".sui"), "/")[1:]
	if len(parts) < 1 {
		return "", nil, fmt.Errorf("path parts error: %s", strings.Join(parts, "/"))
	}

	fileParts := []string{string(os.PathSeparator), "public"}

	// Match the sui
	matchers := core.RouteExactMatchers[parts[0]]
	if matchers == nil {
		for matcher, reMatchers := range core.RouteMatchers {
			matched := matcher.FindStringSubmatch(parts[0])
			if len(matched) > 0 {
				matchers = reMatchers
				fileParts = append(fileParts, matched[0])
				break
			}
		}
	}

	// No matchers
	if matchers == nil {
		if len(parts) < 1 {
			return "", nil, fmt.Errorf("path parts error: %s", strings.Join(parts, "/"))
		}

		fileParts = append(fileParts, parts...)
		return filepath.Join(fileParts...) + ".sui", params, nil
	}

	// Match the page parts
	for i, part := range parts[1:] {
		if len(matchers) < i+1 {
			return "", nil, fmt.Errorf("matchers length error %d < %d", len(matchers), i+1)
		}

		parent := ""
		if i > 0 {
			parent = parts[i]
		}
		matched := false
		for _, matcher := range matchers[i] {

			// Filter the parent
			if matcher.Parent != "" && matcher.Parent != parent {
				continue
			}

			if matcher.Exact == part {
				fileParts = append(fileParts, matcher.Exact)
				matched = true
				break

			} else if matcher.Regex != nil {
				if matcher.Regex.MatchString(part) {
					file := matcher.Ref
					key := strings.TrimRight(strings.TrimLeft(file, "["), "]")
					params[key] = part
					fileParts = append(fileParts, file)
					matched = true
					break
				}
			}
		}

		if !matched {
			return "", nil, fmt.Errorf("route does not match")
		}
	}
	return filepath.Join(fileParts...) + ".sui", params, nil
}

func params(c *gin.Context) map[string]string {
	return nil
}

func payload(c *gin.Context) (map[string]interface{}, interface{}, error) {
	contentType := c.Request.Header.Get("Content-Type")
	var payload map[string]interface{}
	var body interface{}

	switch contentType {
	case "application/x-www-form-urlencoded":
		c.Request.ParseForm()
		payload = make(map[string]interface{})
		for key, value := range c.Request.Form {
			payload[key] = value
		}
		body = nil
		break

	case "multipart/form-data":
		c.Request.ParseMultipartForm(32 << 20)
		payload = make(map[string]interface{})
		for key, value := range c.Request.MultipartForm.Value {
			payload[key] = value
		}
		body = nil
		break

	case "application/json":
		if c.Request.Body == nil {
			return nil, nil, nil
		}

		c.Bind(&payload)
		body = nil
		break

	default:
		if c.Request.Body == nil {
			return nil, nil, nil
		}

		var data []byte
		_, err := c.Request.Body.Read(data)
		if err != nil && err.Error() != "EOF" {
			return nil, nil, err
		}
		body = data
	}

	return payload, body, nil
}
