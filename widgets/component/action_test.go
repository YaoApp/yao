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
	assert.Equal(t, "6212f23054ddacf15e7a7c7354c6a5a6", action.ID)

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
	assert.Equal(t, "85f029739d324bcc3185c419000739e3", action.ID)

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
	assert.Equal(t, "0e1d99723154fbbeea64833be4151ada", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-string"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 1)
	assert.Equal(t, "Actions.test.back", action.Action[0]["name"])
	assert.Equal(t, "Actions.test.back", action.Action[0]["type"])
	assert.Equal(t, "9c8e6e7281ef5df76c2ef84c85308884", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-map"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 1)
	assert.Equal(t, "Form.delete", action.Action[0]["name"])
	assert.Equal(t, "Form.delete", action.Action[0]["type"])
	assert.Equal(t, "d5d48c687da8afd46145cab1f5825523", action.ID)

	action = ActionDSL{}
	err = jsoniter.Unmarshal(data["sugar-map-custom"], &action)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, action.Action, 1)
	assert.Equal(t, "Actions.test.back", action.Action[0]["name"])
	assert.Equal(t, "Actions.test.back", action.Action[0]["type"])
	assert.Equal(t, "99434a339accfe83ff38b74358e0b3af", action.ID)

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
	}
}
