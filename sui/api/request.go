package api

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
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

var reRouteVar = regexp.MustCompile(`\[([0-9a-z_]+)\]`)

// NewRequestContext is the constructor for Request.
func NewRequestContext(c *gin.Context) (*Request, int, error) {

	file, params, err := parserPath(c)
	if err != nil {
		return nil, 404, err
	}

	log.Trace("[Request] %s params:%v", file, params)
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

	// Read content from cache
	var c *core.Cache = nil
	if !r.Request.DisableCache() {
		c = core.GetCache(r.File)
	}

	if c == nil {

		// The page is not cached
		message := fmt.Sprintf("[SUI] The page %s is not cached. file=%s DisableCache=%v", r.Request.URL.Path, r.File, r.Request.DisableCache())
		fmt.Println(color.YellowString(message))
		log.Warn(message)

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
		c = &core.Cache{
			Data:          dataText,
			Global:        globalDataText,
			HTML:          html,
			Guard:         guard,
			GuardRedirect: guardRedirect,
			Config:        configText,
		}
		core.SetCache(r.File, c)
		log.Trace("[SUI] The page %s is cached file=%s", r.Request.URL.Path, r.File)
	}

	// Guard the page
	code, err := r.Guard(c)
	if err != nil {
		return "", code, err
	}

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
	r.Request.WithData(data)
	option := core.ParserOption{
		Theme:   data["$theme"].(string),
		Lang:    data["$lang"].(string),
		Debug:   r.Request.DebugMode(),
		Request: true,
	}

	// Parse the template
	parser := core.NewTemplateParser(data, &option)
	html, err := parser.Render(c.HTML)
	if err != nil {
		return "", 500, fmt.Errorf("render error, please re-complie the page %s", err.Error())
	}

	return html, 200, nil
}

// Guard the page
func (r *Request) Guard(c *core.Cache) (int, error) {

	// Guard not set
	if c.Guard == "" || r.context == nil {
		return 200, nil
	}

	// Built-in guard
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
						return 500, fmt.Errorf("data error, please re-complie the page %s", err.Error())
					}
				}

				if c.Global != "" {
					global, err := r.Request.ExecString(c.Global)
					if err != nil {
						return 500, fmt.Errorf("global data error, please re-complie the page %s", err.Error())
					}
					data["$global"] = global
				}

				redirect, _ = data.Replace(redirect)
				return 302, fmt.Errorf("%s", redirect)
			}

			// Return the error
			ex := exception.Err(err, 403)
			return ex.Code, fmt.Errorf("%s", ex.Message)
		}
		return 200, nil
	}

	// Developer custom guard
	err := r.processGuard(c.Guard)
	if err != nil {
		ex := exception.Err(err, 403)
		return ex.Code, fmt.Errorf("%s", ex.Message)
	}

	return 200, nil
}

func parserPath(c *gin.Context) (string, map[string]string, error) {

	params := map[string]string{}
	parts := strings.Split(strings.TrimSuffix(c.Request.URL.Path, ".sui"), "/")[1:]
	if len(parts) < 1 {
		return "", nil, fmt.Errorf("path parts error: %s", strings.Join(parts, "/"))
	}

	fileParts := []string{string(os.PathSeparator), "public"}
	fileParts = append(fileParts, parts...)
	filename := filepath.Join(fileParts...) + ".sui"

	v, _ := c.Get("rewrite")
	if v != true {
		return filename, params, nil
	}

	// Find the [xxx] in the path
	matchesValues, has := c.Get("matches")
	if !has {
		return filename, params, nil
	}

	values := matchesValues.([]string)
	matches := reRouteVar.FindAllStringSubmatch(c.Request.URL.Path, -1)
	valuesCnt := len(values)
	matchesCnt := len(matches)
	start := valuesCnt - matchesCnt
	if matchesCnt > 0 && start > 0 {
		for i, match := range matches {
			name := match[1]
			params[name] = values[start+i]
		}
	}
	return filename, params, nil
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
