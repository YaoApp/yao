package table

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

var guards = map[string]gin.HandlerFunc{
	"bearer-jwt":   test.GuardBearerJWT,
	"widget-table": Guard,
}

func TestAPISetting(t *testing.T) {

	port := start(t)
	defer stop()

	req := test.NewRequest(port).Route("/api/__yao/table/pet/setting")
	res, err := req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 403, res.Status())

	req = test.NewRequest(port).Route("/api/__yao/table/pet/setting").Token(token(t))
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())

	v, err := res.Map()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(v).MapStr().Dot()
	assert.Equal(t, "/api/xiang/import/pet", data.Get("header.preset.import.api.import"))
	// assert.Equal(t, "跳转", data.Get("header.preset.import.operation.0.title"))
	assert.Equal(t, "/api/__yao/table/pet/component/fields.table."+url.QueryEscape("入院状态")+".view.props.xProps/remote", data.Get("fields.table.入院状态.view.props.xProps.remote.api"))
	assert.Equal(t, "/api/__yao/table/pet/component/fields.table."+url.QueryEscape("入院状态")+".edit.props.xProps/remote", data.Get("fields.table.入院状态.edit.props.xProps.remote.api"))
}

func TestAPISearch(t *testing.T) {
	port := start(t)
	defer test.Stop()

	req := test.NewRequest(port).Route("/api/__yao/table/session/search")
	res, err := req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 403, res.Status())

	req = test.NewRequest(port).Route("/api/__yao/table/session/search").Token(token(t))
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
	resp, err := res.Map()
	if err != nil {
		t.Fatal(err)
	}

	data := any.Of(resp).MapStr().Dot()
	assert.Equal(t, "1", fmt.Sprintf("%v", data.Get("pagesize")))
	assert.Equal(t, "3", fmt.Sprintf("%v", data.Get("total")))
	assert.Equal(t, "checked", data.Get("data.0.status"))
	assert.Equal(t, "enabled", data.Get("data.0.mode"))
	assert.Equal(t, "1", fmt.Sprintf("%v", data.Get("data.0.doctor_id")))

}

func TestAPISave(t *testing.T) {
	port := start(t)
	defer test.Stop()

	payload := map[string]interface{}{
		"name":      "New Pet",
		"type":      "cat",
		"status":    "checked",
		"mode":      "enabled",
		"stay":      66,
		"cost":      24,
		"doctor_id": 1,
	}

	req := test.NewRequest(port).Route("/api/__yao/table/pet/save").Data(payload)
	res, err := req.Post()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 403, res.Status())

	req = test.NewRequest(port).Route("/api/__yao/table/pet/save").Data(payload).Token(token(t))
	res, err = req.Post()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())

	v, err := res.Int()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 4, v)
}

func TestAPICustomGuard(t *testing.T) {

	port := start(t)
	defer test.Stop()

	req := test.NewRequest(port).Route("/api/__yao/table/pet/find/1")
	res, err := req.Get()
	if err != nil {
		t.Fatal(err)
	}

	req = test.NewRequest(port).Route("/api/__yao/table/pet/get")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 403, res.Status())

	req = test.NewRequest(port).Route("/api/__yao/table/pet/get").Token(token(t)).Header("Unit-Test", "yes")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 418, res.Status())

	req = test.NewRequest(port).Route("/api/__yao/table/pet/get").Token(token(t))
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
}

func TestAPIGlobalCustomGuard(t *testing.T) {

	port := start(t)
	defer test.Stop()

	req := test.NewRequest(port).Route("/api/__yao/table/guard/find/1")
	res, err := req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 403, res.Status())

	req = test.NewRequest(port).Route("/api/__yao/table/guard/get")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 403, res.Status())

	req = test.NewRequest(port).Route("/api/__yao/table/guard/get").Token(token(t)).Header("Unit-Test", "yes")
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 418, res.Status())

	req = test.NewRequest(port).Route("/api/__yao/table/guard/get").Token(token(t))
	res, err = req.Get()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, res.Status())
}

func start(t *testing.T) int {
	test.Prepare(t, config.Conf)
	prepare(t)
	clear(t)
	testData(t)
	test.Start(t, guards, config.Conf)

	return test.Port(t)
}

func stop() {
	test.Stop()
}

func token(t *testing.T) string {
	res, err := test.AutoLogin(1)
	if err != nil {
		t.Fatal(err)
	}

	token, ok := res["token"].(string)
	if !ok {
		t.Fatal(fmt.Errorf("get token error %v", res))
	}
	return token
}
