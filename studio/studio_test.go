package studio

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
)

type kv map[string]interface{}
type arr []interface{}

func TestLoad(t *testing.T) {
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, dfs)

	res, err := gou.Yao.Engine.Call(map[string]interface{}{}, "__yao.studio.table", "Ping")
	assert.Nil(t, err)
	assert.Equal(t, "PONG", res)
}

func TestStartStop(t *testing.T) {
	var err error
	go func() { err = Start(config.Conf) }()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)
	Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestStartStopError(t *testing.T) {
	var err error
	go func() { err = Start(config.Conf) }()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)
	go func() { err = Start(config.Conf) }()
	time.Sleep(100 * time.Millisecond)
	assert.NotNil(t, err)

	Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestGetAPI(t *testing.T) {

	Load(config.Conf)

	var err error
	go func() { err = Start(config.Conf) }()
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	time.Sleep(500 * time.Millisecond)

	code, row := httpGet[kv]("/dsl/ReadFile?name=/models/user.json", t)
	assert.Equal(t, 200, code)
	assert.Equal(t, "用户", row["name"])

	code, rows := httpGet[arr]("/dsl/ReadDir?name=/models", t)
	assert.Equal(t, 200, code)
	assert.Equal(t, 11, len(rows))

	code, rows = httpGet[arr]("/dsl/ReadDir?name=/models&recursive=1", t)
	assert.Equal(t, 200, code)
	assert.Equal(t, 12, len(rows))

	code, length := httpPost[int]("/dsl/WriteFile?name=/models/foo.mod.json", []byte(`{"name":"foo"}`), t)
	assert.Equal(t, 200, code)
	assert.Equal(t, 19, length)

	code, _ = httpPost[kv]("/dsl/Remove?name=/models/foo.mod.json", nil, t)
	assert.Equal(t, 200, code)

	code, _ = httpPost[kv]("/dsl/Mkdir?name=/models/bar", nil, t)
	assert.Equal(t, 200, code)

	code, _ = httpPost[kv]("/dsl/Remove?name=/models/bar", nil, t)
	assert.Equal(t, 200, code)

	code, _ = httpPost[kv]("/dsl/MkdirAll?name=/models/bar/hi", nil, t)
	assert.Equal(t, 200, code)

	code, _ = httpPost[kv]("/dsl/RemoveAll?name=/models/bar", nil, t)
	assert.Equal(t, 200, code)

	code, res := httpPostJSON[arr](
		"/service/table",
		kv{
			"method": "UnitTest",
			"args": []interface{}{
				"foo", 1, 0.618,
				kv{"string": "world", "int": 1, "float": 0.618},
				arr{"foo", 1, 0.618},
			},
		}, t)

	assert.Equal(t, 200, code)
	assert.Equal(t, "foo", res[0])
	assert.Equal(t, float64(1), res[1])
	assert.Equal(t, 0.618, res[2])
	assert.Equal(t, "world", res[3].(map[string]interface{})["string"])
	assert.Equal(t, float64(1), res[3].(map[string]interface{})["int"])
	assert.Equal(t, 0.618, res[3].(map[string]interface{})["float"])
	assert.Equal(t, "foo", res[4].([]interface{})[0])
	assert.Equal(t, float64(1), res[4].([]interface{})[1])
	assert.Equal(t, 0.618, res[4].([]interface{})[2])

	code, excp := httpPostJSON[kv]("/service/table", kv{"method": "UnitTest", "args": []interface{}{"throw-test"}}, t)
	assert.Equal(t, 418, code)
	assert.Equal(t, float64(418), excp["code"])
	assert.Equal(t, "I'm a teapot", excp["message"])
}

func httpGet[T kv | arr | interface{} | map[string]interface{} | int | []interface{}](url string, t *testing.T) (int, T) {

	var data T
	url = fmt.Sprintf("http://127.0.0.1:%d%s", config.Conf.Studio.Port, url)
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	if res.Body != nil {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		if body != nil {
			err = jsoniter.Unmarshal(body, &data)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	return res.StatusCode, data
}

func httpPost[T kv | arr | interface{} | map[string]interface{} | int | []interface{}](url string, payload []byte, t *testing.T) (int, T) {

	var data T
	var buff *bytes.Buffer = bytes.NewBuffer([]byte{})

	if payload != nil {
		buff = bytes.NewBuffer(payload)
	}

	url = fmt.Sprintf("http://127.0.0.1:%d%s", config.Conf.Studio.Port, url)
	res, err := http.Post(url, "application/json", buff)
	if err != nil {
		t.Fatal(err)
	}

	if res.Body != nil {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		if body != nil && string(body) != "" {
			err = jsoniter.Unmarshal(body, &data)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	return res.StatusCode, data
}

func httpPostJSON[T kv | arr | interface{} | map[string]interface{} | int | []interface{}](url string, payload interface{}, t *testing.T) (int, T) {
	var data []byte
	var err error
	if payload != nil {
		data, err = jsoniter.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
	}
	return httpPost[T](url, data, t)
}
