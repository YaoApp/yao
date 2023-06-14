package widget

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestWidgetLoadInstances(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	for _, widget := range preare(t) {
		err := widget.LoadInstances()
		if err != nil {
			t.Fatal(err)
		}

		instance, ok := widget.Instances.Load("feedback")
		if !ok {
			t.Fatal("feedback instance not found")
		}

		assert.Equal(t, "feedback", instance.(*Instance).id)
		assert.Equal(t, "feedback", instance.(*Instance).dsl.(map[string]interface{})["id"])
	}
}

func TestWidgetReLoadInstances(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	for _, widget := range preare(t) {
		err := widget.LoadInstances()
		if err != nil {
			t.Fatal(err)
		}

		instance, ok := widget.Instances.Load("feedback")
		if !ok {
			t.Fatal("feedback instance not found")
		}

		err = widget.ReloadInstances()
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "feedback", instance.(*Instance).id)
		assert.Equal(t, "feedback", instance.(*Instance).dsl.(map[string]interface{})["id"])
		assert.Equal(t, true, instance.(*Instance).dsl.(map[string]interface{})["tests.reload"])
	}
}

func TestWidgetUnLoadInstances(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	for _, widget := range preare(t) {
		err := widget.LoadInstances()
		if err != nil {
			t.Fatal(err)
		}

		instance, ok := widget.Instances.Load("feedback")
		if !ok {
			t.Fatal("feedback instance not found")
		}

		assert.Equal(t, "feedback", instance.(*Instance).id)
		assert.Equal(t, "feedback", instance.(*Instance).dsl.(map[string]interface{})["id"])

		err = widget.UnloadInstances()
		if err != nil {
			t.Fatal(err)
		}

		_, ok = widget.Instances.Load("feedback")
		assert.False(t, ok)
	}
}

func TestWidgetRegisterProcess(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	for _, widget := range preare(t) {
		err := widget.LoadInstances()
		if err != nil {
			t.Fatal(err)
		}

		name := fmt.Sprintf("widgets.%s.Setting", widget.ID)
		res := process.New(name, "feedback").Run()
		assert.Equal(t, "feedback", res.(map[string]interface{})["id"])
		assert.Equal(t, "feedback", res.(map[string]interface{})["tests.id"])
	}
}

func TestWidgetRegisterAPI(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	for _, widget := range preare(t) {
		err := widget.LoadInstances()
		if err != nil {
			t.Fatal(err)
		}

		router := testRouter(t)
		response := httptest.NewRecorder()
		url := fmt.Sprintf("/api/__yao/widget/%s/feedback/setting", widget.ID)
		req, _ := http.NewRequest("GET", url, nil)
		router.ServeHTTP(response, req)

		res := map[string]interface{}{}
		err = jsoniter.Unmarshal(response.Body.Bytes(), &res)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "feedback", res["id"])
	}
}

func TestWidgetSaveCreate(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	dyform := preare(t)[0]
	iform := preare(t)[1]

	err := dyform.Save("feedback/new.form.yao", map[string]interface{}{})
	assert.NotEmpty(t, err)

	err = iform.Save("feedback/new.form.yao", map[string]interface{}{"columns": []interface{}{}})
	if err != nil {
		t.Fatal(err)
	}
	defer iform.Remove("feedback/new.form.yao")

	instance, ok := iform.Instances.Load("feedback.new")
	if !ok {
		t.Fatal("feedback instance not found")
	}
	assert.Equal(t, "feedback.new", instance.(*Instance).id)
}

func TestWidgetSaveUpdate(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()
	iform := preare(t)[1]

	err := iform.Save("feedback/new.form.yao", map[string]interface{}{"columns": []interface{}{}})
	if err != nil {
		t.Fatal(err)
	}
	defer iform.Remove("feedback/new.form.yao")

	err = iform.Save("feedback/new.form.yao", map[string]interface{}{"columns": []interface{}{}, "foo": "bar"})
	if err != nil {
		t.Fatal(err)
	}

	instance, ok := iform.Instances.Load("feedback.new")
	if !ok {
		t.Fatal("feedback instance not found")
	}

	assert.Equal(t, "feedback.new", instance.(*Instance).id)
	assert.Equal(t, "bar", instance.(*Instance).dsl.(map[string]interface{})["foo"])
	assert.Equal(t, true, instance.(*Instance).dsl.(map[string]interface{})["tests.reload"])
}

func preare(t *testing.T) []*DSL {
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}

	qb := capsule.Global.Query()
	qb.Table("dsl_iform").Insert(map[string]interface{}{
		"file": "feedback.iform.yao",
		"source": `{
			"columns": [
			  [
				{ "type": "Title", "label": "Feedback Information" },
				{ "type": "Input", "label": "Name" },
				{ "type": "Input", "label": "Email" }
			  ],
			  [
				{ "type": "Title", "label": "Feedback Details" },
				{ "type": "Textarea", "label": "Message" },
				{ "type": "Checkbox", "label": "Anonymous" }
			  ]
			],
			"actions": {
			  "left": [
				{
				  "type": "api",
				  "text": "Submit Feedback",
				  "api": "/api/__yao/widget/dyform/save",
				  "isPrimary": true
				}
			  ],
			  "right": [
				{
				  "type": "info",
				  "text": "Help",
				  "info": "Need assistance? Click here."
				},
				{
				  "type": "api",
				  "text": "Cancel",
				  "process": "widget.dyform.Cancel"
				}
			  ]
			}
		  }
		  `,
	})

	return []*DSL{Widgets["dyform"], Widgets["iform"]}
}

func testRouter(t *testing.T, middlewares ...gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	gin.SetMode(gin.ReleaseMode)
	router.Use(middlewares...)
	api.SetGuards(map[string]gin.HandlerFunc{"bearer-jwt": func(ctx *gin.Context) {}})
	api.SetRoutes(router, "/api")
	return router
}
