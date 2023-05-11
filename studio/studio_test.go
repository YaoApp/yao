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
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
	"github.com/yaoapp/yao/test"
)

type kv map[string]interface{}
type arr []interface{}

func TestLoad(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	// res, err := gou.Yao.Engine.RootCall(map[string]interface{}{}, "table", "Ping")
	// assert.Nil(t, err)
	// assert.Equal(t, "PONG", res)

	// _, err = gou.Yao.Engine.Call(map[string]interface{}{}, "table", "Ping")
	// assert.NotNil(t, err)
	// assert.Contains(t, err.Error(), "The table does not loaded")
}

func TestStartStop(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

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

	test.Prepare(t, config.Conf)
	defer test.Clean()

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

func TestAPI(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	Load(config.Conf)

	var err error
	go func() { err = Start(config.Conf) }()
	if err != nil {
		t.Fatal(err)
	}
	defer Stop()
	time.Sleep(500 * time.Millisecond)

	code, row := httpGet[kv]("/dsl/ReadFile?name=/models/user.mod.yao", t)
	assert.Equal(t, 200, code)
	assert.Equal(t, "User", row["name"])

	code, rows := httpGet[arr]("/dsl/ReadDir?name=/models", t)
	assert.Equal(t, 200, code)
	assert.Equal(t, 8, len(rows))

	code, rows = httpGet[arr]("/dsl/ReadDir?name=/models&recursive=1", t)
	assert.Equal(t, 200, code)
	assert.Equal(t, 13, len(rows))

	code, length := httpPost[int]("/dsl/WriteFile?name=/models/foo.mod.yao", []byte(`{"name":"foo"}`), t)
	assert.Equal(t, 200, code)
	assert.Equal(t, 19, length)

	code, _ = httpPost[kv]("/dsl/Remove?name=/models/foo.mod.yao", nil, t)
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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	token := getToken(t)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if res.Body != nil {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		if body != nil && len(body) > 0 {
			err = jsoniter.Unmarshal(body, &data)
			if err != nil {
				t.Fatal(fmt.Sprintf("%s\n%s\n", err.Error(), string(body)))
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
	req, err := http.NewRequest("POST", url, buff)
	if err != nil {
		t.Fatal(err)
	}

	token := getToken(t)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := http.Client{}
	res, err := client.Do(req)
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

func getToken(t *testing.T) string {
	return helper.JwtMake(
		1,
		map[string]interface{}{"id": 1, "user_id": 1, "user": kv{"id": 1, "name": "test"}},
		map[string]interface{}{"issuer": "unit-test", "timeout": 3600},
		[]byte(config.Conf.Studio.Secret),
	).Token
}
