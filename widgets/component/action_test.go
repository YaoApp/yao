package component

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestActionUnmarshalJSON(t *testing.T) {

	data := testActionData()
	var action ActionDSL
	err := jsoniter.Unmarshal(data["one"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 1)
	assert.Equal(t, "Delete", action.Action[0]["name"])
	assert.Equal(t, "Form.delete", action.Action[0]["type"])
	assert.Equal(t, "408ebbf0c51d7a51417c04ac73a0a1bc", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["many"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 2)
	assert.Equal(t, "Save", action.Action[0]["name"])
	assert.Equal(t, "Form.save", action.Action[0]["type"])
	assert.Equal(t, "historyPush", action.Action[1]["name"])
	assert.Equal(t, "Common.historyPush", action.Action[1]["type"])
	assert.Equal(t, "1c70ca190ae5259a37414f98ed9d86c3", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["flow"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 2)
	assert.Equal(t, "Save", action.Action[0]["name"])
	assert.Equal(t, "Form.save", action.Action[0]["type"])
	assert.Equal(t, "Flow", action.Action[1]["name"])
	assert.Equal(t, "Actions.test.check", action.Action[1]["type"])
	assert.Equal(t, "c6d4ae7f02cea12a236bbae38956179c", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-string"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 1)
	assert.Equal(t, "Actions.test.back", action.Action[0]["name"])
	assert.Equal(t, "Actions.test.back", action.Action[0]["type"])
	assert.Equal(t, "6188373a217ef9312bf14e6ca4b21fd2", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-map"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 1)
	assert.Equal(t, "Form.delete", action.Action[0]["name"])
	assert.Equal(t, "Form.delete", action.Action[0]["type"])
	assert.Equal(t, "1fe1bac887859171a97af154bb193821", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-map-custom"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 1)
	assert.Equal(t, "Actions.test.back", action.Action[0]["name"])
	assert.Equal(t, "Actions.test.back", action.Action[0]["type"])
	assert.Equal(t, "7daf632016d4d8ea77066bc54afd525e", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-hide"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, false, action.ShowWhenAdd)
	assert.Equal(t, false, action.ShowWhenView)
	assert.Equal(t, false, action.HideWhenEdit)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-disabled-eq"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "{{data}}", action.Disabled.Bind)
	assert.Equal(t, "1", action.Disabled.Value)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-disabled-equal"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "{{data}}", action.Disabled.Bind)
	assert.Equal(t, "1", action.Disabled.Value)
}

// testActionData
func testActionData() map[string][]byte {

	return map[string][]byte{
		"one": []byte(`{
			"title": "Delete",
			"icon": "icon-trash-2",
			"style": "danger",
			"action": [
			  {
				"name": "Delete",
				"type": "Form.delete",
				"payload": { "pathname": "/x/Table/env", "foo":"bar", "hello":"world" }
			  }
			],
			"confirm": { "title": "Tips", "desc": "Delete Confirm" }
		}`),

		"many": []byte(`{
			"title": "Cured",
			"icon": "icon-check",
			"style": "success",
			"action": [
			  {
				"name": "Save",
				"type": "Form.save",
				"payload": { "id": ":id", "status": "cured" }
			  },
			  {
                "name": "historyPush",
                "type": "Common.historyPush",
                "payload": { "pathname": "/x/Form/pet/:id/edit" }
              }
			],
			"confirm": { "title": "Tips", "desc": "Cured Confirm" }
		}`),

		"flow": []byte(`{
			"title": "Delete",
			"icon": "icon-trash-2",
			"style": "danger",
			"action": [
				{
				  "name": "Save",
				  "type": "Form.save",
				  "payload": { "id": ":id", "status": "cured" }
				},
				{
				  "name": "Flow",
				  "type": "Actions.test.check",
				  "payload": { "pathname": "/x/Form/pet/:id/edit" }
				}
			],
			"confirm": { "title": "Tips", "desc": "Delete Confirm" }
		}`),

		"sugar-string": []byte(`{
			"title": "Delete",
			"icon": "icon-trash-2",
			"style": "danger",
			"action": "Actions.test.back",
			"confirm": { "title": "Tips", "desc": "Delete Confirm" }
		}`),

		"sugar-map": []byte(`{
			"title": "Delete",
			"icon": "icon-trash-2",
			"style": "danger",
			"action": {
				"Form.delete": {  "pathname": "/x/Table/env" }
			},
			"confirm": { "title": "Tips", "desc": "Delete Confirm" }
		}`),

		"sugar-map-custom": []byte(`{
			"title": "Delete",
			"icon": "icon-trash-2",
			"style": "danger",
			"action": {
				"Actions.test.back": {  "pathname": "/x/Table/env" }
			},
			"confirm": { "title": "Tips", "desc": "Delete Confirm" }
		}`),

		"sugar-hide": []byte(`{
			"title": "Delete",
			"icon": "icon-trash-2",
			"style": "danger",
			"hide": ["view", "add"],
			"action": {
				"Actions.test.back": {  "pathname": "/x/Table/env" }
			},
			"confirm": { "title": "Tips", "desc": "Delete Confirm" }
		}`),

		"sugar-disabled-eq": []byte(`{
			"title": "Delete",
			"icon": "icon-trash-2",
			"style": "danger",
			"hide": ["view", "add"],
			"action": {
				"Actions.test.back": {  "pathname": "/x/Table/env" }
			},
			"confirm": { "title": "Tips", "desc": "Delete Confirm" },
			"disabled": { "field":"data", "eq": "1" }
		}`),

		"sugar-disabled-equal": []byte(`{
			"title": "Delete",
			"icon": "icon-trash-2",
			"style": "danger",
			"hide": ["view", "add"],
			"action": {
				"Actions.test.back": {  "pathname": "/x/Table/env" }
			},
			"confirm": { "title": "Tips", "desc": "Delete Confirm" },
			"disabled": {"field":"data", "equal": "1" }
		}`),
	}
}
