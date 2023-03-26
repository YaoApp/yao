package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/yao/helper"
)

// Request request
type Request struct {
	host    string
	port    int
	route   string
	method  string
	data    map[string]interface{}
	params  map[string]string
	headers map[string]string
}

// Response response
type Response struct {
	status int
	body   []byte
}

// NewRequest create a new request
func NewRequest(port int) *Request {
	return &Request{
		host:    "127.0.0.1",
		port:    port,
		data:    map[string]interface{}{},
		params:  map[string]string{},
		headers: map[string]string{},
	}
}

// Token set token
func (r *Request) Token(token string) *Request {
	r.headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
	return r
}

// Header set header
func (r *Request) Header(key string, value string) *Request {
	r.headers[key] = value
	return r
}

// Param set saram
func (r *Request) Param(key string, value string) *Request {
	r.params[key] = value
	return r
}

// Data set data
func (r *Request) Data(data map[string]interface{}) *Request {
	r.data = data
	return r
}

// Route set the route
func (r *Request) Route(route string) *Request {
	r.route = route
	return r
}

// Get request
func (r *Request) Get() (*Response, error) {
	r.method = "GET"
	return r.Send()
}

// Post request
func (r *Request) Post() (*Response, error) {
	r.method = "POST"
	return r.Send()
}

// Send request
func (r *Request) Send() (*Response, error) {

	client := http.Client{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// set body
	var data io.Reader = nil
	if len(r.data) > 0 {
		content, err := jsoniter.Marshal(r.data)
		if err != nil {
			return nil, err
		}
		data = bytes.NewBuffer(content)
	}

	url := fmt.Sprintf("http://%s:%d%s", r.host, r.port, r.route)
	req, err := http.NewRequestWithContext(ctx, r.method, url, data)
	if err != nil {
		return nil, err
	}

	// Set header
	for key, value := range r.headers {
		req.Header.Add(key, value)
	}

	if _, has := r.headers["Content-Type"]; !has {
		req.Header.Add("Content-Type", "application/json")
	}

	// Set Parms
	if len(r.params) > 0 {
		q := req.URL.Query()
		for key, value := range r.params {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	// Send Request
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	p := &Response{
		status: res.StatusCode,
		body:   body,
	}

	return p, nil
}

// Map to map
func (p *Response) Map() (map[string]interface{}, error) {
	v := map[string]interface{}{}
	err := jsoniter.Unmarshal(p.body, &v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Int to int
func (p *Response) Int() (int, error) {
	v := 0
	err := jsoniter.Unmarshal(p.body, &v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// Status get the reaponse status
func (p *Response) Status() int {
	return p.status
}

// Body get the reaponse body
func (p *Response) Body() string {
	return string(p.body)
}

// To cast to custom sturct
func (p *Response) To(v interface{}) error {
	err := jsoniter.Unmarshal(p.body, v)
	if err != nil {
		return err
	}
	return nil
}

// AutoLogin auto login
func AutoLogin(id int) (map[string]interface{}, error) {

	user := model.Select("admin.user")
	row, err := user.Find(id, model.QueryParam(model.QueryParam{Select: []interface{}{"id", "name", "type", "email", "mobile", "extra", "status"}}))
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Unix() + 3600
	sid := session.ID()
	token := helper.JwtMake(id, map[string]interface{}{}, map[string]interface{}{
		"expires_at": expiresAt,
		"sid":        sid,
		"issuer":     "admin",
	})
	session.Global().Expire(time.Duration(token.ExpiresAt)*time.Second).ID(sid).Set("user_id", id)
	session.Global().ID(sid).Set("user", row)
	session.Global().ID(sid).Set("issuer", "admin")

	p, err := process.Of("yao.app.menu")
	if err != nil {
		return nil, err
	}

	menus, err := p.Exec()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"expires_at": token.ExpiresAt,
		"token":      token.Token,
		"user":       row,
		"menus":      menus,
	}, nil
}
