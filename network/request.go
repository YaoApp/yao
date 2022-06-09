package network

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/dns"
)

// Response 请求响应结果
type Response struct {
	Status  int                    `json:"status"`
	Body    string                 `json:"body"`
	Data    interface{}            `json:"data"`
	Headers map[string]interface{} `json:"headers"`
}

// RequestGet 发送GET请求
func RequestGet(url string, params map[string]interface{}, headers map[string]string) Response {
	return RequestSend("GET", url, params, nil, headers)
}

// RequestPost 发送POST请求
func RequestPost(url string, data interface{}, headers map[string]string) Response {
	return RequestSend("POST", url, map[string]interface{}{}, data, headers)
}

// RequestPostJSON 发送POST请求
func RequestPostJSON(url string, data interface{}, headers map[string]string) Response {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["content-type"] = "application/json;charset=utf8"
	return RequestSend("POST", url, map[string]interface{}{}, data, headers)
}

// RequestPut 发送PUT请求
func RequestPut(url string, data interface{}, headers map[string]string) Response {
	return RequestSend("PUT", url, map[string]interface{}{}, data, headers)
}

// RequestPutJSON 发送PUT请求
func RequestPutJSON(url string, data interface{}, headers map[string]string) Response {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["content-type"] = "application/json;charset=utf8"
	return RequestSend("PUT", url, map[string]interface{}{}, data, headers)
}

// RequestSend 发送Request请求
func RequestSend(method string, url string, params map[string]interface{}, data interface{}, headers map[string]string) Response {

	var body []byte
	var err error
	if data != nil {
		if strings.HasPrefix(strings.ToLower(headers["content-type"]), "application/json") {
			body, err = jsoniter.Marshal(data)
			if err != nil {
				return Response{
					Status: 500,
					Body:   err.Error(),
					Data:   map[string]interface{}{"code": 500, "message": err.Error()},
					Headers: map[string]interface{}{
						"Content-Type": "application/json;charset=utf8",
					},
				}
			}
		} else {
			body = []byte(fmt.Sprintf("%v", data))
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return Response{
			Status: 500,
			Body:   err.Error(),
			Data:   map[string]interface{}{"code": 500, "message": err.Error()},
			Headers: map[string]interface{}{
				"Content-Type": "application/json;charset=utf8",
			},
		}
	}

	// Request Header
	if headers != nil {
		for name, header := range headers {
			req.Header.Set(name, header)
		}
	}

	// Force using system DSN resolver
	// var dialer = &net.Dialer{Resolver: &net.Resolver{PreferGo: false}}
	var dialContext = dns.DialContext()
	var tr = &http.Transport{DialContext: dialContext}
	var client *http.Client = &http.Client{Transport: tr}

	// Https SkipVerify false
	if strings.HasPrefix(url, "https://") {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext:     dialContext,
		}
		client = &http.Client{Transport: tr}
	}
	defer tr.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		return Response{
			Status: 0,
			Body:   err.Error(),
			Data:   map[string]interface{}{"code": 500, "message": err.Error()},
			Headers: map[string]interface{}{
				"Content-Type": "application/json;charset=utf8",
			},
		}
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		return Response{
			Status: 500,
			Body:   err.Error(),
			Data:   map[string]interface{}{"code": resp.StatusCode, "message": err.Error()},
			Headers: map[string]interface{}{
				"Content-Type": "application/json;charset=utf8",
			},
		}
	}

	// JSON 解析
	var res interface{}
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		err = jsoniter.Unmarshal(body, &res)
		if err != nil {
			return Response{
				Status: 500,
				Body:   err.Error(),
				Data:   map[string]interface{}{"code": resp.StatusCode, "message": err.Error()},
				Headers: map[string]interface{}{
					"Content-Type": "application/json;charset=utf8",
				},
			}
		}
	}
	respHeaders := map[string]interface{}{}
	for name := range resp.Header {
		respHeaders[name] = resp.Header.Get(name)
	}
	return Response{
		Status:  resp.StatusCode,
		Body:    string(body),
		Data:    res,
		Headers: respHeaders,
	}
}
